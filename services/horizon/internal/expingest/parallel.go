package expingest

import (
	"fmt"
	"sync"

	"github.com/stellar/go/support/errors"
)

const (
	historyCheckpointLedgerInterval = 64
	minBatchSize                    = historyCheckpointLedgerInterval
)

type ledgerRange struct {
	from uint32
	to   uint32
}

type rangeError struct {
	err         error
	ledgerRange ledgerRange
}

func (e rangeError) Error() string {
	return fmt.Sprintf("error when processing [%d, %d] range: %s", e.ledgerRange.from, e.ledgerRange.to, e.err)
}

type ParallelSystems struct {
	config        Config
	workerCount   uint
	systemFactory func(Config) (System, error)
}

func NewParallelSystems(config Config, workerCount uint) (*ParallelSystems, error) {
	// Leaving this because used in tests, will update after a code review.
	return newParallelSystems(config, workerCount, NewSystem)
}

// private version of NewParallel systems, allowing to inject a mock system
func newParallelSystems(config Config, workerCount uint, systemFactory func(Config) (System, error)) (*ParallelSystems, error) {
	if workerCount < 1 {
		return nil, errors.New("workerCount must be > 0")
	}

	return &ParallelSystems{
		config:        config,
		workerCount:   workerCount,
		systemFactory: systemFactory,
	}, nil
}

func (ps *ParallelSystems) runReingestWorker(s System, stop <-chan struct{}, reingestJobQueue <-chan ledgerRange) rangeError {
	for {
		select {
		case <-stop:
			return rangeError{}
		case reingestRange := <-reingestJobQueue:
			err := s.ReingestRange(reingestRange.from, reingestRange.to, false)
			if err != nil {
				return rangeError{
					err:         err,
					ledgerRange: reingestRange,
				}
			}
		}
	}
}

func calculateParallelLedgerBatchSize(rangeSize uint32, batchSizeSuggestion uint32, workerCount uint) uint32 {
	batchSize := batchSizeSuggestion
	if batchSize == 0 || rangeSize/batchSize < uint32(workerCount) {
		// let's try to make use of all the workers
		batchSize = rangeSize / uint32(workerCount)
	}
	// Use a minimum batch size to make it worth it in terms of overhead
	if batchSize < minBatchSize {
		batchSize = minBatchSize
	}

	// Also, round the batch size to the closest, lower or equal 64 multiple
	return (batchSize / historyCheckpointLedgerInterval) * historyCheckpointLedgerInterval
}

func (ps *ParallelSystems) ReingestRange(fromLedger, toLedger uint32, batchSizeSuggestion uint32) error {
	var (
		batchSize        = calculateParallelLedgerBatchSize(toLedger-fromLedger, batchSizeSuggestion, ps.workerCount)
		reingestJobQueue = make(chan ledgerRange)
		wg               sync.WaitGroup

		// stopOnce is used to close the stop channel once: closing a closed channel panics and it can happen in case
		// of errors in multiple ranges.
		stopOnce sync.Once
		stop     = make(chan struct{})

		lowestRangeErrMutex sync.Mutex
		// lowestRangeErr is an error of the failed range with the lowest starting ledger sequence that is used to tell
		// the user which range to reingest in case of errors. We use that fact that System.ReingestRange is blocking,
		// jobs are sent to a queue (unbuffered channel) in sequence and there is a WaitGroup waiting for all the workers
		// to exit.
		// Because of this when we reach `wg.Wait()` all jobs previously sent to a channel are processed (either success
		// or failure). In case of a failure we save the range with the smallest sequence number because this is where
		// the user needs to start again to prevent the gaps.
		lowestRangeErr *rangeError
	)

	for i := uint(0); i < ps.workerCount; i++ {
		wg.Add(1)
		s, err := ps.systemFactory(ps.config)
		if err != nil {
			return errors.Wrap(err, "error creating new system")
		}
		go func() {
			defer wg.Done()
			rangeErr := ps.runReingestWorker(s, stop, reingestJobQueue)
			if rangeErr.err != nil {
				log.WithError(err).Error("error in reingest worker")
				lowestRangeErrMutex.Lock()
				if lowestRangeErr == nil || lowestRangeErr.ledgerRange.from > rangeErr.ledgerRange.from {
					lowestRangeErr = &rangeErr
				}
				lowestRangeErrMutex.Unlock()
				stopOnce.Do(func() {
					close(stop)
				})
				return
			}
		}()
	}

rangeQueueLoop:
	for subRangeFrom := fromLedger; subRangeFrom < toLedger; {
		// job queuing
		subRangeTo := subRangeFrom + (batchSize - 1) // we subtract one because both from and to are part of the batch
		if subRangeTo > toLedger {
			subRangeTo = toLedger
		}
		select {
		case <-stop:
			break rangeQueueLoop
		case reingestJobQueue <- ledgerRange{subRangeFrom, subRangeTo}:
		}
		subRangeFrom = subRangeTo + 1
	}

	stopOnce.Do(func() {
		close(stop)
	})
	wg.Wait()
	close(reingestJobQueue)

	if lowestRangeErr != nil {
		return errors.Wrapf(lowestRangeErr, "job failed, recommended restart range: [%d, %d]", lowestRangeErr.ledgerRange.from, toLedger)
	}
	return nil
}
