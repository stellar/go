package index

import (
	"context"
	"fmt"
	"io"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

// Module is a way to process data and store it into an index.
type Module func(
	idx Store,
	ledger xdr.LedgerCloseMeta,
	checkpoint uint32,
	transaction ingest.LedgerTransaction,
) error

// IndexBuilder contains everything needed to build indices from ledger ranges.
type IndexBuilder struct {
	store             Store
	history           ledgerbackend.HistoryArchiveBackend
	networkPassphrase string

	modules []Module
}

func NewIndexBuilder(
	indexStore Store,
	backend ledgerbackend.HistoryArchiveBackend,
	networkPassphrase string,
) *IndexBuilder {
	return &IndexBuilder{
		store:             indexStore,
		history:           backend,
		networkPassphrase: networkPassphrase,
	}
}

// RegisterModule adds a module to process every given ledger. It is not
// threadsafe and all calls should be made *before* any calls to `Build`.
func (builder *IndexBuilder) RegisterModule(module Module) {
	builder.modules = append(builder.modules, module)
}

// RunModules executes all of the registered modules on the given ledger.
func (builder *IndexBuilder) RunModules(
	ledger xdr.LedgerCloseMeta,
	checkpoint uint32,
	tx ingest.LedgerTransaction,
) error {
	for _, module := range builder.modules {
		if err := module(builder.store, ledger, checkpoint, tx); err != nil {
			return err
		}
	}

	return nil
}

func (builder *IndexBuilder) Build(ctx context.Context, ledgerRange historyarchive.Range) error {
	for ledgerSeq := ledgerRange.Low; ledgerSeq <= ledgerRange.High; ledgerSeq++ {
		ledger, err := builder.history.GetLedger(ctx, ledgerSeq)
		if err != nil {
			log.WithField("error", err).Errorf("error getting ledger %d", ledgerSeq)
			return err
		}

		checkpoint := (ledgerSeq / 64) + 1

		reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(
			builder.networkPassphrase, ledger)
		if err != nil {
			return err
		}

		for {
			tx, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			if err := builder.RunModules(ledger, checkpoint, tx); err != nil {
				return err
			}
		}
	}

	return nil
}

func ProcessTransaction(
	indexStore Store,
	ledger xdr.LedgerCloseMeta,
	_ uint32,
	tx ingest.LedgerTransaction,
) error {
	return indexStore.AddTransactionToIndexes(
		toid.New(int32(ledger.LedgerSequence()), int32(tx.Index), 0).ToInt64(),
		tx.Result.TransactionHash,
	)
}

func ProcessAccounts(
	indexStore Store,
	_ xdr.LedgerCloseMeta,
	checkpoint uint32,
	tx ingest.LedgerTransaction,
) error {
	allParticipants, err := getParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexes(checkpoint, "all_all", allParticipants)
	if err != nil {
		return err
	}

	paymentsParticipants, err := getPaymentParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexes(checkpoint, "all_payments", paymentsParticipants)
	if err != nil {
		return err
	}

	if tx.Result.Successful() {
		err = indexStore.AddParticipantsToIndexes(checkpoint, "successful_all", allParticipants)
		if err != nil {
			return err
		}

		err = indexStore.AddParticipantsToIndexes(checkpoint, "successful_payments", paymentsParticipants)
		if err != nil {
			return err
		}
	}

	return nil
}
func getPaymentParticipants(transaction ingest.LedgerTransaction) ([]string, error) {
	return participantsForOperations(transaction, true)
}

func getParticipants(transaction ingest.LedgerTransaction) ([]string, error) {
	return participantsForOperations(transaction, false)
}

func participantsForOperations(transaction ingest.LedgerTransaction, onlyPayments bool) ([]string, error) {
	var participants []string

	for opindex, operation := range transaction.Envelope.Operations() {
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
				continue
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
		case xdr.OperationTypeManageBuyOffer:
			// the only direct participant is the source_account
		case xdr.OperationTypeManageSellOffer:
			// the only direct participant is the source_account
		case xdr.OperationTypeCreatePassiveSellOffer:
			// the only direct participant is the source_account
		case xdr.OperationTypeSetOptions:
			// the only direct participant is the source_account
		case xdr.OperationTypeChangeTrust:
			// the only direct participant is the source_account
		case xdr.OperationTypeAllowTrust:
			participants = append(participants, operation.Body.MustAllowTrustOp().Trustor.Address())
		case xdr.OperationTypeAccountMerge:
			participants = append(participants, operation.Body.MustDestination().ToAccountId().Address())
		case xdr.OperationTypeInflation:
			// the only direct participant is the source_account
		case xdr.OperationTypeManageData:
			// the only direct participant is the source_account
		case xdr.OperationTypeBumpSequence:
			// the only direct participant is the source_account
		case xdr.OperationTypeCreateClaimableBalance:
			for _, c := range operation.Body.MustCreateClaimableBalanceOp().Claimants {
				participants = append(participants, c.MustV0().Destination.Address())
			}
		case xdr.OperationTypeClaimClaimableBalance:
			// the only direct participant is the source_account
		case xdr.OperationTypeBeginSponsoringFutureReserves:
			participants = append(participants, operation.Body.MustBeginSponsoringFutureReservesOp().SponsoredId.Address())
		case xdr.OperationTypeEndSponsoringFutureReserves:
			// Failed transactions may not have a compliant sandwich structure
			// we can rely on (e.g. invalid nesting or a being operation with the wrong sponsoree ID)
			// and thus we bail out since we could return incorrect information.
			if transaction.Result.Successful() {
				sponsoree := transaction.Envelope.SourceAccount().ToAccountId().Address()
				if operation.SourceAccount != nil {
					sponsoree = operation.SourceAccount.Address()
				}
				operations := transaction.Envelope.Operations()
				for i := int(opindex) - 1; i >= 0; i-- {
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
				// We don't add signer as a participant because a signer can be arbitrary account.
				// This can spam successful operations history of any account.
			}
		case xdr.OperationTypeClawback:
			op := operation.Body.MustClawbackOp()
			participants = append(participants, op.From.ToAccountId().Address())
		case xdr.OperationTypeClawbackClaimableBalance:
			// the only direct participant is the source_account
		case xdr.OperationTypeSetTrustLineFlags:
			op := operation.Body.MustSetTrustLineFlagsOp()
			participants = append(participants, op.Trustor.Address())
		case xdr.OperationTypeLiquidityPoolDeposit:
			// the only direct participant is the source_account
		case xdr.OperationTypeLiquidityPoolWithdraw:
			// the only direct participant is the source_account
		default:
			return nil, fmt.Errorf("unknown operation type: %s", operation.Body.Type)
		}

		// Requires meta
		// sponsor, err := operation.getSponsor()
		// if err != nil {
		// 	return nil, err
		// }
		// if sponsor != nil {
		// 	otherParticipants = append(otherParticipants, *sponsor)
		// }
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
