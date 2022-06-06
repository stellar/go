package index

import (
	"context"
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"time"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
	"golang.org/x/sync/errgroup"
)

func BuildIndices(
	ctx context.Context,
	sourceUrl string, // where is raw txmeta coming from?
	targetUrl string, // where should the resulting indices go?
	networkPassphrase string,
	startLedger, endLedger uint32,
	modules []string,
	workerCount int,
) error {
	indexStore, err := Connect(targetUrl)
	if err != nil {
		return err
	}

	// Simple file os access
	source, err := historyarchive.ConnectBackend(
		sourceUrl,
		historyarchive.ConnectOptions{
			Context:           ctx,
			NetworkPassphrase: networkPassphrase,
		},
	)
	if err != nil {
		return err
	}

	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(source)
	defer ledgerBackend.Close()

	if endLedger == 0 {
		latest, err := ledgerBackend.GetLatestLedgerSequence(ctx)
		if err != nil {
			return err
		}
		endLedger = latest
	}

	ledgerCount := 1 + (endLedger - startLedger) // +1 because endLedger is inclusive
	parallel := max(1, workerCount)

	startTime := time.Now()
	log.Infof("Creating indices for ledger range: %d through %d (%d ledgers)",
		startLedger, endLedger, ledgerCount)
	log.Infof("Using %d workers", parallel)

	// Create a bunch of workers that process ledgers a checkpoint range at a
	// time (better than a ledger at a time to minimize flushes).
	wg, ctx := errgroup.WithContext(ctx)
	ch := make(chan historyarchive.Range, parallel)

	indexBuilder := NewIndexBuilder(indexStore, ledgerBackend, networkPassphrase)
	for _, part := range modules {
		switch part {
		case "transactions":
			indexBuilder.RegisterModule(ProcessTransaction)
		case "accounts":
			indexBuilder.RegisterModule(ProcessAccounts)
		case "accounts_unbacked":
			indexBuilder.RegisterModule(ProcessAccountsWithoutBackend)
		default:
			return fmt.Errorf("Unknown module: %s", part)
		}
	}

	// Submit the work to the channels, breaking up the range into individual
	// checkpoint ranges.
	go func() {
		// Recall: A ledger X is a checkpoint ledger iff (X + 1) % 64 == 0
		nextCheckpoint := (((startLedger / 64) * 64) + 63)

		ledger := startLedger
		nextLedger := min(endLedger, ledger+(nextCheckpoint-startLedger))
		for ledger <= endLedger {
			chunk := historyarchive.Range{Low: ledger, High: nextLedger}
			log.Debugf("Submitted [%d, %d] for work", chunk.Low, chunk.High)
			ch <- chunk

			ledger = nextLedger + 1
			nextLedger = min(endLedger, ledger+63) // don't exceed upper bound
		}

		close(ch)
	}()

	processed := uint64(0)
	for i := 0; i < parallel; i++ {
		wg.Go(func() error {
			for ledgerRange := range ch {
				count := (ledgerRange.High - ledgerRange.Low) + 1
				nprocessed := atomic.AddUint64(&processed, uint64(count))

				log.Debugf("Working on checkpoint range [%d, %d]",
					ledgerRange.Low, ledgerRange.High)

				// Assertion for testing
				if ledgerRange.High != endLedger && (ledgerRange.High+1)%64 != 0 {
					log.Fatalf("Upper ledger isn't a checkpoint: %v", ledgerRange)
				}

				err = indexBuilder.Build(ctx, ledgerRange)
				if err != nil {
					return err
				}

				printProgress("Reading ledgers", nprocessed, uint64(ledgerCount), startTime)

				// Upload indices once per checkpoint to save memory
				if err := indexStore.Flush(); err != nil {
					return errors.Wrap(err, "flushing indices failed")
				}
			}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		return errors.Wrap(err, "one or more workers failed")
	}

	printProgress("Reading ledgers", uint64(ledgerCount), uint64(ledgerCount), startTime)

	// Assertion for testing
	if processed != uint64(ledgerCount) {
		log.Fatalf("processed %d but expected %d", processed, ledgerCount)
	}

	log.Infof("Processed %d ledgers via %d workers", processed, parallel)
	log.Infof("Uploading indices to %s", targetUrl)
	if err := indexStore.Flush(); err != nil {
		return errors.Wrap(err, "flushing indices failed")
	}

	return nil
}

// Module is a way to process ingested data and shove it into an index store.
type Module func(
	indexStore Store,
	ledger xdr.LedgerCloseMeta,
	transaction ingest.LedgerTransaction,
) error

// IndexBuilder contains everything needed to build indices from ledger ranges.
type IndexBuilder struct {
	store             Store
	history           *ledgerbackend.HistoryArchiveBackend
	networkPassphrase string

	modules []Module
}

func NewIndexBuilder(
	indexStore Store,
	backend *ledgerbackend.HistoryArchiveBackend,
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
	tx ingest.LedgerTransaction,
) error {
	for _, module := range builder.modules {
		if err := module(builder.store, ledger, tx); err != nil {
			return err
		}
	}

	return nil
}

// Build sequentially creates indices for each ledger in the given range based
// on the registered modules.
//
// TODO: We can probably optimize this by doing GetLedger in parallel with the
// ingestion & index building, since the network will be idle during the latter
// portion.
func (builder *IndexBuilder) Build(ctx context.Context, ledgerRange historyarchive.Range) error {
	for ledgerSeq := ledgerRange.Low; ledgerSeq <= ledgerRange.High; ledgerSeq++ {
		ledger, err := builder.history.GetLedger(ctx, ledgerSeq)
		if err != nil {
			log.WithField("error", err).Errorf("error getting ledger %d", ledgerSeq)
			return err
		}

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

			if err := builder.RunModules(ledger, tx); err != nil {
				return err
			}
		}
	}

	return nil
}

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

func ProcessAccounts(
	indexStore Store,
	ledger xdr.LedgerCloseMeta,
	tx ingest.LedgerTransaction,
) error {
	checkpoint := (ledger.LedgerSequence() / 64) + 1
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

func ProcessAccountsWithoutBackend(
	indexStore Store,
	ledger xdr.LedgerCloseMeta,
	tx ingest.LedgerTransaction,
) error {
	checkpoint := (ledger.LedgerSequence() / 64) + 1
	allParticipants, err := getParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "all_all", allParticipants)
	if err != nil {
		return err
	}

	paymentsParticipants, err := getPaymentParticipants(tx)
	if err != nil {
		return err
	}

	err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "all_payments", paymentsParticipants)
	if err != nil {
		return err
	}

	if tx.Result.Successful() {
		err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "successful_all", allParticipants)
		if err != nil {
			return err
		}

		err = indexStore.AddParticipantsToIndexesNoBackend(checkpoint, "successful_payments", paymentsParticipants)
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
		//  return nil, err
		// }
		// if sponsor != nil {
		//  otherParticipants = append(otherParticipants, *sponsor)
		// }
	}

	// FIXME: This could probably be a set rather than a list, since there's no
	// reason to track a participating account more than once if they are
	// participants across multiple operations.
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

func printProgress(prefix string, done, total uint64, startTime time.Time) {
	// This should never happen, more of a runtime assertion for now.
	// We can remove it when production-ready.
	if done > total {
		panic(fmt.Errorf("error for %s: done > total (%d > %d)",
			prefix, done, total))
	}

	progress := float64(done) / float64(total)
	elapsed := time.Since(startTime)

	// Approximate based on how many ledgers are left and how long this much
	// progress took, e.g. if 4/10 took 2s then 6/10 will "take" 3s (though this
	// assumes consistent ledger load).
	remaining := (float64(elapsed) / float64(done)) * float64(total-done)

	var remainingStr string
	if math.IsInf(remaining, 0) || math.IsNaN(remaining) {
		remainingStr = "unknown"
	} else {
		remainingStr = time.Duration(remaining).Round(time.Millisecond).String()
	}

	log.Infof("%s - %.1f%% (%d/%d) - elapsed: %s, remaining: ~%s", prefix,
		100*progress, done, total,
		elapsed.Round(time.Millisecond),
		remainingStr,
	)
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
