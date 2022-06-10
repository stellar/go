package index

import (
	"sync"
	"sync/atomic"
	"time"

	types "github.com/stellar/go/exp/lighthorizon/index/types"
	"github.com/stellar/go/support/log"
)

type batch struct {
	account string
	indexes types.NamedIndices
}

type flushBatch func(b *batch) error

func parallelFlush(parallel uint32, allIndexes map[string]types.NamedIndices, f flushBatch) error {
	var wg sync.WaitGroup

	batches := make(chan *batch, parallel)

	go func() {
		for account, indexes := range allIndexes {
			batches <- &batch{
				account: account,
				indexes: indexes,
			}
		}

		if len(allIndexes) == 0 {
			close(batches)
		}
	}()

	written := uint64(0)
	for i := uint32(0); i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range batches {
				if err := f(batch); err != nil {
					log.Error(err)
					time.Sleep(50 * time.Millisecond)
					batches <- batch
					continue
				}

				nwritten := atomic.AddUint64(&written, 1)
				if nwritten%1000 == 0 {
					log.Infof("Writing indexes... %d/%d %.2f%%", nwritten, len(allIndexes), (float64(nwritten)/float64(len(allIndexes)))*100)
				}

				if nwritten == uint64(len(allIndexes)) {
					close(batches)
				}
			}
		}()
	}

	wg.Wait()

	return nil
}
