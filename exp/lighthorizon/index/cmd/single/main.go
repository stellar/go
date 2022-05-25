package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
	"golang.org/x/sync/errgroup"
)

func main() {
	sourceUrl := flag.String("source", "gcs://horizon-archive-poc", "history archive url to read txmeta files")
	targetUrl := flag.String("target", "file://indexes", "where to write indexes")
	networkPassphrase := flag.String("network-passphrase", network.TestNetworkPassphrase, "network passphrase")
	start := flag.Int("start", -1, "ledger to start at (inclusive, default: earliest)")
	end := flag.Int("end", -1, "ledger to end at (inclusive, default: latest)")
	modules := flag.String("modules", "accounts,transactions", "comma-separated list of modules to index (default: all)")

	// Should we use runtime.NumCPU() for a reasonable default?
	// Yes, but leave a CPU open so I can actually use my PC while this runs.
	workerCount := flag.Int("workers", runtime.NumCPU()-1, "number of workers (default: # of CPUs - 1)")

	flag.Parse()
	log.SetLevel(log.InfoLevel)

	ctx := context.Background()

	indexStore, err := index.Connect(*targetUrl)
	if err != nil {
		panic(err)
	}

	// Simple file os access
	source, err := historyarchive.ConnectBackend(
		*sourceUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: *networkPassphrase,
		},
	)
	if err != nil {
		panic(err)
	}
	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(source)
	defer ledgerBackend.Close()

	startTime := time.Now()

	startLedger := uint32(max(*start, 2))
	endLedger := uint32(*end)
	if endLedger < 0 {
		latest, err := ledgerBackend.GetLatestLedgerSequence(ctx)
		if err != nil {
			panic(err)
		}
		endLedger = latest
	}
	ledgerCount := 1 + (endLedger - startLedger) // +1 because endLedger is inclusive
	parallel := max(1, *workerCount)

	log.Infof("Creating indices for ledger range: %d through %d (%d ledgers)",
		startLedger, endLedger, ledgerCount)
	log.Infof("Using %d workers", parallel)

	// Create a bunch of workers that process ledgers a checkpoint range at a
	// time (better than a ledger at a time to minimize flushes).
	type work struct {
		startLedger, endLedger uint32
	}
	wg, ctx := errgroup.WithContext(ctx)
	ch := make(chan work, parallel)

	// Submit the work to the channels, breaking up the range into checkpoints.
	go func() {
		// Recall: A ledger X is a checkpoint ledger iff (X + 1) % 64 == 0
		nextCheckpoint := (((startLedger / 64) * 64) + 63)

		ledger := startLedger
		nextLedger := ledger + (nextCheckpoint - startLedger)
		for ledger <= endLedger {
			ch <- work{ledger, nextLedger}

			ledger = nextLedger + 1
			// Ensure we don't exceed the upper ledger bound
			nextLedger = uint32(min(int(endLedger), int(ledger+63)))
		}

		close(ch)
	}()

	processed := uint64(0)
	for i := 0; i < parallel; i++ {
		wg.Go(func() error {
			for ledgerRange := range ch {
				count := (ledgerRange.endLedger - ledgerRange.startLedger) + 1
				nprocessed := atomic.AddUint64(&processed, uint64(count))

				log.Debugf("Working on checkpoint range [%d, %d]",
					ledgerRange.startLedger, ledgerRange.endLedger)

				// Assertion for testing
				if ledgerRange.endLedger != endLedger &&
					(ledgerRange.endLedger+1)%64 != 0 {
					log.Fatalf("Uh oh: bad range")
				}

				err = buildIndices(ctx, indexStore, ledgerBackend,
					*networkPassphrase,
					strings.Split(*modules, ","),
					ledgerRange.startLedger, ledgerRange.endLedger)
				if err != nil {
					return err
				}

				postProgress("Reading checkpoints",
					nprocessed, uint64(ledgerCount), startTime)

				// Upload indices once per checkpoint to save memory
				if err := indexStore.Flush(); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		panic(err)
	}

	postProgress("Reading checkpoints",
		uint64(ledgerCount), uint64(ledgerCount), startTime)

	// Assertion for testing
	if processed != uint64(ledgerCount) {
		log.Fatalf("wtf? processed %d but expected %d", processed, ledgerCount)
	}

	log.Infof("Processed %d ledgers via %d workers", processed, parallel)
	log.Infof("Uploading indices to %s", *targetUrl)
	if err := indexStore.Flush(); err != nil {
		panic(err)
	}
}

func buildIndices(
	ctx context.Context,
	indexStore index.Store,
	ledgerBackend *ledgerbackend.HistoryArchiveBackend,
	networkPassphrase string,
	modules []string,
	startLedger, endLedger uint32,
) error {
	for ledgerSeq := startLedger; ledgerSeq <= endLedger; ledgerSeq++ {
		ledger, err := ledgerBackend.GetLedger(ctx, ledgerSeq)
		if err != nil {
			log.WithField("error", err).Errorf("error getting ledger %d", ledgerSeq)
			return err
		}

		checkpoint := ledgerSeq / 64

		reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase, ledger)
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

			for _, part := range modules {
				var err error
				switch part {
				case "transactions":
					err = processTransactionModule(indexStore, ledger, tx)
				case "accounts":
					err = processAccountsModule(indexStore, checkpoint, tx)
				default:
					err = fmt.Errorf("unknown module: %s", part)
				}

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func postProgress(prefix string, done, total uint64, startTime time.Time) {
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

func processTransactionModule(
	indexStore index.Store,
	ledger xdr.LedgerCloseMeta,
	tx ingest.LedgerTransaction,
) error {
	return indexStore.AddTransactionToIndexes(
		toid.New(int32(ledger.LedgerSequence()), int32(tx.Index), 0).ToInt64(),
		tx.Result.TransactionHash,
	)
}

func processAccountsModule(
	indexStore index.Store,
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
