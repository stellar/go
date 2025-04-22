package token_transfer

import (
	"errors"
	"fmt"
	"io"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
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

type EventsProcessor struct {
	networkPassphrase     string
	disableContractEvents bool
}

type EventsProcessorOption func(*EventsProcessor)

var DisableContractEvents EventsProcessorOption = func(processor *EventsProcessor) {
	processor.disableContractEvents = true
}

func NewEventsProcessor(networkPassphrase string, options ...EventsProcessorOption) *EventsProcessor {
	proc := &EventsProcessor{
		networkPassphrase: networkPassphrase,
	}
	for _, opt := range options {
		opt(proc)
	}
	return proc
}

// EventsFromLedger processes token transfer events for all transactions in a given ledger.
// This function operates at the ledger level, iterating over all transactions in the ledger.
// it calls EventsFromTransaction to process token transfer events from each transaction within the ledger.
func (p *EventsProcessor) EventsFromLedger(lcm xdr.LedgerCloseMeta) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	txReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(p.networkPassphrase, lcm)
	if err != nil {
		return nil, fmt.Errorf("error creating transaction reader: %w", err)
	}

	for {
		var tx ingest.LedgerTransaction
		var txEvents []*TokenTransferEvent
		tx, err = txReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading transaction: %w", err)
		}
		txEvents, err = p.EventsFromTransaction(tx)
		if err != nil {
			return nil, err
		}
		events = append(events, txEvents...)
	}
	return events, nil
}

// EventsFromTransaction processes token transfer events for all operations within a given transaction.
//
//	First, it generates a FeeEvent for the transaction
//	If the transaction was successful, it processes all operations in the transaction by calling EventsFromOperation for each operation in the transaction.
//
// If the transaction is unsuccessful, it only generates events for transaction fees.
func (p *EventsProcessor) EventsFromTransaction(tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	feeEvents, err := p.generateFeeEvent(tx)
	if err != nil {
		return nil, fmt.Errorf("error generating fee event: %w", err)
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
		opEvents, err := p.EventsFromOperation(tx, uint32(i), op, opResult)
		if err != nil {
			return nil, err
		}

		events = append(events, opEvents...)
	}

	return events, nil
}

// EventsFromOperation processes token transfer events for a given operation within a transaction.
// It operates at the operation level, analyzing the operation type and generating corresponding token transfer events.
// If the operation is successful, it processes the event based on the operation type (e.g., payment, account creation, etc.).
// It handles various operation types like payments, account merges, trust line modifications, and more.
// There is a separate private function to derive events for each classic operation.
// It is implicitly assumed that the operation is successful, and thus will contribute towards generating events.
// which is why we don't check for the success code in the OperationResult
func (p *EventsProcessor) EventsFromOperation(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, opResult xdr.OperationResult) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	var err error
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		events, err = p.accountCreateEvents(tx, opIndex, op)
	case xdr.OperationTypeAccountMerge:
		events, err = p.mergeAccountEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePayment:
		events, err = p.paymentEvents(tx, opIndex, op)
	case xdr.OperationTypeCreateClaimableBalance:
		events, err = p.createClaimableBalanceEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeClaimClaimableBalance:
		events, err = p.claimClaimableBalanceEvents(tx, opIndex, op)
	case xdr.OperationTypeClawback:
		events, err = p.clawbackEvents(tx, opIndex, op)
	case xdr.OperationTypeClawbackClaimableBalance:
		events, err = p.clawbackClaimableBalanceEvents(tx, opIndex, op)
	case xdr.OperationTypeAllowTrust:
		events, err = p.allowTrustEvents(tx, opIndex, op)
	case xdr.OperationTypeSetTrustLineFlags:
		events, err = p.setTrustLineFlagsEvents(tx, opIndex, op)
	case xdr.OperationTypeLiquidityPoolDeposit:
		events, err = p.liquidityPoolDepositEvents(tx, opIndex, op)
	case xdr.OperationTypeLiquidityPoolWithdraw:
		events, err = p.liquidityPoolWithdrawEvents(tx, opIndex, op)
	case xdr.OperationTypeManageBuyOffer:
		events, err = p.manageBuyOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeManageSellOffer:
		events, err = p.manageSellOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeCreatePassiveSellOffer:
		events, err = p.createPassiveSellOfferEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePathPaymentStrictSend:
		events, err = p.pathPaymentStrictSendEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePathPaymentStrictReceive:
		events, err = p.pathPaymentStrictReceiveEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeInflation:
		events, err = p.inflationEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypeInvokeHostFunction:
		if !p.disableContractEvents {
			events, err = p.contractEvents(tx, opIndex)
		}
	default:
		return nil, nil
	}

	if err != nil {
		return nil, formatError(err, tx, opIndex, op)
	}

	// DO not run this reconciliation check for ledgers with protocol version >= 8
	if tx.Ledger.ProtocolVersion() >= 8 {
		return events, nil
	}

	// Run reconciliation for all operations except InvokeHostFunction
	reconciliationEvent, err := p.generateXlmReconciliationEvents(tx, opIndex, op, events)
	if err != nil {
		return nil, formatError(fmt.Errorf("error generating reconciliation events: %w", err), tx, opIndex, op)
	}

	if reconciliationEvent != nil {
		if reconciliationEvent.GetMint() != nil {
			// If it is a mint, put the mint event before the list of other events for this operation
			events = append([]*TokenTransferEvent{reconciliationEvent}, events...)
		} else if reconciliationEvent.GetBurn() != nil {
			// If it is a burn, put the burn event at the end of the list of other events for this operation
			events = append(events, reconciliationEvent)
		} else {
			return nil, formatError(fmt.Errorf("invalid reconciliation event type: %v. reconciliation event type can be only mint or burn", reconciliationEvent.GetEventType()), tx, opIndex, op)
		}
	}

	return events, nil
}

func (p *EventsProcessor) contractEvents(tx ingest.LedgerTransaction, opIndex uint32) ([]*TokenTransferEvent, error) {
	contractEvents, err := tx.GetContractEvents()
	if err != nil {
		return nil, fmt.Errorf("error getting contract events: %w", err)
	}
	events := make([]*TokenTransferEvent, 0, len(contractEvents))
	for _, contractEvent := range contractEvents {
		ev, err := p.parseEvent(tx, &opIndex, contractEvent)

		// You dont bail on error here, since error here means that it is not a sep-41 compliant token event.
		// Instead, if no error, it is a valid event, and it should be included in the output.
		if err == nil {
			events = append(events, ev)
		}
	}
	return events, nil
}

/*
Depending on the asset - if src or dest account == issuer of asset, then mint/burn event, else transfer event
All operation related functions will call this function instead of directly calling the underlying proto functions to generate events
The only exception to this is clawbackOperation and claimableClawbackOperation.
Those 2 will call the underlying proto function for clawback
*/
func (p *EventsProcessor) mintOrBurnOrTransferEvent(tx ingest.LedgerTransaction, opIndex *uint32, asset xdr.Asset, fromStrkey string, toStrkey string, amt string) (*TokenTransferEvent, error) {
	var isFromIssuer, isToIssuer bool
	fromAddress, toAddress := fromStrkey, toStrkey
	assetIssuerAccountId, _ := asset.GetIssuerAccountId()

	if strkey.IsValidEd25519PublicKey(fromStrkey) || strkey.IsValidMuxedAccountEd25519PublicKey(fromStrkey) {
		fromAccount := xdr.MustMuxedAddress(fromStrkey).ToAccountId()
		// Always revert back to G-Address for the from field, even if it is an M-address
		fromAddress = fromAccount.Address()

		if !asset.IsNative() && assetIssuerAccountId.Equals(fromAccount) {
			isFromIssuer = true
		}
	}

	if strkey.IsValidEd25519PublicKey(toStrkey) || strkey.IsValidMuxedAccountEd25519PublicKey(toStrkey) {
		toAccount := xdr.MustMuxedAddress(toStrkey).ToAccountId()
		// Always revert back to G-Address for the to field, even if it is an M-address
		toAddress = toAccount.Address()

		if !asset.IsNative() && assetIssuerAccountId.Equals(toAccount) {
			isToIssuer = true
		}
	}

	protoAsset := assetProto.NewProtoAsset(asset)
	meta := p.generateEventMeta(tx, opIndex, asset)

	// This means that the payment is a wierd one, where the src == dest AND in addition, the src/dest address is the issuer of the asset
	// Check this section out in CAP-67 https://github.com/stellar/stellar-protocol/blob/master/core/cap-0067.md#payment
	// We need to issue a TRANSFER event for this.
	// Keep in mind though that this wont show up in operationMeta as a balance change
	// This has happened in ledgerSequence: 4522126 on pubnet
	if isFromIssuer && isToIssuer {
		// There is no need to check or set muxed info here.
		return NewTransferEvent(meta, fromAddress, toAddress, amt, protoAsset), nil
	} else if isFromIssuer {

		// Check for Mint Event
		if toAddress == "" {
			return nil, NewEventError("mint event error: to address is nil")
		}
		mintEvent := NewMintEvent(meta, toAddress, amt, protoAsset)
		// Add muxed information - this will only have `to_muxed_info`, if at all
		err := mintEvent.addMuxedInfoForMintEvent(toStrkey, tx)
		if err != nil {
			return nil, err
		}
		return mintEvent, nil
	} else if isToIssuer {

		// Check for Burn Event
		if fromAddress == "" {
			return nil, NewEventError("burn event error: from address is nil")
		}
		return NewBurnEvent(meta, fromAddress, amt, protoAsset), nil
	}

	if fromAddress == "" {
		return nil, NewEventError("transfer event error: from address is nil")
	}
	if toAddress == "" {
		return nil, NewEventError("transfer event error: to address is nil")
	}
	// Create transfer event
	transferEvent := NewTransferEvent(meta, fromAddress, toAddress, amt, protoAsset)
	err := transferEvent.addMuxedInfoForTransferEvent(toStrkey, tx) // the addresses have to be the original from and to address
	if err != nil {
		return nil, err
	}
	return transferEvent, nil
}

func (p *EventsProcessor) generateEventMeta(tx ingest.LedgerTransaction, opIndex *uint32, asset xdr.Asset) *EventMeta {
	// Update the meta to always have contractId of the asset
	contractId, err := asset.ContractID(p.networkPassphrase)
	if err != nil {
		panic(fmt.Errorf("unable to generate ContractId from Asset:%v: %w", asset.StringCanonical(), err))
	}
	contractAddress := strkey.MustEncode(strkey.VersionByteContract, contractId[:])
	return NewEventMetaFromTx(tx, opIndex, contractAddress)
}

func (p *EventsProcessor) generateFeeEvent(tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
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

	meta := p.generateEventMeta(tx, nil, xlmAsset)
	event := NewFeeEvent(meta, protoAddressFromAccount(feeAccount), amount.String64Raw(xdr.Int64(feeAmt)), xlmProtoAsset)
	return []*TokenTransferEvent{event}, nil
}

// Function stubs
func (p *EventsProcessor) accountCreateEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	createAccountOp := op.Body.MustCreateAccountOp()
	destAcc, amt := createAccountOp.Destination.ToMuxedAccount(), amount.String64Raw(createAccountOp.StartingBalance)
	event, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, xlmAsset, opSrcAcc.Address(), destAcc.Address(), amt)
	if err != nil {
		return nil, err
	}

	return []*TokenTransferEvent{event}, nil // Just one event will be generated
}

func (p *EventsProcessor) mergeAccountEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	res := result.Tr.MustAccountMergeResult()
	// If there is no transfer of XLM from source account to destination (i.e. src account is empty), then no need to generate a transfer event
	if res.SourceAccountBalance == nil {
		return nil, nil
	}
	opSrcAcc := operationSourceAccount(tx, op)
	destAcc := op.Body.MustDestination()
	amt := amount.String64Raw(*res.SourceAccountBalance)
	event, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, xlmAsset, opSrcAcc.Address(), destAcc.Address(), amt)
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil // Just one event will be generated
}

func (p *EventsProcessor) paymentEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	paymentOp := op.Body.MustPaymentOp()
	opSrcAcc := operationSourceAccount(tx, op)
	destAcc := paymentOp.Destination
	amt := amount.String64Raw(paymentOp.Amount)

	event, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, paymentOp.Asset, opSrcAcc.Address(), destAcc.Address(), amt)
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil
}

func (p *EventsProcessor) inflationEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	payouts := result.Tr.MustInflationResult().MustPayouts()

	var mintEvents []*TokenTransferEvent
	meta := p.generateEventMeta(tx, &opIndex, xlmAsset)
	for _, recipient := range payouts {
		// NOTE: there can never be any muxed info in the inflation mint event. so no need to fo mintEvent.addMuxedInfoForMintEvent()
		mintEvents = append(mintEvents, NewMintEvent(meta, recipient.Destination.Address(), amount.String64Raw(recipient.Amount), xlmProtoAsset))
	}
	return mintEvents, nil
}

func (p *EventsProcessor) createClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	createCbOp := op.Body.MustCreateClaimableBalanceOp()
	createCbResult := result.Tr.MustCreateClaimableBalanceResult()
	opSrcAcc := operationSourceAccount(tx, op)
	cbId := createCbResult.MustBalanceId()

	event, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, createCbOp.Asset, opSrcAcc.Address(), cbIdToStrkey(cbId), amount.String64Raw(createCbOp.Amount))
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil
}

func (p *EventsProcessor) claimClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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
	event, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, cb.Asset, cbIdToStrkey(cbId), opSrcAcc.Address(), amount.String64Raw(cb.Amount))
	if err != nil {
		return nil, err
	}
	return []*TokenTransferEvent{event}, nil
}

func (p *EventsProcessor) clawbackEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	clawbackOp := op.Body.MustClawbackOp()
	meta := p.generateEventMeta(tx, &opIndex, clawbackOp.Asset)

	// fromAddress is NOT the operationSourceAccount.
	// It is the account specified in the operation from whom you want money to be clawed back
	from := protoAddressFromAccount(clawbackOp.From)
	event := NewClawbackEvent(meta, from, amount.String64Raw(clawbackOp.Amount), assetProto.NewProtoAsset(clawbackOp.Asset))
	return []*TokenTransferEvent{event}, nil
}

func (p *EventsProcessor) clawbackClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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
	meta := p.generateEventMeta(tx, &opIndex, cb.Asset)
	// Money is clawed back from the claimableBalanceId
	event := NewClawbackEvent(meta, cbIdToStrkey(cbId), amount.String64Raw(cb.Amount), assetProto.NewProtoAsset(cb.Asset))
	return []*TokenTransferEvent{event}, nil
}

func (p *EventsProcessor) allowTrustEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	// ?? Should I be checking for generation of liquidity pools and CBs iff the flag is set to false?
	// isAuthRevoked := op.Body.MustAllowTrustOp().Authorize == 0
	return p.generateEventsForRevokedTrustlines(tx, opIndex)
}

func (p *EventsProcessor) setTrustLineFlagsEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	// ?? Should I be checking for generation of liquidity pools and CBs iff the flag is set to false?
	// isAuthRevoked := op.Body.MustSetTrustLineFlagsOp().ClearFlags != 0
	return p.generateEventsForRevokedTrustlines(tx, opIndex)
}

func (p *EventsProcessor) generateEventsForRevokedTrustlines(tx ingest.LedgerTransaction, opIndex uint32) ([]*TokenTransferEvent, error) {
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

			transferEvent, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, assetInCb,
				lpIdToStrkey(from),
				cbIdToStrkey(cbsCreatedByThisLp[0].BalanceId),
				amount.String64Raw(cbsCreatedByThisLp[0].Amount))
			if err != nil {
				return nil, err
			}

			burnMeta := p.generateEventMeta(tx, &opIndex, burnedAsset)
			burnEvent := NewBurnEvent(burnMeta, lpIdToStrkey(from), amount.String64Raw(burnedAmount), assetProto.NewProtoAsset(burnedAsset))
			events = append(events, transferEvent, burnEvent)

		} else if len(cbsCreatedByThisLp) == 2 {
			// Easy case - This LP created 2 claimable balances - one for each of the assets in the LP, to be sent to the account whose trustline was revoked.
			// so generate 2 transfer events
			from := lp.liquidityPoolId
			asset1, asset2 := cbsCreatedByThisLp[0].Asset, cbsCreatedByThisLp[1].Asset
			to1, to2 := cbsCreatedByThisLp[0].BalanceId, cbsCreatedByThisLp[1].BalanceId
			amt1, amt2 := amount.String64Raw(cbsCreatedByThisLp[0].Amount), amount.String64Raw(cbsCreatedByThisLp[1].Amount)

			ev1, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, asset1,
				lpIdToStrkey(from),
				cbIdToStrkey(to1),
				amt1)
			if err != nil {
				return nil, err
			}

			ev2, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, asset2,
				lpIdToStrkey(from),
				cbIdToStrkey(to2),
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

func (p *EventsProcessor) liquidityPoolDepositEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be zero or negative for LiquidityPool: %v", amtA, assetA.String(), lpIdToStrkey(lpId))
	}
	if amtB <= 0 {
		return nil,
			fmt.Errorf("deposited amount (%v) for asset: %v, cannot be zero or negative for LiquidityPool: %v", amtB, assetB.String(), lpIdToStrkey(lpId))
	}

	opSrcAcc := operationSourceAccount(tx, op)
	// From = operation source account, to = LP
	var events []*TokenTransferEvent
	event, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, assetA, opSrcAcc.Address(), lpIdToStrkey(delta.liquidityPoolId), amount.String64Raw(amtA))
	if err != nil {
		return nil, err
	}
	events = append(events, event)

	event, err = p.mintOrBurnOrTransferEvent(tx, &opIndex, assetB, opSrcAcc.Address(), lpIdToStrkey(delta.liquidityPoolId), amount.String64Raw(amtB))
	if err != nil {
		return nil, err
	}
	events = append(events, event)

	return events, nil
}

func (p *EventsProcessor) liquidityPoolWithdrawEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
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
	/*
		This is slightly different from the LpDeposit operation check. In LpDeposit, if amt <=0: then error
		However, for LpWithdraw, the check is if amt < 0: then error.
		This is because, the rounding on withdraw could result in nothing being withdrawn
		Refer https://github.com/sisuresh/stellar-protocol/blob/unified/core/cap-0038.md#price-bounds-for-liquiditypoolwithdrawop
	*/
	if amtA < 0 {
		return nil,
			fmt.Errorf("withdrawn amount (%v) for asset: %v, cannot be negative for LiquidityPool: %v", amtA, assetA.String(), lpIdToStrkey(lpId))
	}
	if amtB < 0 {
		return nil,
			fmt.Errorf("withdrawn amount (%v) for asset: %v, cannot be negative for LiquidityPool: %v", amtB, assetB.String(), lpIdToStrkey(lpId))
	}

	opSrcAcc := operationSourceAccount(tx, op)
	// Opposite of LP Deposit. from = LP, to = operation source account
	var events []*TokenTransferEvent
	event, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, assetA, lpIdToStrkey(delta.liquidityPoolId), opSrcAcc.Address(), amount.String64Raw(amtA))
	if err != nil {
		return nil, err
	}
	events = append(events, event)

	event, err = p.mintOrBurnOrTransferEvent(tx, &opIndex, assetB, lpIdToStrkey(delta.liquidityPoolId), opSrcAcc.Address(), amount.String64Raw(amtB))
	if err != nil {
		return nil, err
	}
	events = append(events, event)

	return events, nil
}

func (p *EventsProcessor) generateEventsFromClaimAtoms(tx ingest.LedgerTransaction, opIndex uint32, opSrcAcc xdr.MuxedAccount, claims []xdr.ClaimAtom) ([]*TokenTransferEvent, error) {
	// Converting MuxedAccount to strictly G-Addresss, since events from trading pair shouldnt have destination muxed_info set
	operationSource := opSrcAcc.ToAccountId().Address()
	var events []*TokenTransferEvent
	var seller string

	for _, claim := range claims {
		if claim.Type == xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool {
			lpId := claim.MustLiquidityPool().LiquidityPoolId
			seller = lpIdToStrkey(lpId)
		} else {
			sellerId := claim.SellerId()
			sellerAccount := sellerId.ToMuxedAccount()
			seller = sellerAccount.Address()
		}

		ev1, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, claim.AssetSold(), seller, operationSource, amount.String64Raw(claim.AmountSold()))
		if err != nil {
			return nil, err
		}
		ev2, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, claim.AssetBought(), operationSource, seller, amount.String64Raw(claim.AmountBought()))
		if err != nil {
			return nil, err
		}

		// 2 events generated per trade
		events = append(events, ev1, ev2)
	}
	return events, nil
}

func (p *EventsProcessor) manageBuyOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	offersClaimed := result.Tr.MustManageBuyOfferResult().Success.OffersClaimed
	if len(offersClaimed) == 0 {
		return nil, nil
	}
	return p.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, offersClaimed)
}

func (p *EventsProcessor) manageSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	offersClaimed := result.Tr.MustManageSellOfferResult().Success.OffersClaimed
	if len(offersClaimed) == 0 {
		return nil, nil
	}
	return p.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, offersClaimed)
}

// EXACTLY SAME as manageSellOfferEvents
func (p *EventsProcessor) createPassiveSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	return p.manageSellOfferEvents(tx, opIndex, op, result)
}

func (p *EventsProcessor) pathPaymentStrictSendEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	strictSendOp := op.Body.MustPathPaymentStrictSendOp()
	strictSendResult := result.Tr.MustPathPaymentStrictSendResult()

	var events []*TokenTransferEvent
	ev, err := p.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, strictSendResult.MustSuccess().Offers)
	if err != nil {
		return nil, err
	}
	events = append(events, ev...)

	// Generate one final event indicating the amount that the destination received in terms of destination asset
	finalEvent, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, strictSendOp.DestAsset, opSrcAcc.Address(), strictSendOp.Destination.Address(), amount.String64Raw(strictSendResult.DestAmount()))
	if err != nil {
		return nil, err
	}
	events = append(events, finalEvent)
	return events, nil
}

func (p *EventsProcessor) pathPaymentStrictReceiveEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	opSrcAcc := operationSourceAccount(tx, op)
	strictReceiveOp := op.Body.MustPathPaymentStrictReceiveOp()
	strictReceiveResult := result.Tr.MustPathPaymentStrictReceiveResult()

	var events []*TokenTransferEvent
	ev, err := p.generateEventsFromClaimAtoms(tx, opIndex, opSrcAcc, strictReceiveResult.MustSuccess().Offers)
	if err != nil {
		return nil, err
	}
	events = append(events, ev...)

	// Generate one final event indicating the amount that the destination received in terms of destination asset
	finalEvent, err := p.mintOrBurnOrTransferEvent(tx, &opIndex, strictReceiveOp.DestAsset, opSrcAcc.Address(), strictReceiveOp.Destination.Address(), amount.String64Raw(strictReceiveOp.DestAmount))
	if err != nil {
		return nil, err
	}
	events = append(events, finalEvent)
	return events, nil
}

/*
This code needs to be run to compare the diffs between the changesMap and eventsMap for the (sourceAccount, XLM) combination
This needs to be run for each operation, when ledgerSeq <= 8
For more details, on why this is needed, refer - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0067.md#retroactively-emitting-events

The maybeGenerateMintOrBurnEvents function takes in an account and an asset, but in reality, this will only be called for operationSourceAccount and strictly for XLM
*/
func (p *EventsProcessor) maybeGenerateMintOrBurnEventsForReconciliation(tx ingest.LedgerTransaction, opIndex uint32, changesMap, eventsMap map[balanceKey]int64, account xdr.MuxedAccount, asset xdr.Asset) (*TokenTransferEvent, error) {
	accountStr := account.ToAccountId().Address()
	// Create the balance key for this account and XLM asset
	key := balanceKey{holder: accountStr, asset: asset.StringCanonical()}

	// Get the balance changes from both maps
	changesBalance := changesMap[key]
	eventsBalance := eventsMap[key]

	/*
		Highlighting all possible scenarios:

		1.	 Account exists in both maps
			changesMap[account, XLM] = 100	-- 	Account gained 100 XLM according to ledger changes
			eventsMap[account, XLM] = 70	--	But our events only account for 70 XLM
			diff = 100 - 70 = 30 > 0, so we issue a MINT event for 30 XLM

		2.	 Account exists in both maps (negative balance)
			changesMap[account, XLM] = -50	--	Account lost 50 XLM according to ledger changes
			eventsMap[account, XLM] = -20	--	But our events only account for 20 XLM being lost
			diff = -50 - (-20) = -30 < 0, so we issue a BURN event for 30 XLM

		3.	Account only in changesMap
			changesMap[account, XLM] = 50	--	Account gained 50 XLM according to ledger changes
			eventsMap[account, XLM] = 0	--	No events recorded for this account (key doesn't exist)
			diff = 50 - 0 = 50 > 0, so we issue a MINT event for 50 XLM

		4.	Account only in eventsMap
			changesMap[account, XLM] = 0	--	No changes recorded for this account (key doesn't exist)
			eventsMap[account, XLM] = 30	--	But our events show 30 XLM being credited
			diff = 0 - 30 = -30 < 0, so we issue a BURN event for 30 XLM

			More nuanced scenarios

		5.  changesMap positive, eventsMap negative
			changesMap[account, XLM] = 100	--	Account gained 100 XLM according to ledger changes
			eventsMap[account, XLM] = -50	--	But our events show 50 XLM being deducted
			diff = 100 - (-50) = 150 > 0, so we issue a MINT event for 150 XLM

		6.	changesMap negative, eventsMap positive
			changesMap[account, XLM] = -80	--	Account lost 80 XLM according to ledger changes
			eventsMap[account, XLM] = 40	--	But our events show 40 XLM being credited
			diff = -80 - 40 = -120 < 0, so we issue a BURN event for 120 XLM

			Even more nuanced scenarios

		7.	Opposite of Case 2
			changesMap[account, XLM] = -20	--	Account lost 20 XLM according to ledger changes
			eventsMap[account, XLM] = -50	--	But our events show 50 XLM being lost
			diff = -20 - (-50) = 30 > 0, so we issue a MINT event for 30 XLM

		8. 	Opposite of Case 1
			changesMap[account, XLM] = 70	-- 	Account gained 70 XLM according to ledger changes
			eventsMap[account, XLM] = 100	--	But our events only account for 100 XLM
			diff = 70 - 100 = -30, so we issue a BURN event for 30 XLM


	*/

	// Both maps have entries for this account/asset
	diff := changesBalance - eventsBalance
	// Not in either map, no difference

	// If no difference, no mint or burn needs to be emitted
	if diff == 0 {
		return nil, nil
	}

	// There will only be one event - either a mint or burn that will need to be generated.
	var mintOrBurnEvent *TokenTransferEvent
	meta := p.generateEventMeta(tx, &opIndex, asset)
	protoAsset := assetProto.NewProtoAsset(asset)

	// Generate appropriate event based on the difference
	if diff > 0 {
		// changesMap shows more XLM than eventsMap - need to MINT
		mintOrBurnEvent = NewMintEvent(meta, accountStr, amount.String64Raw(xdr.Int64(diff)), protoAsset)
		err := mintOrBurnEvent.addMuxedInfoForMintEvent(account.Address(), tx)
		if err != nil {
			return nil, fmt.Errorf("error in generating XLM reconciliation mint event: %w", err)
		}
	} else {
		// changesMap shows less XLM than eventsMap - need to BURN
		mintOrBurnEvent = NewBurnEvent(meta, accountStr, amount.String64Raw(xdr.Int64(-diff)), protoAsset)
	}

	return mintOrBurnEvent, nil
}

func (p *EventsProcessor) generateXlmReconciliationEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, operationEvents []*TokenTransferEvent) (*TokenTransferEvent, error) {

	operationChanges, err := tx.GetOperationChanges(opIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation changes for operation Index: %v: %w", opIndex, err)
	}
	changesMap := findBalanceDeltasFromChanges(operationChanges)
	eventsMap := findBalanceDeltasFromEvents(operationEvents)
	operationSrcAccount := operationSourceAccount(tx, op)

	return p.maybeGenerateMintOrBurnEventsForReconciliation(tx, opIndex, changesMap, eventsMap, operationSrcAccount, xlmAsset)
}
