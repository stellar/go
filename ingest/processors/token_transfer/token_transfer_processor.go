package token_transfer

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	addressProto "github.com/stellar/go/ingest/address"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
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

	operations := tx.Envelope.Operations()
	operationResults, _ := tx.Result.OperationResults()

	// Ensure we only process operations if the transaction was successful
	if tx.Result.Successful() {
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
	}

	return events, nil
}

// opIndex will be needed, on the offchance that we need to fetch ledgerEntry changes, especially in setTrustline or AllowTrust
func ProcessTokenTransferEventsFromOperation(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, opResult xdr.OperationResult) ([]*TokenTransferEvent, error) {
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		return accountCreateEvents(tx, opIndex, op)
	case xdr.OperationTypeAccountMerge:
		return mergeAccountEvents(tx, opIndex, op, opResult)
	case xdr.OperationTypePayment:
		return paymentEvents(tx, opIndex, op)
	case xdr.OperationTypeCreateClaimableBalance:
		return createClaimableBalanceEvents(tx, opIndex, op.Body.MustCreateClaimableBalanceOp(), opResult.Tr.MustCreateClaimableBalanceResult())
	case xdr.OperationTypeClaimClaimableBalance:
		return claimClaimableBalanceEvents(tx, opIndex, op.Body.MustClaimClaimableBalanceOp(), opResult.Tr.MustClaimClaimableBalanceResult())
	case xdr.OperationTypeClawback:
		return clawbackEvents(tx, opIndex, op.Body.MustClawbackOp(), opResult.Tr.MustClawbackResult())
	case xdr.OperationTypeClawbackClaimableBalance:
		return clawbackClaimableBalanceEvents(tx, opIndex, op.Body.MustClawbackClaimableBalanceOp(), opResult.Tr.MustClawbackClaimableBalanceResult())
	case xdr.OperationTypeAllowTrust:
		return allowTrustEvents(tx, opIndex, op.Body.MustAllowTrustOp(), opResult.Tr.MustAllowTrustResult())
	case xdr.OperationTypeSetTrustLineFlags:
		return setTrustLineFlagsEvents(tx, opIndex, op.Body.MustSetTrustLineFlagsOp(), opResult.Tr.MustSetTrustLineFlagsResult())
	case xdr.OperationTypeLiquidityPoolDeposit:
		return liquidityPoolDepositEvents(tx, opIndex, op.Body.MustLiquidityPoolDepositOp(), opResult.Tr.MustLiquidityPoolDepositResult())
	case xdr.OperationTypeLiquidityPoolWithdraw:
		return liquidityPoolWithdrawEvents(tx, opIndex, op.Body.MustLiquidityPoolWithdrawOp(), opResult.Tr.MustLiquidityPoolWithdrawResult())
	case xdr.OperationTypeManageBuyOffer:
		return manageBuyOfferEvents(tx, opIndex, op.Body.MustManageBuyOfferOp(), opResult.Tr.MustManageBuyOfferResult())
	case xdr.OperationTypeManageSellOffer:
		return manageSellOfferEvents(tx, opIndex, op.Body.MustManageSellOfferOp(), opResult.Tr.MustManageSellOfferResult())
	case xdr.OperationTypeCreatePassiveSellOffer:
		return createPassiveSellOfferEvents(tx, opIndex, op.Body.MustCreatePassiveSellOfferOp(), opResult.Tr.MustCreatePassiveSellOfferResult())
	case xdr.OperationTypePathPaymentStrictSend:
		return pathPaymentStrictSendEvents(tx, opIndex, op.Body.MustPathPaymentStrictSendOp(), opResult.Tr.MustPathPaymentStrictSendResult())
	case xdr.OperationTypePathPaymentStrictReceive:
		return pathPaymentStrictReceiveEvents(tx, opIndex, op.Body.MustPathPaymentStrictReceiveOp(), opResult.Tr.MustPathPaymentStrictReceiveResult())
	default:
		return nil, nil
	}
}

func generateFeeEvent(tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	var events []*TokenTransferEvent
	feeChanges := tx.GetFeeChanges()
	for _, change := range feeChanges {
		if change.Type != xdr.LedgerEntryTypeAccount {
			return nil, errors.Errorf("invalid ledgerEntryType for fee change: %s", change.Type.String())
		}

		// Do I need to do all this? Can I not simply use tx.Result.Result.FeeCharged
		preBalance := change.Pre.Data.MustAccount().Balance
		postBalance := change.Post.Data.MustAccount().Balance
		accId := change.Pre.Data.MustAccount().AccountId
		amt := amount.String(postBalance - preBalance)
		event := NewFeeEvent(tx.Ledger.LedgerSequence(), tx.Ledger.ClosedAt(), tx.Hash.HexString(), protoAddressFromAccountId(accId), amt, assetProto.NewNativeAsset())
		events = append(events, event)
	}
	return events, nil
}

// Function stubs
func accountCreateEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	srcAcc := operationSourceAccount(tx, op)
	createAccountOp := op.Body.MustCreateAccountOp()
	destAcc, amt := createAccountOp.Destination, amount.String(createAccountOp.StartingBalance)
	meta := NewEventMeta(tx, &opIndex, nil)
	event := NewTransferEvent(meta, protoAddressFromAccount(srcAcc), protoAddressFromAccountId(destAcc), amt, assetProto.NewNativeAsset())
	return []*TokenTransferEvent{event}, nil // Just one event will be generated
}

func mergeAccountEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation, result xdr.OperationResult) ([]*TokenTransferEvent, error) {
	res := result.Tr.MustAccountMergeResult()
	// If there is no transfer of XLM from source account to destination (i.e src account is empty), then no need to generate a transfer event
	if res.SourceAccountBalance == nil {
		return nil, nil
	}
	srcAcc := operationSourceAccount(tx, op)
	destAcc := op.Body.MustDestination()
	amt := amount.String(*res.SourceAccountBalance)
	meta := NewEventMeta(tx, &opIndex, nil)
	event := NewTransferEvent(meta, protoAddressFromAccount(srcAcc), protoAddressFromAccount(destAcc), amt, assetProto.NewNativeAsset())
	return []*TokenTransferEvent{event}, nil // Just one event will be generated
}

// Depending on the asset - if src or dest account == issuer of asset, then mint/burn event, else transfer event
func mintOrBurnOrTransferEvent(asset xdr.Asset, srcAcc xdr.MuxedAccount, destAcc xdr.MuxedAccount, amt string, meta *EventMeta) *TokenTransferEvent {
	protoAsset := assetProto.NewIssuedAsset(asset.GetCode(), asset.GetIssuer())
	var event *TokenTransferEvent
	sAddress := protoAddressFromAccount(srcAcc)
	dAddress := protoAddressFromAccount(destAcc)
	assetIssuerAccountId, _ := asset.GetIssuerAccountId()
	if assetIssuerAccountId.Equals(srcAcc.ToAccountId()) {
		// Mint event
		event = NewMintEvent(meta, dAddress, amt, protoAsset)
	} else if assetIssuerAccountId.Equals(destAcc.ToAccountId()) {
		// Burn event
		event = NewBurnEvent(meta, sAddress, amt, protoAsset)
	} else {
		// Regular transfer
		event = NewTransferEvent(meta, sAddress, dAddress, amt, protoAsset)
	}
	return event
}

func paymentEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.Operation) ([]*TokenTransferEvent, error) {
	paymentOp := op.Body.MustPaymentOp()
	srcAcc := operationSourceAccount(tx, op)
	destAcc := paymentOp.Destination
	amt := amount.String(paymentOp.Amount)
	meta := NewEventMeta(tx, &opIndex, nil)

	var event *TokenTransferEvent
	if paymentOp.Asset.IsNative() {
		// If native assetProto, it is always a regular transfer
		event = NewTransferEvent(meta, protoAddressFromAccount(srcAcc), protoAddressFromAccount(destAcc), amt, assetProto.NewNativeAsset())
	} else {
		event = mintOrBurnOrTransferEvent(paymentOp.Asset, srcAcc, destAcc, amt, meta)
	}
	return []*TokenTransferEvent{event}, nil
}

func createClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.CreateClaimableBalanceOp, result xdr.CreateClaimableBalanceResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func claimClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.ClaimClaimableBalanceOp, result xdr.ClaimClaimableBalanceResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func clawbackEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.ClawbackOp, result xdr.ClawbackResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func clawbackClaimableBalanceEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.ClawbackClaimableBalanceOp, result xdr.ClawbackClaimableBalanceResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func allowTrustEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.AllowTrustOp, result xdr.AllowTrustResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func setTrustLineFlagsEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.SetTrustLineFlagsOp, result xdr.SetTrustLineFlagsResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func liquidityPoolDepositEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.LiquidityPoolDepositOp, result xdr.LiquidityPoolDepositResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func liquidityPoolWithdrawEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.LiquidityPoolWithdrawOp, result xdr.LiquidityPoolWithdrawResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func manageBuyOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.ManageBuyOfferOp, result xdr.ManageBuyOfferResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func manageSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.ManageSellOfferOp, result xdr.ManageSellOfferResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func createPassiveSellOfferEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.CreatePassiveSellOfferOp, result xdr.ManageSellOfferResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func pathPaymentStrictSendEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.PathPaymentStrictSendOp, result xdr.PathPaymentStrictSendResult) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func pathPaymentStrictReceiveEvents(tx ingest.LedgerTransaction, opIndex uint32, op xdr.PathPaymentStrictReceiveOp, result xdr.PathPaymentStrictReceiveResult) ([]*TokenTransferEvent, error) {
	return nil, nil
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

func protoAddressFromAccountId(account xdr.AccountId) *addressProto.Address {
	return &addressProto.Address{
		AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT,
		StrKey:      account.Address(),
	}
}

func protoAddressFromLpHash(lpHash xdr.PoolId) *addressProto.Address {
	return &addressProto.Address{
		AddressType: addressProto.AddressType_ADDRESS_TYPE_LIQUIDITY_POOL,
		StrKey:      xdr.Hash(lpHash).HexString(), // replace with strkey
	}
}

func protoAddressFromClaimableBalanceId(cb xdr.ClaimableBalanceId) *addressProto.Address {
	return &addressProto.Address{
		AddressType: addressProto.AddressType_ADDRESS_TYPE_LIQUIDITY_POOL,
		StrKey:      cb.MustV0().HexString(), //replace with strkey
	}
}
