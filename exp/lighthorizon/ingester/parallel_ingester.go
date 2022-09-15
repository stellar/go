package ingester

import (
	"context"
	"sync"
	"time"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/metaarchive"
	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

type parallelIngester struct {
	liteIngester

	ledgerFeed  sync.Map // thread-safe version of map[uint32]downloadState
	ledgerQueue set.ISet[uint32]

	workQueue  chan uint32
	signalChan chan error
}

type downloadState struct {
	ledger xdr.SerializedLedgerCloseMeta
	err    error
}

// NewParallelIngester creates an ingester on the given `ledgerSource` using the
// given `networkPassphrase` that can download ledgers in parallel via
// `workerCount` workers via `PrepareRange()`.
func NewParallelIngester(
	archive metaarchive.MetaArchive,
	networkPassphrase string,
	workerCount uint,
) *parallelIngester {
	self := &parallelIngester{
		liteIngester: liteIngester{
			MetaArchive:       archive,
			networkPassphrase: networkPassphrase,
		},
		ledgerFeed:  sync.Map{},
		ledgerQueue: set.NewSafeSet[uint32](64),
		workQueue:   make(chan uint32, workerCount),
		signalChan:  make(chan error),
	}

	// These are the workers that download & store ledgers in memory.
	for j := uint(0); j < workerCount; j++ {
		go func(jj uint) {
			for ledgerSeq := range self.workQueue {
				start := time.Now()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				txmeta, err := self.liteIngester.GetLedger(ctx, ledgerSeq)
				cancel()

				log.WithField("duration", time.Since(start)).
					WithField("worker", jj).WithError(err).
					Debugf("Downloaded ledger %d", ledgerSeq)

				self.ledgerFeed.Store(ledgerSeq, downloadState{txmeta, err})
				self.signalChan <- err
			}
		}(j)
	}

	return self
}

// PrepareRange will create a set of parallel worker routines that feed ledgers
// to a channel in the order they're downloaded and store the results in an
// array. You can use this to download ledgers in parallel to fetching them
// individually via `GetLedger()`. `PrepareRange()` is thread-safe.
//
// Note: The passed in range `r` is inclusive of the boundaries.
func (i *parallelIngester) PrepareRange(ctx context.Context, r historyarchive.Range) error {
	// The taskmaster adds ledger sequence numbers to the work queue.
	go func() {
		start := time.Now()
		defer func() {
			log.WithField("duration", time.Since(start)).
				WithError(ctx.Err()).
				Infof("Download of ledger range: [%d, %d] (%d ledgers) complete",
					r.Low, r.High, r.Size())
		}()

		for seq := r.Low; seq <= r.High; seq++ {
			if ctx.Err() != nil {
				log.Warnf("Cancelling remaining downloads ([%d, %d]): %v",
					seq, r.High, ctx.Err())
				break
			}

			// Adding this to the "set of ledgers being downloaded in parallel"
			// means that if a GetLedger() request happens in this range but
			// outside of the realm of processing, it can be prioritized by the
			// normal, direct download.
			i.ledgerQueue.Add(seq)

			i.workQueue <- seq // blocks until there's an available worker

			// We don't remove from the queue here, preferring to remove when
			// it's actually pulled from the worker. Removing here would mean
			// you could have multiple instances of a ledger download happening.
		}
	}()

	return nil
}

func (i *parallelIngester) GetLedger(
	ctx context.Context, ledgerSeq uint32,
) (xdr.SerializedLedgerCloseMeta, error) {
	// If the requested ledger is out of the queued up ranges, we can fall back
	// to the default non-parallel download method.
	if !i.ledgerQueue.Contains(ledgerSeq) {
		return i.liteIngester.GetLedger(ctx, ledgerSeq)
	}

	// If the ledger isn't available yet, wait for the download worker.
	var err error
	for err == nil {
		if iState, ok := i.ledgerFeed.Load(ledgerSeq); ok {
			state := iState.(downloadState)
			i.ledgerFeed.Delete(ledgerSeq)
			i.ledgerQueue.Remove(ledgerSeq)
			return state.ledger, state.err
		}

		select {
		case err = <-i.signalChan: // blocks until another ledger downloads
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	return xdr.SerializedLedgerCloseMeta{}, err
}

var _ Ingester = (*parallelIngester)(nil) // ensure conformity to the interface
