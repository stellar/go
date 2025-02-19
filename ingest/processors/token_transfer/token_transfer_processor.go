package token_transfer

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/address"
	"github.com/stellar/go/ingest/asset"
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
			opEvents, err := ProcessTokenTransferEventsFromOperation(uint32(i), op, opResult, tx)
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
func ProcessTokenTransferEventsFromOperation(opIndex uint32, op xdr.Operation, opResult xdr.OperationResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		return accountCreateEvents(op.Body.MustCreateAccountOp(), opResult.Tr.MustCreateAccountResult(), tx)
	// AccountMerge is wierd
	case xdr.OperationTypeAccountMerge:
		return mergeAccountEvents(op, opResult, tx)
	case xdr.OperationTypePayment:
		return paymentEvents(op.Body.MustPaymentOp(), opResult.Tr.MustPaymentResult(), tx)
	case xdr.OperationTypeCreateClaimableBalance:
		return createClaimableBalanceEvents(op.Body.MustCreateClaimableBalanceOp(), opResult.Tr.MustCreateClaimableBalanceResult(), tx)
	case xdr.OperationTypeClaimClaimableBalance:
		return claimClaimableBalanceEvents(op.Body.MustClaimClaimableBalanceOp(), opResult.Tr.MustClaimClaimableBalanceResult(), tx)
	case xdr.OperationTypeClawback:
		return clawbackEvents(op.Body.MustClawbackOp(), opResult.Tr.MustClawbackResult(), tx)
	case xdr.OperationTypeClawbackClaimableBalance:
		return clawbackClaimableBalanceEvents(op.Body.MustClawbackClaimableBalanceOp(), opResult.Tr.MustClawbackClaimableBalanceResult(), tx)
	case xdr.OperationTypeAllowTrust:
		return allowTrustEvents(op.Body.MustAllowTrustOp(), opResult.Tr.MustAllowTrustResult(), tx)
	case xdr.OperationTypeSetTrustLineFlags:
		return setTrustLineFlagsEvents(op.Body.MustSetTrustLineFlagsOp(), opResult.Tr.MustSetTrustLineFlagsResult(), tx)
	case xdr.OperationTypeLiquidityPoolDeposit:
		return liquidityPoolDepositEvents(op.Body.MustLiquidityPoolDepositOp(), opResult.Tr.MustLiquidityPoolDepositResult(), tx)
	case xdr.OperationTypeLiquidityPoolWithdraw:
		return liquidityPoolWithdrawEvents(op.Body.MustLiquidityPoolWithdrawOp(), opResult.Tr.MustLiquidityPoolWithdrawResult(), tx)
	case xdr.OperationTypeManageBuyOffer:
		return manageBuyOfferEvents(op.Body.MustManageBuyOfferOp(), opResult.Tr.MustManageBuyOfferResult(), tx)
	case xdr.OperationTypeManageSellOffer:
		return manageSellOfferEvents(op.Body.MustManageSellOfferOp(), opResult.Tr.MustManageSellOfferResult(), tx)
	case xdr.OperationTypeCreatePassiveSellOffer:
		return createPassiveSellOfferEvents(op.Body.MustCreatePassiveSellOfferOp(), opResult.Tr.MustCreatePassiveSellOfferResult(), tx)
	case xdr.OperationTypePathPaymentStrictSend:
		return pathPaymentStrictSendEvents(op.Body.MustPathPaymentStrictSendOp(), opResult.Tr.MustPathPaymentStrictSendResult(), tx)
	case xdr.OperationTypePathPaymentStrictReceive:
		return pathPaymentStrictReceiveEvents(op.Body.MustPathPaymentStrictReceiveOp(), opResult.Tr.MustPathPaymentStrictReceiveResult(), tx)
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
		event := NewFeeEvent(tx.Ledger.LedgerSequence(), tx.Ledger.ClosedAt(), tx.Hash.HexString(), address.AddressFromAccountId(accId), amt, asset.NewNativeAsset())
		events = append(events, event)
	}
	return events, nil
}

// Function stubs
func accountCreateEvents(op xdr.CreateAccountOp, result xdr.CreateAccountResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func mergeAccountEvents(op xdr.Operation, result xdr.OperationResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func paymentEvents(op xdr.PaymentOp, result xdr.PaymentResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func createClaimableBalanceEvents(op xdr.CreateClaimableBalanceOp, result xdr.CreateClaimableBalanceResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func claimClaimableBalanceEvents(op xdr.ClaimClaimableBalanceOp, result xdr.ClaimClaimableBalanceResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func clawbackEvents(op xdr.ClawbackOp, result xdr.ClawbackResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func clawbackClaimableBalanceEvents(op xdr.ClawbackClaimableBalanceOp, result xdr.ClawbackClaimableBalanceResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func allowTrustEvents(op xdr.AllowTrustOp, result xdr.AllowTrustResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func setTrustLineFlagsEvents(op xdr.SetTrustLineFlagsOp, result xdr.SetTrustLineFlagsResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func liquidityPoolDepositEvents(op xdr.LiquidityPoolDepositOp, result xdr.LiquidityPoolDepositResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func liquidityPoolWithdrawEvents(op xdr.LiquidityPoolWithdrawOp, result xdr.LiquidityPoolWithdrawResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func manageBuyOfferEvents(op xdr.ManageBuyOfferOp, result xdr.ManageBuyOfferResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func manageSellOfferEvents(op xdr.ManageSellOfferOp, result xdr.ManageSellOfferResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func createPassiveSellOfferEvents(op xdr.CreatePassiveSellOfferOp, result xdr.ManageSellOfferResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func pathPaymentStrictSendEvents(op xdr.PathPaymentStrictSendOp, result xdr.PathPaymentStrictSendResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}

func pathPaymentStrictReceiveEvents(op xdr.PathPaymentStrictReceiveOp, result xdr.PathPaymentStrictReceiveResult, tx ingest.LedgerTransaction) ([]*TokenTransferEvent, error) {
	return nil, nil
}
