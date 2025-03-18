package token_transfer

import (
	"fmt"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
)

type EventError struct {
	Message string
}

func (e *EventError) Error() string {
	return e.Message
}

func NewEventError(message string) *EventError {
	return &EventError{
		Message: message,
	}
}

var (
	xlmAsset                        = xdr.MustNewNativeAsset()
	xlmProtoAsset                   = assetProto.NewNativeAsset()
	ErrNoLiquidityPoolEntryFound    = errors.New("no liquidity pool entry found in operation changes")
	ErrNoClaimableBalanceEntryFound = errors.New("no claimable balance entry found in operation changes")
	abs64                           = func(a xdr.Int64) xdr.Int64 {
		if a < 0 {
			return -a
		}
		return a
	}
)

type TokenTransferProcessor struct {
	networkPassphrase string
}

func NewTokenTransferProcessor(networkPassphrase string) *TokenTransferProcessor {
	return &TokenTransferProcessor{
		networkPassphrase: networkPassphrase,
	}
}

// ProcessTokenTransferEventsFromLedger processes token transfer events for all transactions in a given ledger.
// This function operates at the ledger level, iterating over all transactions in the ledger.
// it calls ProcessTokenTransferEventsFromTransaction to process token transfer events from each transaction within the ledger.
func (ttp *TokenTransferProcessor) ProcessTokenTransferEventsFromLedger(lcm xdr.LedgerCloseMeta) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	txReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(ttp.networkPassphrase, lcm)
	if err != nil {
		return nil, errors.Wrap(err, "error creating transaction reader")
	}

	for {
		var tx ingest.LedgerTransaction
		var txEvents []*TokenTransferEvent
		tx, err = txReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "error reading transaction")
		}
		txEvents, err = ttp.ProcessTokenTransferEventsFromTransaction(tx)
		if err != nil {
			return nil, errors.Wrap(err, "error processing token transfer events from transaction")
		}
		events = append(events, txEvents...)
	}
	return events, nil
}

// ProcessTokenTransferEventsFromTransaction processes token transfer events for all operations within a given transaction.
//
//	First, it generates a FeeEvent for the transaction
//	If the transaction was successful, it processes all operations in the transaction by calling ProcessTokenTransferEventsFromOperationAndOperationResult for each operation in the transaction.
//
// If the transaction is unsuccessful, it only generates events for transaction fees.
func (ttp *TokenTransferProcessor) ProcessTokenTransferEventsFromTransaction(tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	feeEvents, err := ttp.generateFeeEvent(tx)
	if err != nil {
		return nil, errors.Wrap(err, "error generating fee event")
	}
	events = append(events, feeEvents...)

	// Ensure we only process operations if the transaction was successful
	if !tx.Result.Successful() {
		return events, nil
	}

	operations := tx.Envelope.Operations()
	operationResults, _ := tx.Result.OperationResults()
	for i := range operations {
		op := operations[i]
		opResult := operationResults[i]

		// Process the operation and collect events
		opEvents, err := ttp.ProcessTokenTransferEventsFromOperationAndOperationResult(tx, uint32(i), op, opResult)
		if err != nil {
			return nil,
				errors.Wrapf(err, "error processing token transfer events from operation, index: %d,  %s", i, op.Body.Type.String())
		}

		events = append(events, opEvents...)
	}

	return events, nil
}

// ProcessTokenTransferEventsFromOperationAndOperationResult processes token transfer events for a given operation within a transaction.
// It operates at the operation level, analyzing the operation type and generating corresponding token transfer events.
// If the operation is successful, it processes the event based on the operation type (e.g., payment, account creation, etc.).
// It handles various operation types like payments, account merges, trust line modifications, and more.
// There is a separate private function to derive events for each classic operation.
// It is implicitly assumed that the operation is successful, and thus will contribute towards generating events.
// which is why we don't check for the success code in the OperationResult
func (ttp *TokenTransferProcessor) ProcessTokenTransferEventsFromOperationAndOperationResult(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, opResult xdr.OperationResult) ([]*TokenTransferEvent, error) {
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		return ttp.accountCreateEvents(tx, opIndex, op)
	case xdr.OperationTypeAccountMerge:
		return ttp.mergeAccountEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePayment:
		return ttp.paymentEvents(tx, opIndex, op)
	case xdr.OperationTypeCreateClaimableBalance:
		return ttp.createClaimableBalanceEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeClaimClaimableBalance:
		return ttp.claimClaimableBalanceEvents(tx, opIndex, op)
	case xdr.OperationTypeClawback:
		return ttp.clawbackEvents(tx, opIndex, op)
	case xdr.OperationTypeClawbackClaimableBalance:
		return ttp.clawbackClaimableBalanceEvents(tx, opIndex, op)
	case xdr.OperationTypeAllowTrust:
		return ttp.allowTrustEvents(tx, opIndex, op)
	case xdr.OperationTypeSetTrustLineFlags:
		return ttp.setTrustLineFlagsEvents(tx, opIndex, op)
	case xdr.OperationTypeLiquidityPoolDeposit:
		return ttp.liquidityPoolDepositEvents(tx, opIndex, op)
	case xdr.OperationTypeLiquidityPoolWithdraw:
		return ttp.liquidityPoolWithdrawEvents(tx, opIndex, op)
	case xdr.OperationTypeManageBuyOffer:
		return ttp.manageBuyOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeManageSellOffer:
		return ttp.manageSellOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeCreatePassiveSellOffer:
		return ttp.createPassiveSellOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePathPaymentStrictSend:
		return ttp.pathPaymentStrictSendEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePathPaymentStrictReceive:
		return ttp.pathPaymentStrictReceiveEvents(tx, opIndex, op, opResult)
	default:
		return nil, nil
	}
}

type addressWrapper struct {
	account            *xdr.MuxedAccount
	liquidityPoolId    *xdr.PoolId
	claimableBalanceId *xdr.ClaimableBalanceId
}

/*
Depending on the asset - if src or dest account == issuer of asset, then mint/burn event, else transfer event
All operation related functions will call this function instead of directly calling the underlying proto functions to generate events
The only exception to this is clawbackOperation and claimableClawbackOperation.
Those 2 will call the underlying proto function for clawback
*/
func (ttp *TokenTransferProcessor) mintOrBurnOrTransferEvent(tx ingest.LedgerTransaction, opIndex *uint32, asset xdr.Asset, from addressWrapper, to addressWrapper, amt string) (*TokenTransferEvent, error) {
	var fromAddress, toAddress string
	// no need to have a separate flag for transferEvent. if neither burn nor mint, then it is regular transfer
	var isMintEvent, isBurnEvent bool

	assetIssuerAccountId, _ := asset.GetIssuerAccountId()

	// Checking 'from' address
	if from.account != nil {
		fromAddress = protoAddressFromAccount(*from.account)
		if !asset.IsNative() && assetIssuerAccountId.Equals(from.account.ToAccountId()) {
			isMintEvent = true
		}
	} else if from.liquidityPoolId != nil {
		fromAddress = protoAddressFromLpHash(*from.liquidityPoolId)
	} else if from.claimableBalanceId != nil {
		fromAddress = protoAddressFromClaimableBalanceId(*from.claimableBalanceId)
	}

	// Checking 'to' address
	if to.account != nil {
		toAddress = protoAddressFromAccount(*to.account)
		if !asset.IsNative() && assetIssuerAccountId.Equals(to.account.ToAccountId()) {
			isBurnEvent = true
		}
	} else if to.liquidityPoolId != nil {
		toAddress = protoAddressFromLpHash(*to.liquidityPoolId)
	} else if to.claimableBalanceId != nil {
		toAddress = protoAddressFromClaimableBalanceId(*to.claimableBalanceId)
	}

	protoAsset := assetProto.NewProtoAsset(asset)
	meta := ttp.generateEventMeta(tx, opIndex, asset)

	// Check for Mint Event
	if isMintEvent {
		if toAddress == "" {
			return nil, NewEventError("mint event error: to address is nil")
		}
		return NewMintEvent(meta, toAddress, amt, protoAsset), nil
	}

	// Check for Burn Event
	if isBurnEvent {
		if fromAddress == "" {
			return nil, NewEventError("burn event error: from address is nil")
		}
		return NewBurnEvent(meta, fromAddress, amt, protoAsset), nil
	}

	// If you are here, then it's a transfer event
	if toAddress == "" {
		return nil, NewEventError("transfer event error: to address is nil")
	}
	if fromAddress == "" {
		return nil, NewEventError("transfer event error: from address is nil")
	}

	// Create transfer event
	return NewTransferEvent(meta, fromAddress, toAddress, amt, protoAsset), nil
}

func (ttp *TokenTransferProcessor) generateEventMeta(tx ingest.LedgerTransaction, opIndex *uint32, asset xdr.Asset) *EventMeta {
	// Update the meta to always have contractId of the asset
	contractId, err := asset.ContractID(ttp.networkPassphrase)
	if err != nil {
		panic(errors.Wrapf(err, "Unable to generate ContractId from Asset:%v", asset.StringCanonical()))
	}
	contractAddress := strkey.MustEncode(strkey.VersionByteContract, contractId[:])
	return NewEventMetaFromTx(tx, opIndex, contractAddress)
}

func (ttp *TokenTransferProcessor) generateFeeEvent(tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	/*
		For a feeBump transaction, this will be the outer transaction.
		FeeAccount() gives the proper "muxed" account that paid the fees.
		And we want the "muxed" Account, so that it can be passed directly to protoAddressFromAccount
	*/
	feeAccount := tx.FeeAccount()
	// FeeCharged() takes care of a bug in an intermediate protocol release. So using that
	feeAmt, ok := tx.FeeCharged()
	if !ok {
		return nil, errors.New("error getting fee amount from transaction")
	}

	meta := ttp.generateEventMeta(tx, nil, xlmAsset)
	event := NewFeeEvent(meta, protoAddressFromAccount(feeAccount), amount.String64Raw(xdr.Int64(feeAmt)), xlmProtoAsset)
	return []*TokenTransferEvent{event}, nil
}

// Function stubs
func (ttp *TokenTransferProcessor) accountCreateEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	createAccountOp := op.Body.MustCreateAccountOp()
	destAcc, amt := createAccountOp.Destination.ToMuxedAccount(), amount.String64Raw(createAccountOp.StartingBalance)
	from := addressWrapper{account: &opSrcAcc}
	to := addressWrapper{account: &destAcc}
	event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, xlmAsset, from, to, amt)
	if err != nil {
		return nil, err
	}

	return []*TokenTransferEvent{event}, nil // Just one event will be generated
}

func (ttp *TokenTransferProcessor) mergeAccountEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	res := result.Tr.MustAccountMergeResult()
	// If there is no transfer of XLM from source account to destination (i.e. src account is empty), then no need to generate a transfer event
	if res.SourceAccountBalance == nil {
		return nil, nil
	}
	opSrcAcc := operationSourceAccount(tx, op)
	destAcc := op.Body.MustDestination()
	amt := amount.String64Raw(*res.SourceAccountBalance)
	event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, xlmAsset, addressWrapper{account: &opSrcAcc}, addressWrapper{account: &destAcc}, amt)
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil // Just one event will be generated
}

func (ttp *TokenTransferProcessor) paymentEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	paymentOp := op.Body.MustPaymentOp()
	opSrcAcc := operationSourceAccount(tx, op)
	destAcc := paymentOp.Destination
	amt := amount.String64Raw(paymentOp.Amount)

	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{account: &destAcc}
	event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, paymentOp.Asset, from, to, amt)
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil
}

func (ttp *TokenTransferProcessor) createClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	createCbOp := op.Body.MustCreateClaimableBalanceOp()
	createCbResult := result.Tr.MustCreateClaimableBalanceResult()
	opSrcAcc := operationSourceAccount(tx, op)
	claimableBalanceId := createCbResult.MustBalanceId()

	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{claimableBalanceId: &claimableBalanceId}
	event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, createCbOp.Asset, from, to, amount.String64Raw(createCbOp.Amount))
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil
}

func possibleClaimableBalanceIdsFromRevocation(lpEntry liquidityPoolEntryDelta, tx ingest.LedgerTransaction, txSrcAccount xdr.AccountId, opIndex uint32) ([]xdr.ClaimableBalanceId, error) {
	var possibleClaimableBalanceIds []xdr.ClaimableBalanceId
	lpId := lpEntry.liquidityPoolId
	seqNum := xdr.SequenceNumber(tx.Envelope.SeqNum())

	for _, asset := range []xdr.Asset{lpEntry.assetA, lpEntry.assetB} {
		cbId, err := ClaimableBalanceIdFromRevocation(lpId, asset, seqNum, txSrcAccount, opIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate claimable balance id from LiquidityPoolId: %v, for asset: %v", lpIdToStrkey(lpId), asset.String())
		}
		possibleClaimableBalanceIds = append(possibleClaimableBalanceIds, cbId)
	}
	return possibleClaimableBalanceIds, nil
}

func (ttp *TokenTransferProcessor) claimClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	claimCbOp := op.Body.MustClaimClaimableBalanceOp()
	cbId := claimCbOp.BalanceId

	// After this operation, the CB will be deleted.
	cbEntries, err := getClaimableBalanceEntriesFromOperationChanges(xdr.LedgerEntryChangeTypeLedgerEntryRemoved, tx, opIndex)
	if err != nil {
		return nil, err
	} else if len(cbEntries) == 0 {
		return nil, ErrNoClaimableBalanceEntryFound
	} else if len(cbEntries) != 1 {
		return nil, fmt.Errorf("more than one claimable entry found for operation: %s", op.Body.Type.String())
	}

	opSrcAcc := operationSourceAccount(tx, op)
	cb := cbEntries[0]

	// This is one case where the order is reversed. Money flows from CBid --> sourceAccount of this claimCb operation
	from, to := addressWrapper{claimableBalanceId: &cbId}, addressWrapper{account: &opSrcAcc}
	event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, cb.Asset, from, to, amount.String64Raw(cb.Amount))
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil
}

func (ttp *TokenTransferProcessor) clawbackEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	clawbackOp := op.Body.MustClawbackOp()
	meta := ttp.generateEventMeta(tx, &opIndex, clawbackOp.Asset)

	// fromAddress is NOT the operationSourceAccount.
	// It is the account specified in the operation from whom you want money to be clawed back
	from := protoAddressFromAccount(clawbackOp.From)
	event := NewClawbackEvent(meta, from, amount.String64Raw(clawbackOp.Amount), assetProto.NewProtoAsset(clawbackOp.Asset))
	return []*TokenTransferEvent{event}, nil
}

func (ttp *TokenTransferProcessor) clawbackClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	clawbackCbOp := op.Body.MustClawbackClaimableBalanceOp()
	cbId := clawbackCbOp.BalanceId

	// After this operation, the CB will be deleted.
	cbEntries, err := getClaimableBalanceEntriesFromOperationChanges(xdr.LedgerEntryChangeTypeLedgerEntryRemoved, tx, opIndex)
	if err != nil {
		return nil, err
	} else if len(cbEntries) == 0 {
		return nil, ErrNoClaimableBalanceEntryFound
	} else if len(cbEntries) != 1 {
		return nil, fmt.Errorf("more than one claimable entry found for operation: %s", op.Body.Type.String())
	}

	cb := cbEntries[0]
	meta := ttp.generateEventMeta(tx, &opIndex, cb.Asset)
	// Money is clawed back from the claimableBalanceId
	event := NewClawbackEvent(meta, protoAddressFromClaimableBalanceId(cbId), amount.String64Raw(cb.Amount), assetProto.NewProtoAsset(cb.Asset))
	return []*TokenTransferEvent{event}, nil
}

func (ttp *TokenTransferProcessor) allowTrustEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	// ?? Should I be checking for generation of liquidity pools and CBs iff the flag is set to false?
	// isAuthRevoked := op.Body.MustAllowTrustOp().Authorize == 0
	return ttp.generateEventsForRevokedTrustlines(tx, opIndex)
}

func (ttp *TokenTransferProcessor) setTrustLineFlagsEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	// ?? Should I be checking for generation of liquidity pools and CBs iff the flag is set to false?
	// isAuthRevoked := op.Body.MustSetTrustLineFlagsOp().ClearFlags != 0
	return ttp.generateEventsForRevokedTrustlines(tx, opIndex)
}

func (ttp *TokenTransferProcessor) generateEventsForRevokedTrustlines(tx ingest.LedgerTransaction, opIndex uint32) ([]*TokenTransferEvent, error) {
	// IF this operation is used to revoke authorization from a trustline that deposited into a liquidity pool,
	// then UPTO 2 claimable balances will be created for the withdrawn assets (See CAP-0038 for more info) PER each impacted Liquidity Pool

	// Go through the operation changes and find the LiquidityPools that were impacted
	impactedLiquidityPools, err := getImpactedLiquidityPoolEntriesFromOperation(tx, opIndex)
	if err != nil {
		return nil, err
	}

	// The operation didn't cause revocation of the trustor account's pool shares in any liquidity pools.
	// So no events generated at all
	if len(impactedLiquidityPools) == 0 {
		return nil, nil
	}

	createdClaimableBalances, err := getClaimableBalanceEntriesFromOperationChanges(xdr.LedgerEntryChangeTypeLedgerEntryCreated, tx, opIndex)
	if err != nil {
		return nil, err
	}

	// There were no claimable balances created, even though there were some liquidity pools that showed up.
	// This can happen, if all the liquidity pools impacted were empty in the first place,
	// i.e. they didn't have any pool shares OR reserves of asset A and B
	// So there is no transfer of money, so no events generated
	if len(createdClaimableBalances) == 0 {
		return nil, nil
	}

	/*
		MAGIC happens here.
		Based on the logic in CAP-38 (https://github.com/stellar/stellar-protocol/blob/master/core/cap-0038.md#settrustlineflagsop-and-allowtrustop),
		There is no concrete way to tie which claimable balances(either 1 or 2) were created in response to which liquidity pool.
		So, it is difficult to generate the `from`, `to` in the transfer event.
		So, we need to re-implement the code logic to derive claimableBalanceIds from liquidity pool
		and then do additional logic to compare it with the claimableBalances that were actually created (i,e the ones that show up in the operation changes)

		It would be nice, if somehow, core could trickle this information up in the ledger entry changes for the claimable balances created - something like createdBy in the extension field.
	*/

	createdClaimableBalancesById := map[xdr.Hash]xdr.ClaimableBalanceEntry{}
	for _, cb := range createdClaimableBalances {
		createdClaimableBalancesById[cb.BalanceId.MustV0()] = cb
	}

	var events []*TokenTransferEvent

	for _, lp := range impactedLiquidityPools {
		// This `possibleClaimableBalanceIds` slice will have exactly 2 entries - one for each asset in the LP
		// The logic in CAP-38 dictates that the transactionAccount is what is passed to generate the ClaimableBalanceId
		// and NOT the operation source account.
		possibleClaimableBalanceIds, err := possibleClaimableBalanceIdsFromRevocation(lp, tx, tx.Envelope.SourceAccount().ToAccountId(), opIndex)
		if err != nil {
			return nil, err
		}

		var cbsCreatedByThisLp []xdr.ClaimableBalanceEntry
		for _, id := range possibleClaimableBalanceIds {
			if cb, ok := createdClaimableBalancesById[id.MustV0()]; ok {
				cbsCreatedByThisLp = append(cbsCreatedByThisLp, cb)
			}
		}

		if len(cbsCreatedByThisLp) == 0 {
			continue // there are no events since there are no claimable balances that were created by this LP
		} else if len(cbsCreatedByThisLp) == 1 {

			// There was exactly 1 claimable balance created by this LP, whereas, normally, you'd expect 2.
			// which means that the other asset was sent back to the issuer.
			// This is the case where the trustor (account in operation whose trustline is being revoked) is the issuer.
			// i.e. that asset's amount was burned, which is why no LP was created in the first place.
			//  issue 2 events:
			//		- transfer event -- from: LPHash, to:Cb that was created, asset: Asset in cbEntry, amt: Amt in CB
			//		- burn event for the asset for which no CB was created -- from: LPHash, asset: burnedAsset from LP, amt: Amt of burnedAsset in the LP

			/*
				For e.g. suppose an account - ethIssuerAccount, deposits to USDC-ETH liquidity pool
				Now, a setTrustlineFlags operation is issued by the USDC-Issuer account to revoke ethIssuerAccount's USDC trustline.
				As a side effect of this trustline revocation, there is a removal of ethIssuerAcount's stake from the USDB-ETH liquidity pool.
				In this case, 1 Claimable balance for USDC will be created, with claimantAccount = trustor i.e.. ethIssuerAccount
				No Claimable balance for ETH will be created. it will simply be burned.
			*/
			assetInCb := cbsCreatedByThisLp[0].Asset

			// The asset that needs to be burned is the one that is the OPPOSITE of the asset in the CB, so find that in the LP
			var burnedAsset xdr.Asset
			var burnedAmount xdr.Int64
			if assetInCb == lp.assetA {
				burnedAsset = lp.assetB
				burnedAmount = lp.amountChangeForAssetB
			} else {
				burnedAsset = lp.assetA
				burnedAmount = lp.amountChangeForAssetA
			}

			from := lp.liquidityPoolId

			transferEvent, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, assetInCb,
				addressWrapper{liquidityPoolId: &from},
				addressWrapper{claimableBalanceId: &cbsCreatedByThisLp[0].BalanceId},
				amount.String64Raw(cbsCreatedByThisLp[0].Amount))
			if err != nil {
				return nil, err
			}

			burnMeta := ttp.generateEventMeta(tx, &opIndex, burnedAsset)
			burnEvent := NewBurnEvent(burnMeta, protoAddressFromLpHash(from), amount.String64Raw(burnedAmount), assetProto.NewProtoAsset(burnedAsset))
			events = append(events, transferEvent, burnEvent)

		} else if len(cbsCreatedByThisLp) == 2 {
			// Easy case - This LP created 2 claimable balances - one for each of the assets in the LP, to be sent to the account whose trustline was revoked.
			// so generate 2 transfer events
			from := lp.liquidityPoolId
			asset1, asset2 := cbsCreatedByThisLp[0].Asset, cbsCreatedByThisLp[1].Asset
			to1, to2 := cbsCreatedByThisLp[0].BalanceId, cbsCreatedByThisLp[1].BalanceId
			amt1, amt2 := amount.String64Raw(cbsCreatedByThisLp[0].Amount), amount.String64Raw(cbsCreatedByThisLp[1].Amount)

			ev1, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, asset1,
				addressWrapper{liquidityPoolId: &from},
				addressWrapper{claimableBalanceId: &to1},
				amt1)
			if err != nil {
				return nil, err
			}

			ev2, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, asset2,
				addressWrapper{liquidityPoolId: &from},
				addressWrapper{claimableBalanceId: &to2},
				amt2)
			if err != nil {
				return nil, err
			}

			events = append(events, ev1, ev2)

		} else {
			return nil,
				fmt.Errorf("more than two claimable balances created from LP: %v. This shouldnt be possible",
					lpIdToStrkey(lp.liquidityPoolId))
		}
	}

	return events, nil
}

type liquidityPoolEntryDelta struct {
	liquidityPoolId       xdr.PoolId
	assetA                xdr.Asset
	assetB                xdr.Asset
	amountChangeForAssetA xdr.Int64
	amountChangeForAssetB xdr.Int64
}

func getImpactedLiquidityPoolEntriesFromOperation(tx ingest.LedgerTransaction, opIndex uint32) ([]liquidityPoolEntryDelta, error) {
	changes, err := tx.GetOperationChanges(opIndex)
	if err != nil {
		return nil, err
	}

	var entries []liquidityPoolEntryDelta
	for _, c := range changes {
		if c.Type != xdr.LedgerEntryTypeLiquidityPool {
			continue
		}
		var lp *xdr.LiquidityPoolEntry
		var entry liquidityPoolEntryDelta

		var preA, preB xdr.Int64
		if c.Pre != nil {
			lp = c.Pre.Data.LiquidityPool
			entry.liquidityPoolId = lp.LiquidityPoolId
			cp := lp.Body.ConstantProduct
			entry.assetA, entry.assetB = cp.Params.AssetA, cp.Params.AssetB
			preA, preB = cp.ReserveA, cp.ReserveB
		}

		var postA, postB xdr.Int64
		if c.Post != nil {
			lp = c.Post.Data.LiquidityPool
			entry.liquidityPoolId = lp.LiquidityPoolId
			cp := lp.Body.ConstantProduct
			entry.assetA, entry.assetB = cp.Params.AssetA, cp.Params.AssetB
			postA, postB = cp.ReserveA, cp.ReserveB
		}

		entry.amountChangeForAssetA = abs64(postA - preA)
		entry.amountChangeForAssetB = abs64(postB - preB)
		entries = append(entries, entry)
	}

	return entries, nil
}

func (ttp *TokenTransferProcessor) liquidityPoolDepositEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	lpDeltas, e := getImpactedLiquidityPoolEntriesFromOperation(tx, opIndex)
	if e != nil {
		return nil, e
	} else if len(lpDeltas) == 0 {
		return nil, ErrNoLiquidityPoolEntryFound
	} else if len(lpDeltas) != 1 {
		return nil, fmt.Errorf("more than one Liquidiy pool entry found for operation: %s", op.Body.Type.String())
	}

	delta := lpDeltas[0]
	lpId := delta.liquidityPoolId
	assetA, assetB := delta.assetA, delta.assetB
	amtA, amtB := delta.amountChangeForAssetA, delta.amountChangeForAssetB
	if amtA <= 0 {
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be zero or negative in LiquidityPool: %v", amtA, assetA.String(), lpIdToStrkey(lpId))
	}
	if amtB < 0 {
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be zero or negative in LiquidityPool: %v", amtB, assetB.String(), lpIdToStrkey(lpId))
	}

	opSrcAcc := operationSourceAccount(tx, op)
	// From = operation source account, to = LP
	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{liquidityPoolId: &delta.liquidityPoolId}

	var events []*TokenTransferEvent
	// I am not sure if it is possible for amtA or amtB to be ever 0, for e,g when LpDeposit updates the amount for just 1 asset in an already existing LP
	// So, out of abundance of caution, I will generate the event only if the amounts are greater than 0
	if amtA >= 0 {
		event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, assetA, from, to, amount.String64Raw(amtA))
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if amtB >= 0 {
		event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, assetB, from, to, amount.String64Raw(amtB))
		if err != nil {
			return nil, err
		}
		events = append(events, event)

	}
	return events, nil
}

func (ttp *TokenTransferProcessor) liquidityPoolWithdrawEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	lpDeltas, e := getImpactedLiquidityPoolEntriesFromOperation(tx, opIndex)
	if e != nil {
		return nil, e
	} else if len(lpDeltas) == 0 {
		return nil, ErrNoLiquidityPoolEntryFound
	} else if len(lpDeltas) != 1 {
		return nil, fmt.Errorf("more than one Liquidiy pool entry found for operation: %s", op.Body.Type.String())
	}

	delta := lpDeltas[0]
	lpId := delta.liquidityPoolId
	assetA, assetB := delta.assetA, delta.assetB
	amtA, amtB := delta.amountChangeForAssetA, delta.amountChangeForAssetB
	if amtA <= 0 {
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be zero or negative in LiquidityPool: %v", amtA, assetA.String(), lpIdToStrkey(lpId))
	}
	if amtB <= 0 {
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be zero or negative in LiquidityPool: %v", amtB, assetB.String(), lpIdToStrkey(lpId))
	}

	opSrcAcc := operationSourceAccount(tx, op)
	// Opposite of LP Deposit. from = LP, to = operation source account
	from, to := addressWrapper{liquidityPoolId: &delta.liquidityPoolId}, addressWrapper{account: &opSrcAcc}

	var events []*TokenTransferEvent
	// I am not sure if it is possible for amtA or amtB to be ever 0, for e,g when LpDeposit updates the amount for just 1 asset in an already existing LP
	// So, out of abundance of caution, I will generate the event only if the amounts are greater than 0
	if amtA > 0 {
		event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, assetA, from, to, amount.String64Raw(amtA))
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if amtB > 0 {
		event, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, assetB, from, to, amount.String64Raw(amtB))
		if err != nil {
			return nil, err
		}
		events = append(events, event)

	}
	return events, nil
}

func (ttp *TokenTransferProcessor) generateEventsFromClaimAtoms(tx ingest.LedgerTransaction, opIndex uint32, opSrcAcc xdr.MuxedAccount, claims []xdr.ClaimAtom) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	operationSourceAddressWrapper := addressWrapper{account: &opSrcAcc}
	var sellerAddressWrapper addressWrapper

	for _, claim := range claims {
		if claim.Type == xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool {
			lpId := claim.MustLiquidityPool().LiquidityPoolId
			sellerAddressWrapper = addressWrapper{liquidityPoolId: &lpId}
		} else {
			sellerId := claim.SellerId()
			sellerAccount := sellerId.ToMuxedAccount()
			sellerAddressWrapper = addressWrapper{account: &sellerAccount}

		}

		ev1, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, claim.AssetSold(), sellerAddressWrapper, operationSourceAddressWrapper, amount.String64Raw(claim.AmountSold()))
		if err != nil {
			return nil, err
		}
		ev2, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, claim.AssetBought(), operationSourceAddressWrapper, sellerAddressWrapper, amount.String64Raw(claim.AmountBought()))
		if err != nil {
			return nil, err
		}

		// 2 events generated per trade
		events = append(events, ev1, ev2)
	}
	return events, nil
}

func (ttp *TokenTransferProcessor) manageBuyOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	offersClaimed := result.Tr.MustManageBuyOfferResult().Success.OffersClaimed
	if len(offersClaimed) == 0 {
		return nil, nil
	}
	return ttp.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, offersClaimed)
}

func (ttp *TokenTransferProcessor) manageSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	offersClaimed := result.Tr.MustManageSellOfferResult().Success.OffersClaimed
	if len(offersClaimed) == 0 {
		return nil, nil
	}
	return ttp.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, offersClaimed)
}

// EXACTLY SAME as manageSellOfferEvents
func (ttp *TokenTransferProcessor) createPassiveSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	return ttp.manageSellOfferEvents(tx, opIndex, op, result)
}

func (ttp *TokenTransferProcessor) pathPaymentStrictSendEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	strictSendOp := op.Body.MustPathPaymentStrictSendOp()
	strictSendResult := result.Tr.MustPathPaymentStrictSendResult()

	var events []*TokenTransferEvent
	ev, err := ttp.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, strictSendResult.MustSuccess().Offers)
	if err != nil {
		return nil, err
	}
	events = append(events, ev...)

	// Generate one final event indicating the amount that the destination received in terms of destination asset
	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{account: &strictSendOp.Destination}
	finalEvent, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, strictSendOp.DestAsset, from, to, amount.String64Raw(strictSendResult.DestAmount()))
	if err != nil {
		return nil, err
	}
	events = append(events, finalEvent)
	return events, nil
}

func (ttp *TokenTransferProcessor) pathPaymentStrictReceiveEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	strictReceiveOp := op.Body.MustPathPaymentStrictReceiveOp()
	strictReceiveResult := result.Tr.MustPathPaymentStrictReceiveResult()

	var events []*TokenTransferEvent
	ev, err := ttp.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, strictReceiveResult.MustSuccess().Offers)
	if err != nil {
		return nil, err
	}
	events = append(events, ev...)

	// Generate one final event indicating the amount that the destination received in terms of destination asset
	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{account: &strictReceiveOp.Destination}
	finalEvent, err := ttp.mintOrBurnOrTransferEvent(tx, &opIndex, strictReceiveOp.DestAsset, from, to, amount.String64Raw(strictReceiveOp.DestAmount))
	if err != nil {
		return nil, err
	}
	events = append(events, finalEvent)
	return events, nil
}
