package expingest

import (
	"fmt"
	"math"
	"sync"

	"github.com/stellar/go/support/errors"
)

type ledgerRange struct {
	from uint32
	to   uint32
}

type rangeResult struct {
	err            error
	requestedRange ledgerRange
}

type ParallelSystems struct {
	workerCount       uint
	reingestJobQueue  chan ledgerRange
	shutdown          chan struct{}
	wait              sync.WaitGroup
	reingestJobResult chan rangeResult
}

func NewParallelSystems(config Config, workerCount uint) (*ParallelSystems, error) {
	return newParallelSystems(config, workerCount, NewSystem)
}

// private version of NewParallel systems, allowing to inject a mock system
func newParallelSystems(config Config, workerCount uint, systemFactory func(Config) (System, error)) (*ParallelSystems, error) {
	if workerCount < 1 {
		return nil, errors.New("workerCount must be > 0")
	}

	result := ParallelSystems{
		workerCount:       workerCount,
		reingestJobQueue:  make(chan ledgerRange),
		shutdown:          make(chan struct{}),
		reingestJobResult: make(chan rangeResult),
	}
	for i := uint(0); i < workerCount; i++ {
		s, err := systemFactory(config)
		if err != nil {
			result.Shutdown()
			return nil, errors.Wrap(err, "cannot create new system")
		}
		result.wait.Add(1)
		go result.work(s)
	}
	return &result, nil
}

func (ps *ParallelSystems) work(s System) {
	defer func() {
		s.Shutdown()
		ps.wait.Done()
	}()
	for {
		select {
		case <-ps.shutdown:
			return
		case reingestRange := <-ps.reingestJobQueue:
			err := s.ReingestRange(reingestRange.from, reingestRange.to, false)
			select {
			case <-ps.shutdown:
				return
			case ps.reingestJobResult <- rangeResult{err, reingestRange}:
			}
		}
	}
}

const (
	historyCheckpointLedgerInterval = 64
	minBatchSize                    = historyCheckpointLedgerInterval
)

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
		wait      sync.WaitGroup
		stop      = make(chan struct{})
		batchSize = calculateParallelLedgerBatchSize(toLedger-fromLedger, batchSizeSuggestion, ps.workerCount)
		// we add one because both toLedger and fromLedger are included in the rabge
		totalJobs = uint32(math.Ceil(float64(toLedger-fromLedger+1) / float64(batchSize)))
	)

	wait.Add(1)
	defer func() {
		close(stop)
		wait.Wait()
	}()

	// queue subranges
	go func() {
		defer wait.Done()
		for subRangeFrom := fromLedger; subRangeFrom < toLedger; {
			// job queuing
			subRangeTo := subRangeFrom + (batchSize - 1) // we subtract one because both from and to are part of the batch
			if subRangeTo > toLedger {
				subRangeTo = toLedger
			}
			select {
			case <-stop:
				return
			case ps.reingestJobQueue <- ledgerRange{subRangeFrom, subRangeTo}:
			}
			subRangeFrom = subRangeTo + 1
		}
	}()

	// collect subrange results
	for i := uint32(0); i < totalJobs; i++ {
		// collect results
		select {
		case <-ps.shutdown:
			return errors.New("aborted")
		case subRangeResult := <-ps.reingestJobResult:
			if subRangeResult.err != nil {
				// TODO: give account of what ledgers were correctly reingested?
				errMsg := fmt.Sprintf("in subrange %d to %d",
					subRangeResult.requestedRange.from, subRangeResult.requestedRange.to)
				return errors.Wrap(subRangeResult.err, errMsg)
			}
		}

	}

	return nil
}

func (ps *ParallelSystems) Shutdown() {
	if ps.shutdown != nil {
		close(ps.shutdown)
		ps.wait.Wait()
		close(ps.reingestJobQueue)
		close(ps.reingestJobResult)
	}
}
