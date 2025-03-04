package token_transfer

import (
	"fmt"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	addressProto "github.com/stellar/go/ingest/address"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/support/converters"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
)

var (
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

func ProcessTokenTransferEventsFromLedger(lcm xdr.LedgerCloseMeta, networkPassPhrase string) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	txReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassPhrase, lcm)
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
		txEvents, err = ProcessTokenTransferEventsFromTransaction(tx, networkPassPhrase)
		if err != nil {
			return nil, errors.Wrap(err, "error processing token transfer events from transaction")
		}
		events = append(events, txEvents...)
	}
	return events, nil
}

func ProcessTokenTransferEventsFromTransaction(tx ingest.LedgerTransaction, networkPassPhrase string) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	feeEvents, err := generateFeeEvent(tx)
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
		opEvents, err := ProcessTokenTransferEventsFromOperationAndOperationResult(tx, uint32(i), op, opResult, networkPassPhrase)
		if err != nil {
			return nil,
				errors.Wrapf(err, "error processing token transfer events from operation, index: %d,  %s", i, op.Body.Type.String())
		}

		events = append(events, opEvents...)
	}

	return events, nil
}

// ProcessTokenTransferEventsFromOperationAndOperationResult
// There is a separate private function to derive events for each classic operation.
// It is implicitly assumed that the operation is successful, and thus will contribute towards generating events.
// which is why we dont check for the code in the OperationResult
func ProcessTokenTransferEventsFromOperationAndOperationResult(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, opResult xdr.OperationResult, networkPassPhrase string) ([]*TokenTransferEvent, error) {
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		return accountCreateEvents(tx, opIndex, op)
	case xdr.OperationTypeAccountMerge:
		return mergeAccountEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePayment:
		return paymentEvents(tx, opIndex, op)
	case xdr.OperationTypeCreateClaimableBalance:
		return createClaimableBalanceEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeClaimClaimableBalance:
		return claimClaimableBalanceEvents(tx, opIndex, op)
	case xdr.OperationTypeClawback:
		return clawbackEvents(tx, opIndex, op)
	case xdr.OperationTypeClawbackClaimableBalance:
		return clawbackClaimableBalanceEvents(tx, opIndex, op)
	case xdr.OperationTypeAllowTrust:
		return allowTrustEvents(tx, opIndex, op)
	case xdr.OperationTypeSetTrustLineFlags:
		return setTrustLineFlagsEvents(tx, opIndex, op)
	case xdr.OperationTypeLiquidityPoolDeposit:
		return liquidityPoolDepositEvents(tx, opIndex, op)
	case xdr.OperationTypeLiquidityPoolWithdraw:
		return liquidityPoolWithdrawEvents(tx, opIndex, op)
	case xdr.OperationTypeManageBuyOffer:
		return manageBuyOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeManageSellOffer:
		return manageSellOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeCreatePassiveSellOffer:
		return createPassiveSellOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePathPaymentStrictSend:
		return pathPaymentStrictSendEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePathPaymentStrictReceive:
		return pathPaymentStrictReceiveEvents(tx, opIndex, op, opResult)
	default:
		return nil, nil
	}
}

func generateFeeEvent(tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	/*
		For a feeBump transaction, this will be the outer transaction.
		FeeAccount() gives the proper "muxed" account that paid the fees.
		And we want the "muxed" Account, so that it can be passed directly to protoAddressFromAccount
	*/
	feeAccount := tx.FeeAccount()
	// FeeCharged() takes care of a bug in an intermediate protocol release. So using that
	feeAmt, _ := tx.FeeCharged()

	event := NewFeeEvent(tx.Ledger.LedgerSequence(), tx.Ledger.ClosedAt(), tx.Hash.HexString(), protoAddressFromAccount(feeAccount), amount.String(xdr.Int64(feeAmt)))
	return []*TokenTransferEvent{event}, nil
}

// Function stubs
func accountCreateEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	createAccountOp := op.Body.MustCreateAccountOp()
	destAcc, amt := createAccountOp.Destination.ToMuxedAccount(), amount.String(createAccountOp.StartingBalance)
	meta := NewEventMeta(tx, &opIndex, nil)
	event := NewTransferEvent(meta, protoAddressFromAccount(opSrcAcc), protoAddressFromAccount(destAcc), amt, xlmProtoAsset)
	return []*TokenTransferEvent{event}, nil // Just one event will be generated
}

func mergeAccountEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	res := result.Tr.MustAccountMergeResult()
	// If there is no transfer of XLM from source account to destination (i.e src account is empty), then no need to generate a transfer event
	if res.SourceAccountBalance == nil {
		return nil, nil
	}
	opSrcAcc := operationSourceAccount(tx, op)
	destAcc := op.Body.MustDestination()
	amt := amount.String(*res.SourceAccountBalance)
	meta := NewEventMeta(tx, &opIndex, nil)
	event := NewTransferEvent(meta, protoAddressFromAccount(opSrcAcc), protoAddressFromAccount(destAcc), amt, xlmProtoAsset)
	return []*TokenTransferEvent{event}, nil // Just one event will be generated
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
func mintOrBurnOrTransferEvent(asset xdr.Asset, from addressWrapper, to addressWrapper, amt string, meta *EventMeta) *TokenTransferEvent {
	var fromAddress, toAddress *addressProto.Address
	// no need to have a separate flag for transferEvent. if neither burn nor mint, then it is regular transfer
	var isMintEvent, isBurnEvent bool

	assetIssuerAccountId, _ := asset.GetIssuerAccountId()

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

	if to.account != nil {
		toAddress = protoAddressFromAccount(*to.account)
		if !asset.IsNative() && assetIssuerAccountId.Equals(to.account.ToAccountId()) {
			isBurnEvent = true
		}
	} else if to.liquidityPoolId != nil {
		toAddress = protoAddressFromLpHash(*to.liquidityPoolId)
	} else if from.claimableBalanceId != nil {
		toAddress = protoAddressFromClaimableBalanceId(*to.claimableBalanceId)
	}

	protoAsset := assetProto.NewProtoAsset(asset)

	var event *TokenTransferEvent
	if isMintEvent {
		// Mint event
		event = NewMintEvent(meta, toAddress, amt, protoAsset)
	} else if isBurnEvent {
		// Burn event
		event = NewBurnEvent(meta, fromAddress, amt, protoAsset)
	} else {
		event = NewTransferEvent(meta, fromAddress, toAddress, amt, protoAsset)
	}
	return event
}

func paymentEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	paymentOp := op.Body.MustPaymentOp()
	opSrcAcc := operationSourceAccount(tx, op)
	destAcc := paymentOp.Destination
	amt := amount.String(paymentOp.Amount)
	meta := NewEventMeta(tx, &opIndex, nil)

	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{account: &destAcc}
	event := mintOrBurnOrTransferEvent(paymentOp.Asset, from, to, amt, meta)
	return []*TokenTransferEvent{event}, nil
}

func createClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	createCbOp := op.Body.MustCreateClaimableBalanceOp()
	createCbResult := result.Tr.MustCreateClaimableBalanceResult()
	opSrcAcc := operationSourceAccount(tx, op)
	meta := NewEventMeta(tx, &opIndex, nil)
	claimableBalanceId := createCbResult.MustBalanceId()

	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{claimableBalanceId: &claimableBalanceId}
	event := mintOrBurnOrTransferEvent(createCbOp.Asset, from, to, amount.String(createCbOp.Amount), meta)
	return []*TokenTransferEvent{event}, nil
}

func generateClaimableBalanceIdFromLiquidityPoolId(lpEntry liquidityPoolEntryDelta, tx ingest.LedgerTransaction, txSrcAccount xdr.AccountId, opIndex uint32) ([]xdr.ClaimableBalanceId, error) {
	var generatedClaimableBalanceIds []xdr.ClaimableBalanceId
	lpId := lpEntry.liquidityPoolId
	seqNum := xdr.SequenceNumber(tx.Envelope.SeqNum())

	for _, asset := range []xdr.Asset{lpEntry.assetA, lpEntry.assetB} {
		cbId, err := converters.ConvertLiquidityPoolIdToClaimableBalanceId(lpId, asset, seqNum, txSrcAccount, opIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate claimable balance id from LiquidityPoolId: %v, for asset: %v", lpIdToStrkey(lpId), asset.String())
		}
		generatedClaimableBalanceIds = append(generatedClaimableBalanceIds, cbId)
	}
	return generatedClaimableBalanceIds, nil
}

// This operation is used to only find CB entries that are either created or deleted, not updated
func getClaimableBalanceEntriesFromOperationChanges(changeType xdr.LedgerEntryChangeType, tx ingest.LedgerTransaction, opIndex uint32) ([]xdr.ClaimableBalanceEntry, error) {
	if !(changeType == xdr.LedgerEntryChangeTypeLedgerEntryRemoved || changeType == xdr.LedgerEntryChangeTypeLedgerEntryCreated) {
		return nil, fmt.Errorf("changeType: %v, not allowed", changeType)
	}

	changes, err := tx.GetOperationChanges(opIndex)
	if err != nil {
		return nil, err
	}

	var entries []xdr.ClaimableBalanceEntry
	/*
		This function is expected to be called only to get details of newly created claimable balance
		(for e.g as a result of allowTrust or setTrustlineFlags  operations)
		or claimable balances that are be deleted
		(for e.g due to clawback claimable balance operation)
	*/
	var cb xdr.ClaimableBalanceEntry
	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeClaimableBalance {
			continue
		}
		// Check if claimable balance entry is deleted
		//?? maybe it is not necessary to check change.Pre != nil && change.Post since that will be true for deleted entries?
		if changeType == xdr.LedgerEntryChangeTypeLedgerEntryRemoved && change.Pre != nil && change.Post == nil {
			cb = change.Pre.Data.MustClaimableBalance()
			entries = append(entries, cb)
		} else if changeType == xdr.LedgerEntryChangeTypeLedgerEntryCreated && change.Post != nil && change.Pre == nil { // check if claimable balance entry is created
			cb = change.Post.Data.MustClaimableBalance()
			entries = append(entries, cb)
		}
	}

	return entries, nil
}

func claimClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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

	meta := NewEventMeta(tx, &opIndex, nil)
	opSrcAcc := operationSourceAccount(tx, op)
	cb := cbEntries[0]

	// This is one case where the order is reversed. Money flows from CBid --> sourceAccount of this claimCb operation
	from, to := addressWrapper{claimableBalanceId: &cbId}, addressWrapper{account: &opSrcAcc}
	event := mintOrBurnOrTransferEvent(cb.Asset, from, to, amount.String(cb.Amount), meta)
	return []*TokenTransferEvent{event}, nil
}

func clawbackEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	clawbackOp := op.Body.MustClawbackOp()
	meta := NewEventMeta(tx, &opIndex, nil)

	// fromAddress is NOT the operationSourceAccount.
	// It is the account specified in the operation from whom you want money to be clawed back
	from := protoAddressFromAccount(clawbackOp.From)
	event := NewClawbackEvent(meta, from, amount.String(clawbackOp.Amount), assetProto.NewProtoAsset(clawbackOp.Asset))
	return []*TokenTransferEvent{event}, nil
}

func clawbackClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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
	meta := NewEventMeta(tx, &opIndex, nil)
	// Money is clawed back from the claimableBalanceId
	event := NewClawbackEvent(meta, protoAddressFromClaimableBalanceId(cbId), amount.String(cb.Amount), assetProto.NewProtoAsset(cb.Asset))
	return []*TokenTransferEvent{event}, nil
}

func allowTrustEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	// ?? Should I be checking for generation of liquidity pools and CBs iff the flag is set to false?
	// isAuthRevoked := op.Body.MustAllowTrustOp().Authorize == 0
	return generateEventsForRevokedTrustlines(tx, opIndex)
}

func setTrustLineFlagsEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	// ?? Should I be checking for generation of liquidity pools and CBs iff the flag is set to false?
	// isAuthRevoked := op.Body.MustSetTrustLineFlagsOp().ClearFlags != 0
	return generateEventsForRevokedTrustlines(tx, opIndex)
}

func generateEventsForRevokedTrustlines(tx ingest.LedgerTransaction, opIndex uint32) ([]*TokenTransferEvent, error) {
	// IF this operation is used to revoke authorization from a trustline that deposited into a liquidity pool,
	// then UPTO 2 claimable balances will be created for the withdrawn assets (See CAP-0038 for more info) PER each impacted Liquidity Pool

	// Go through the operation changes and find the LiquidityPools that were impacted
	impactedLiquidityPools, err := getImpactedLiquidityPoolEntriesFromOperation(tx, opIndex)
	if err != nil {
		return nil, err
	}

	// The operation didnt cause revocation of the trustor account's pool shares in any liquidity pools.
	// So no events generated at all
	if len(impactedLiquidityPools) == 0 {
		return nil, nil
	}

	impactedClaimableBalances, err := getClaimableBalanceEntriesFromOperationChanges(xdr.LedgerEntryChangeTypeLedgerEntryCreated, tx, opIndex)
	if err != nil {
		return nil, err
	}

	// There were no claimable balances created, even though there were some liquidity pools that showed up.
	// This can happen, if all the liquidity pools impacted were empty in the first place,
	// i.e they didnt have any pool shares OR reserves of asset A and B
	// So there is no transfer of money, so no events generated
	if len(impactedClaimableBalances) == 0 {
		return nil, nil
	}

	/*
		MAGIC happens here.
		Based on the logic in CAP-38 (https://github.com/stellar/stellar-protocol/blob/master/core/cap-0038.md#settrustlineflagsop-and-allowtrustop),
		There is no concrete way to tie which claimable balances(either 1 or 2) were created in response to which liquidity pool.
		So, it is difficult to generate the from, to in the transfer event.
		So, we need to re-implement the code logic to derive claimableBalanceIds from liquidity pool
		and then do additional logic to compare it with the claimableBalances that were actually created (i,e the ones that show up in the operation changes)

		It would be nice, if somehow, core could trickle this information up in the ledger entry changes for the claimable balances created - something like createdBy in the extension field.
	*/
	createdCbIdToCreatorLpId := make(map[string]string)
	for _, lp := range impactedLiquidityPools {
		// This `claimableBalancesCreated` slice will have exactly 2 entries - one for each asset in the LP
		// The logic in CAP-38 dictates that the transactionAccount is what is passed to generate the ClaimableBalanceId
		// and NOT the operation source account.
		claimableBalancesCreated, err := generateClaimableBalanceIdFromLiquidityPoolId(lp, tx, tx.Envelope.SourceAccount().ToAccountId(), opIndex)
		if err != nil {
			return nil, err
		}
		for _, cbId := range claimableBalancesCreated {
			createdCbIdToCreatorLpId[cbIdToStrkey(cbId)] = lpIdToStrkey(lp.liquidityPoolId)
		}
	}

	meta := NewEventMeta(tx, &opIndex, nil)
	var events []*TokenTransferEvent

	for _, lp := range impactedLiquidityPools {
		var cbsCreatedByThisLp []xdr.ClaimableBalanceEntry

		// Nested for loop.
		// See the  logic in CAP-38(https://github.com/stellar/stellar-protocol/blob/master/core/cap-0038.md#settrustlineflagsop-and-allowtrustop).
		// This is consistent with that
		currentLpId := lpIdToStrkey(lp.liquidityPoolId)
		for _, cbEntry := range impactedClaimableBalances {
			cbId := cbIdToStrkey(cbEntry.BalanceId)
			// Maybe the additional check for creatorLpId == currentLpId is redundant, but just in case
			if creatorLpId, found := createdCbIdToCreatorLpId[cbId]; found && creatorLpId == currentLpId {
				cbsCreatedByThisLp = append(cbsCreatedByThisLp, cbEntry)
			}
		}

		if len(cbsCreatedByThisLp) == 0 {
			continue // there are no events since there are no claimable balances that were created by this LP
		} else if len(cbsCreatedByThisLp) == 1 {

			// There was exactly 1 claimable balance created by this LP, whereas, normally, you'd expect 2.
			// which means that the other asset was sent back to the issuer.
			// This is the case where the trustor (account in operation whose trustline is being revoked) is the issuer.
			//, i.e that asset's amount was burned, which is why no LP was created in the first place.
			//  issue 2 events:
			//		- transfer event -- from: LPHash, to:Cb that was created, asset: Asset in cbEntry, amt: Amt in CB
			//		- burn event for the asset for which no CB was created -- from: LPHash, asset: burnedAsset from LP, amt: Amt of burnedAsset in the LP

			/*
				For e.g suppose an account - ethIssuerAccount, deposits to USDC-ETH liquidity pool
				Now, a setTrustlineFlags operation is issued by the USDC-Issuer account to revoke ethIssuerAccount's USDC trustline.
				As a side-effect of this trustline revocation, there is a removal of ethIssuerAcount's stake from the USDB-ETH liquidity pool.
				In this case, 1 Claimable balance for USDC will be created, with claimantAccount = trustor i.e. ethIssuerAccount
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

			from := protoAddressFromLpHash(lp.liquidityPoolId)
			transferEvent := NewTransferEvent(meta, from,
				protoAddressFromClaimableBalanceId(cbsCreatedByThisLp[0].BalanceId),
				amount.String(cbsCreatedByThisLp[0].Amount),
				assetProto.NewProtoAsset(assetInCb))

			burnEvent := NewBurnEvent(meta, from, amount.String(burnedAmount), assetProto.NewProtoAsset(burnedAsset))
			events = append(events, transferEvent, burnEvent)

		} else if len(cbsCreatedByThisLp) == 2 {
			// Easy case - This LP created 2 claimable balances - one for each of the assets in the LP, to be sent to the account whose trustline was revoked.
			// so generate 2 transfer events
			from := protoAddressFromLpHash(lp.liquidityPoolId)
			asset1, asset2 := cbsCreatedByThisLp[0].Asset, cbsCreatedByThisLp[1].Asset
			to1, to2 := protoAddressFromClaimableBalanceId(cbsCreatedByThisLp[0].BalanceId), protoAddressFromClaimableBalanceId(cbsCreatedByThisLp[1].BalanceId)
			amt1, amt2 := amount.String(cbsCreatedByThisLp[0].Amount), amount.String(cbsCreatedByThisLp[1].Amount)

			events = append(events,
				NewTransferEvent(meta, from, to1, amt1, assetProto.NewProtoAsset(asset1)),
				NewTransferEvent(meta, from, to2, amt2, assetProto.NewProtoAsset(asset2)),
			)

		} else if len(cbsCreatedByThisLp) > 2 {
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

func liquidityPoolDepositEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be negative in LiquidityPool: %v", amtA, assetA.String(), lpIdToStrkey(lpId))
	}
	if amtB <= 0 {
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be negative in LiquidityPool: %v", amtB, assetB.String(), lpIdToStrkey(lpId))
	}

	meta := NewEventMeta(tx, &opIndex, nil)
	opSrcAcc := operationSourceAccount(tx, op)
	// From = operation source account, to = LP
	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{liquidityPoolId: &delta.liquidityPoolId}
	return []*TokenTransferEvent{
		mintOrBurnOrTransferEvent(assetA, from, to, amount.String(amtA), meta),
		mintOrBurnOrTransferEvent(assetB, from, to, amount.String(amtB), meta),
	}, nil
}

func liquidityPoolWithdrawEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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
		//TODO convert to strkey for LPId
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be negative in LiquidityPool: %v", amtA, assetA.String(), lpIdToStrkey(lpId))
	}
	if amtB <= 0 {
		//TODO convert to strkey for LPId
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be negative in LiquidityPool: %v", amtB, assetB.String(), lpIdToStrkey(lpId))
	}

	meta := NewEventMeta(tx, &opIndex, nil)
	opSrcAcc := operationSourceAccount(tx, op)
	// Opposite of LP Deposit. from = LP, to = operation source acocunt
	from, to := addressWrapper{liquidityPoolId: &delta.liquidityPoolId}, addressWrapper{account: &opSrcAcc}
	return []*TokenTransferEvent{
		mintOrBurnOrTransferEvent(assetA, from, to, amount.String(amtA), meta),
		mintOrBurnOrTransferEvent(assetB, from, to, amount.String(amtB), meta),
	}, nil
}

func generateEventsFromClaimAtoms(meta *EventMeta, opSrcAcc xdr.MuxedAccount, claims []xdr.ClaimAtom) []*TokenTransferEvent {
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

		// 2 events generated per trade
		events = append(events,
			mintOrBurnOrTransferEvent(claim.AssetSold(), sellerAddressWrapper, operationSourceAddressWrapper, amount.String(claim.AmountSold()), meta),
			mintOrBurnOrTransferEvent(claim.AssetBought(), operationSourceAddressWrapper, sellerAddressWrapper, amount.String(claim.AmountBought()), meta))
	}
	return events
}

func manageBuyOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	offersClaimed := result.Tr.MustManageBuyOfferResult().Success.OffersClaimed
	if len(offersClaimed) == 0 {
		return nil, nil
	}
	meta := NewEventMeta(tx, &opIndex, nil)
	return generateEventsFromClaimAtoms(meta, opSrcAcc, offersClaimed), nil
}

func manageSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	offersClaimed := result.Tr.MustManageSellOfferResult().Success.OffersClaimed
	if len(offersClaimed) == 0 {
		return nil, nil
	}
	meta := NewEventMeta(tx, &opIndex, nil)
	return generateEventsFromClaimAtoms(meta, opSrcAcc, offersClaimed), nil
}

// EXACTLY SAME as manageSellOfferEvents
func createPassiveSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	return manageSellOfferEvents(tx, opIndex, op, result)
}

func pathPaymentStrictSendEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	meta := NewEventMeta(tx, &opIndex, nil)
	opSrcAcc := operationSourceAccount(tx, op)
	strictSendOp := op.Body.MustPathPaymentStrictSendOp()
	strictSendResult := result.Tr.MustPathPaymentStrictSendResult()

	var events []*TokenTransferEvent
	events = append(events, generateEventsFromClaimAtoms(meta, opSrcAcc, strictSendResult.MustSuccess().Offers)...)

	// Generate one final event indicating the amount that the destination received in terms of destination asset
	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{account: &strictSendOp.Destination}
	events = append(events,
		mintOrBurnOrTransferEvent(strictSendOp.DestAsset, from, to, amount.String(strictSendResult.DestAmount()), meta))
	return events, nil
}

func pathPaymentStrictReceiveEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	meta := NewEventMeta(tx, &opIndex, nil)
	opSrcAcc := operationSourceAccount(tx, op)
	strictReceiveOp := op.Body.MustPathPaymentStrictReceiveOp()
	strictReceiveResult := result.Tr.MustPathPaymentStrictReceiveResult()

	var events []*TokenTransferEvent
	events = append(events, generateEventsFromClaimAtoms(meta, opSrcAcc, strictReceiveResult.MustSuccess().Offers)...)

	// Generate one final event indicating the amount that the destination received in terms of destination asset
	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{account: &strictReceiveOp.Destination}
	events = append(events,
		mintOrBurnOrTransferEvent(strictReceiveOp.DestAsset, from, to, amount.String(strictReceiveOp.DestAmount), meta))
	return events, nil
}

// Helper functions
func operationSourceAccount(tx ingest.LedgerTransaction, op xdr.Operation) xdr.MuxedAccount {
	acc := op.SourceAccount
	if acc != nil {
		return *acc
	}
	res := tx.Envelope.SourceAccount()
	return res
}

func protoAddressFromAccount(account xdr.MuxedAccount) *addressProto.Address {
	addr := &addressProto.Address{}
	switch account.Type {
	case xdr.CryptoKeyTypeKeyTypeEd25519:
		addr.AddressType = addressProto.AddressType_ADDRESS_TYPE_ACCOUNT
	case xdr.CryptoKeyTypeKeyTypeMuxedEd25519:
		addr.AddressType = addressProto.AddressType_ADDRESS_TYPE_MUXED_ACCOUNT
	}
	addr.StrKey = account.Address()
	return addr
}

func protoAddressFromLpHash(lpId xdr.PoolId) *addressProto.Address {
	return &addressProto.Address{
		AddressType: addressProto.AddressType_ADDRESS_TYPE_LIQUIDITY_POOL,
		StrKey:      lpIdToStrkey(lpId),
	}
}

func protoAddressFromClaimableBalanceId(cb xdr.ClaimableBalanceId) *addressProto.Address {
	return &addressProto.Address{
		AddressType: addressProto.AddressType_ADDRESS_TYPE_CLAIMABLE_BALANCE,
		StrKey:      cbIdToStrkey(cb),
	}
}

func cbIdToStrkey(cb xdr.ClaimableBalanceId) string {
	return cb.MustV0().HexString()
}

func lpIdToStrkey(lpId xdr.PoolId) string {
	return xdr.Hash(lpId).HexString()
}
