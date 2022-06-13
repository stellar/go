package index

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

func BuildIndices(
	ctx context.Context,
	sourceUrl string, // where is raw txmeta coming from?
	targetUrl string, // where should the resulting indices go?
	networkPassphrase string,
	startLedger, endLedger uint32,
	modules []string,
	workerCount int,
) (*IndexBuilder, error) {
	if endLedger < startLedger {
		log.Warnf("Nothing to do: start > end (%d > %d). "+
			"Was this a mistake?", startLedger, endLedger)
		return nil, nil
	}

	indexStore, err := Connect(targetUrl)
	if err != nil {
		return nil, err
	}

	// We use historyarchive as a backend here just to abstract away dealing
	// with the filesystem directly.
	source, err := historyarchive.ConnectBackend(
		sourceUrl,
		historyarchive.ConnectOptions{
			Context:           ctx,
			NetworkPassphrase: networkPassphrase,
		},
	)
	if err != nil {
		return nil, err
	}

	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(source)
	defer ledgerBackend.Close()

	if endLedger == 0 {
		latest, err := ledgerBackend.GetLatestLedgerSequence(ctx)
		if err != nil {
			return nil, err
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
			return indexBuilder, fmt.Errorf("Unknown module: %s", part)
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
		return indexBuilder, errors.Wrap(err, "one or more workers failed")
	}

	printProgress("Reading ledgers", uint64(ledgerCount), uint64(ledgerCount), startTime)

	// Assertion for testing
	if processed != uint64(ledgerCount) {
		log.Fatalf("processed %d but expected %d", processed, ledgerCount)
	}

	log.Infof("Processed %d ledgers via %d workers", processed, parallel)
	log.Infof("Uploading indices to %s", targetUrl)
	if err := indexStore.Flush(); err != nil {
		return indexBuilder, errors.Wrap(err, "flushing indices failed")
	}

	return indexBuilder, nil
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
	ledgerBackend     ledgerbackend.LedgerBackend
	networkPassphrase string
	lastBuiltLedger   uint32

	modules []Module
}

func NewIndexBuilder(
	indexStore Store,
	backend ledgerbackend.LedgerBackend,
	networkPassphrase string,
) *IndexBuilder {
	return &IndexBuilder{
		store:             indexStore,
		ledgerBackend:     backend,
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
		ledger, err := builder.ledgerBackend.GetLedger(ctx, ledgerSeq)
		if err != nil {
			if !os.IsNotExist(err) {
				log.WithField("error", err).Errorf("error getting ledger %d", ledgerSeq)
			}
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

	builder.lastBuiltLedger = uint32(
		max(int(builder.lastBuiltLedger),
			int(ledgerRange.High)),
	)
	return nil
}

func (b *IndexBuilder) GetStore() Store {
	return b.store
}

func (b *IndexBuilder) Watch() error {
	return nil
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
