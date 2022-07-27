package index

import (
	"fmt"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

var (
	checkpointManager = historyarchive.NewCheckpointManager(0)
)

func ProcessTransaction(
	indexStore Store,
	ledger xdr.LedgerCloseMeta,
	tx ingest.LedgerTransaction,
) error {
	return indexStore.AddTransactionToIndexes(
		toid.New(int32(ledger.LedgerSequence()), int32(tx.Index), 0).ToInt64(),
		tx.Result.TransactionHash,
	)
}

// GetCheckpointNumber returns the next checkpoint NUMBER (NOT the checkpoint
// ledger sequence) corresponding to a given ledger sequence.
func GetCheckpointNumber(ledger uint32) uint32 {
	return 1 + (ledger / checkpointManager.GetCheckpointFrequency())
}

func ProcessAccounts(
	indexStore Store,
	ledger xdr.LedgerCloseMeta,
	tx ingest.LedgerTransaction,
) error {
	checkpoint := GetCheckpointNumber(ledger.LedgerSequence())

	allParticipants, err := GetTransactionParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexes(checkpoint, "all/all", allParticipants)
	if err != nil {
		return err
	}

	paymentsParticipants, err := GetPaymentParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexes(checkpoint, "all/payments", paymentsParticipants)
	if err != nil {
		return err
	}

	// if tx.Result.Successful() {
	// 	err = indexStore.AddParticipantsToIndexes(checkpoint, "successful/all", allParticipants)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	err = indexStore.AddParticipantsToIndexes(checkpoint, "successful/payments", paymentsParticipants)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func ProcessAccountsWithoutBackend(
	indexStore Store,
	ledger xdr.LedgerCloseMeta,
	tx ingest.LedgerTransaction,
) error {
	checkpoint := GetCheckpointNumber(ledger.LedgerSequence())

	allParticipants, err := GetTransactionParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "all/all", allParticipants)
	if err != nil {
		return err
	}

	paymentsParticipants, err := GetPaymentParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "all/payments", paymentsParticipants)
	if err != nil {
		return err
	}

	// if tx.Result.Successful() {
	// 	err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "successful/all", allParticipants)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "successful/payments", paymentsParticipants)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func GetPaymentParticipants(transaction ingest.LedgerTransaction) ([]string, error) {
	return participantsForOperations(transaction, true)
}

func GetTransactionParticipants(transaction ingest.LedgerTransaction) ([]string, error) {
	return participantsForOperations(transaction, false)
}

// transaction - the ledger transaction
// operation   - the operation within this transaction
// opIndex     - the 0 based index of the operation within the transaction
func GetOperationParticipants(transaction ingest.LedgerTransaction, operation xdr.Operation, opIndex int) ([]string, error) {
	return participantsForOperation(transaction, operation, opIndex, false)
}

func participantsForOperations(transaction ingest.LedgerTransaction, onlyPayments bool) ([]string, error) {
	var participants []string

	for opindex, operation := range transaction.Envelope.Operations() {
		opParticipants, err := participantsForOperation(transaction, operation, opindex, onlyPayments)
		if err != nil {
			return []string{}, err
		}
		participants = append(participants, opParticipants...)
	}

	// FIXME: Can/Should we make this a set? It may mean less superfluous
	// insertions into the index if there's a lot of activity by this
	// account in this transaction.
	return participants, nil
}

// transaction - the ledger transaction
// operation   - the operation within this transaction
// opIndex     - the 0 based index of the operation within the transaction
func participantsForOperation(transaction ingest.LedgerTransaction, operation xdr.Operation, opIndex int, onlyPayments bool) ([]string, error) {
	participants := []string{}
	opSource := operation.SourceAccount
	if opSource == nil {
		txSource := transaction.Envelope.SourceAccount()
		opSource = &txSource
	}
	switch operation.Body.Type {
	case xdr.OperationTypeCreateAccount,
		xdr.OperationTypePayment,
		xdr.OperationTypePathPaymentStrictReceive,
		xdr.OperationTypePathPaymentStrictSend,
		xdr.OperationTypeAccountMerge:
		participants = append(participants, opSource.Address())

	default:
		if onlyPayments {
			return participants, nil
		}
		participants = append(participants, opSource.Address())
	}

	switch operation.Body.Type {
	case xdr.OperationTypeCreateAccount:
		participants = append(participants, operation.Body.MustCreateAccountOp().Destination.Address())

	case xdr.OperationTypePayment:
		participants = append(participants, operation.Body.MustPaymentOp().Destination.ToAccountId().Address())

	case xdr.OperationTypePathPaymentStrictReceive:
		participants = append(participants, operation.Body.MustPathPaymentStrictReceiveOp().Destination.ToAccountId().Address())

	case xdr.OperationTypePathPaymentStrictSend:
		participants = append(participants, operation.Body.MustPathPaymentStrictSendOp().Destination.ToAccountId().Address())

	case xdr.OperationTypeAllowTrust:
		participants = append(participants, operation.Body.MustAllowTrustOp().Trustor.Address())

	case xdr.OperationTypeAccountMerge:
		participants = append(participants, operation.Body.MustDestination().ToAccountId().Address())

	case xdr.OperationTypeCreateClaimableBalance:
		for _, c := range operation.Body.MustCreateClaimableBalanceOp().Claimants {
			participants = append(participants, c.MustV0().Destination.Address())
		}

	case xdr.OperationTypeBeginSponsoringFutureReserves:
		participants = append(participants, operation.Body.MustBeginSponsoringFutureReservesOp().SponsoredId.Address())

	case xdr.OperationTypeEndSponsoringFutureReserves:
		// Failed transactions may not have a compliant sandwich structure
		// we can rely on (e.g. invalid nesting or a being operation with
		// the wrong sponsoree ID) and thus we bail out since we could
		// return incorrect information.
		if transaction.Result.Successful() {
			sponsoree := transaction.Envelope.SourceAccount().ToAccountId().Address()
			if operation.SourceAccount != nil {
				sponsoree = operation.SourceAccount.Address()
			}
			operations := transaction.Envelope.Operations()
			for i := opIndex - 1; i >= 0; i-- {
				if beginOp, ok := operations[i].Body.GetBeginSponsoringFutureReservesOp(); ok &&
					beginOp.SponsoredId.Address() == sponsoree {
					participants = append(participants, beginOp.SponsoredId.Address())
				}
			}
		}

	case xdr.OperationTypeRevokeSponsorship:
		op := operation.Body.MustRevokeSponsorshipOp()
		switch op.Type {
		case xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry:
			participants = append(participants, getLedgerKeyParticipants(*op.LedgerKey)...)

		case xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner:
			participants = append(participants, op.Signer.AccountId.Address())
			// We don't add signer as a participant because a signer can be
			// arbitrary account. This can spam successful operations
			// history of any account.
		}

	case xdr.OperationTypeClawback:
		op := operation.Body.MustClawbackOp()
		participants = append(participants, op.From.ToAccountId().Address())

	case xdr.OperationTypeSetTrustLineFlags:
		op := operation.Body.MustSetTrustLineFlagsOp()
		participants = append(participants, op.Trustor.Address())

	// for the following, the only direct participant is the source_account
	case xdr.OperationTypeManageBuyOffer:
	case xdr.OperationTypeManageSellOffer:
	case xdr.OperationTypeCreatePassiveSellOffer:
	case xdr.OperationTypeSetOptions:
	case xdr.OperationTypeChangeTrust:
	case xdr.OperationTypeInflation:
	case xdr.OperationTypeManageData:
	case xdr.OperationTypeBumpSequence:
	case xdr.OperationTypeClaimClaimableBalance:
	case xdr.OperationTypeClawbackClaimableBalance:
	case xdr.OperationTypeLiquidityPoolDeposit:
	case xdr.OperationTypeLiquidityPoolWithdraw:

	default:
		return nil, fmt.Errorf("unknown operation type: %s", operation.Body.Type)
	}
	return participants, nil
}

// getLedgerKeyParticipants returns a list of accounts that are considered
// "participants" in a particular ledger entry.
//
// This list will have zero or one element, making it easy to expand via `...`.
func getLedgerKeyParticipants(ledgerKey xdr.LedgerKey) []string {
	switch ledgerKey.Type {
	case xdr.LedgerEntryTypeAccount:
		return []string{ledgerKey.Account.AccountId.Address()}
	case xdr.LedgerEntryTypeData:
		return []string{ledgerKey.Data.AccountId.Address()}
	case xdr.LedgerEntryTypeOffer:
		return []string{ledgerKey.Offer.SellerId.Address()}
	case xdr.LedgerEntryTypeTrustline:
		return []string{ledgerKey.TrustLine.AccountId.Address()}
	case xdr.LedgerEntryTypeClaimableBalance:
		// nothing to do
	}
	return []string{}
}
