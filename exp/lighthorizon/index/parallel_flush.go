package index

import (
	"sync"
	"sync/atomic"

	"github.com/stellar/go/support/log"
)

type batch struct {
	account string
	indexes map[string]*CheckpointIndex
}

type retry func(b *batch)

type flushBatch func(b *batch, r retry) error

func parallelFlush(parallel uint32, allIndexes map[string]map[string]*CheckpointIndex, f flushBatch) error {
	var wg sync.WaitGroup

	batches := make(chan *batch, parallel)

	retry := func(b *batch) {
		batches <- b
	}

	go func() {
		for account, indexes := range allIndexes {
			retry(&batch{
				account: account,
				indexes: indexes,
			})
		}
		close(batches)
	}()

	written := uint64(0)
	for i := uint32(0); i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range batches {
				if err := f(batch, retry); err != nil {
					log.Error(err)
					continue
				}

				nwritten := atomic.AddUint64(&written, 1)
				if nwritten%1000 == 0 {
					log.Infof("Writing indexes... %d/%d %.2f%%", nwritten, len(allIndexes), (float64(nwritten)/float64(len(allIndexes)))*100)
				}
			}
		}()
	}

	wg.Wait()

	return nil
}
