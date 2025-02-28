package token_transfer

import (
	"fmt"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	addressProto "github.com/stellar/go/ingest/address"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
)

var (
	xlmProtoAsset                    = assetProto.NewNativeAsset()
	ErrLiquidityPoolEntryNotFound    = errors.New("liquidity pool entry not found in operation changes")
	ErrClaimableBalanceEntryNotFound = errors.New("claimable balance entry not found in operation changes")
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
		txEvents, err = ProcessTokenTransferEventsFromTransaction(tx)
		if err != nil {
			return nil, errors.Wrap(err, "error processing token transfer events from transaction")
		}
		events = append(events, txEvents...)
	}
	return events, nil
}

func ProcessTokenTransferEventsFromTransaction(tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
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
		opEvents, err := ProcessTokenTransferEventsFromOperation(tx, uint32(i), op, opResult)
		if err != nil {
			return nil,
				errors.Wrapf(err, "error processing token transfer events from operation, index: %d,  %s", i, op.Body.Type.String())
		}

		events = append(events, opEvents...)
	}

	return events, nil
}

// ProcessTokenTransferEventsFromOperation
// There is a separate private function to derive events for each operation.
// It is implicitly assumed that the operation is successful, and thus will contribute towards generating events.
// which is why we dont check for the code in the OperationResult
func ProcessTokenTransferEventsFromOperation(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, opResult xdr.OperationResult) ([]*TokenTransferEvent, error) {
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
		return allowTrustEvents(tx, opIndex, op.Body.MustAllowTrustOp(), opResult.Tr.MustAllowTrustResult())
	case xdr.OperationTypeSetTrustLineFlags:
		return setTrustLineFlagsEvents(tx, opIndex, op.Body.MustSetTrustLineFlagsOp(), opResult.Tr.MustSetTrustLineFlagsResult())
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

func getClaimableBalanceEntriesFromOperationChanges(tx ingest.LedgerTransaction, opIndex uint32) ([]xdr.ClaimableBalanceEntry, error) {
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
		if change.Pre != nil && change.Post == nil {
			cb = change.Pre.Data.MustClaimableBalance()
			entries = append(entries, cb)
		} else if change.Post != nil && change.Pre == nil { // check if claimable balance entry is created
			cb = change.Post.Data.MustClaimableBalance()
			entries = append(entries, cb)
		}
	}

	if len(entries) == 0 {
		return nil, ErrClaimableBalanceEntryNotFound
	}
	return entries, nil
}

func claimClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	claimCbOp := op.Body.MustClaimClaimableBalanceOp()
	cbId := claimCbOp.BalanceId

	// After this operation, the CB will be deleted.
	cbEntries, err := getClaimableBalanceEntriesFromOperationChanges(tx, opIndex)
	if err != nil {
		return nil, err
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
	cbEntries, err := getClaimableBalanceEntriesFromOperationChanges(tx, opIndex)
	if err != nil {
		return nil, err
	} else if len(cbEntries) != 1 {
		return nil, fmt.Errorf("more than one claimable entry found for operation: %s", op.Body.Type.String())
	}

	cb := cbEntries[0]
	meta := NewEventMeta(tx, &opIndex, nil)
	// Money is clawed back from the claimableBalanceId
	event := NewClawbackEvent(meta, protoAddressFromClaimableBalanceId(cbId), amount.String(cb.Amount), assetProto.NewProtoAsset(cb.Asset))
	return []*TokenTransferEvent{event}, nil
}

func allowTrustEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.AllowTrustOp, result xdr.AllowTrustResult) ([]*TokenTransferEvent, error) {
	/*
		IF this operation is used to revoke authorization from a trustline that deposited into a liquidity pool,
		then UPTO 2 claimable balances will be created for the withdrawn assets (See CAP-0038 for more info)

		Code logic:
		- Go through the operation changes and find the LiquidityPool from which assets were withdrawn. (QUESTION = Can there be more than LP that shows up in operation ledgerEntryChanges??)
		- IF no LP found, then nothing to do
		- If a LP is found, then go through the operation changes to fetch the (upto 2) Claimable Balances created.
			for each claimable balance create a transfer event -- from: LpHash, to: CBid, asset = one asset from LP, someAmount


	*/
	return nil, nil
}

type liquidityPoolEntryDelta struct {
	lpId               xdr.PoolId
	assetA             xdr.Asset
	assetB             xdr.Asset
	reservesRemainingA xdr.Int64
	reservesRemainingB xdr.Int64
}

func setTrustLineFlagsEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.SetTrustLineFlagsOp, result xdr.SetTrustLineFlagsResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func getImpactedLiquidityPoolEntriesFromOperation(opIndex uint32, tx ingest.LedgerTransaction) ([]liquidityPoolEntryDelta, error) {
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
			entry.lpId = lp.LiquidityPoolId
			cp := lp.Body.ConstantProduct
			entry.assetA, entry.assetB = cp.Params.AssetA, cp.Params.AssetB
			preA, preB = cp.ReserveA, cp.ReserveB
		}

		var postA, postB xdr.Int64
		if c.Post != nil {
			lp = c.Post.Data.LiquidityPool
			entry.lpId = lp.LiquidityPoolId
			cp := lp.Body.ConstantProduct
			entry.assetA, entry.assetB = cp.Params.AssetA, cp.Params.AssetB
			postA, postB = cp.ReserveA, cp.ReserveB
		}
		
		entry.reservesRemainingA = postA - preA
		entry.reservesRemainingB = postB - preB
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, ErrLiquidityPoolEntryNotFound
	}
	return entries, nil
}

func liquidityPoolDepositEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	lpDeltas, e := getImpactedLiquidityPoolEntriesFromOperation(opIndex, tx)
	if e != nil {
		return nil, e
	} else if len(lpDeltas) != 1 {
		return nil, fmt.Errorf("more than one Liquidiy pool entry found for operation: %s", op.Body.Type.String())
	}

	delta := lpDeltas[0]
	lpId := delta.lpId
	assetA, assetB := delta.assetA, delta.assetB
	// delta is calculated as (post - pre) for the ledgerEntryChange
	amtA, amtB := delta.reservesRemainingA, delta.reservesRemainingB
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
	from, to := addressWrapper{account: &opSrcAcc}, addressWrapper{liquidityPoolId: &delta.lpId}
	return []*TokenTransferEvent{
		mintOrBurnOrTransferEvent(assetA, from, to, amount.String(amtA), meta),
		mintOrBurnOrTransferEvent(assetB, from, to, amount.String(amtB), meta),
	}, nil
}

func liquidityPoolWithdrawEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	lpDeltas, e := getImpactedLiquidityPoolEntriesFromOperation(opIndex, tx)
	if e != nil {
		return nil, e
	} else if len(lpDeltas) != 1 {
		return nil, fmt.Errorf("more than one Liquidiy pool entry found for operation: %s", op.Body.Type.String())
	}

	delta := lpDeltas[0]
	lpId := delta.lpId
	assetA, assetB := delta.assetA, delta.assetB
	// delta is calculated as (post - pre) for the ledgerEntryChange. For withdraw operation, reverse the sign
	amtA, amtB := -delta.reservesRemainingA, -delta.reservesRemainingB
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
	from, to := addressWrapper{liquidityPoolId: &delta.lpId}, addressWrapper{account: &opSrcAcc}
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
