package ingest

import (
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
)

func TestCalculateParallelLedgerBatchSize(t *testing.T) {
	assert.Equal(t, uint32(6656), calculateParallelLedgerBatchSize(20096, 3))
	assert.Equal(t, uint32(4992), calculateParallelLedgerBatchSize(20096, 4))
	assert.Equal(t, uint32(4992), calculateParallelLedgerBatchSize(20096, 4))
	assert.Equal(t, uint32(64), calculateParallelLedgerBatchSize(64, 4))
	assert.Equal(t, uint32(64), calculateParallelLedgerBatchSize(64, 4))
	assert.Equal(t, uint32(64), calculateParallelLedgerBatchSize(2, 4))
	assert.Equal(t, uint32(20096), calculateParallelLedgerBatchSize(20096, 1))
}

func TestParallelReingestRange(t *testing.T) {
	config := Config{}
	var (
		rangesCalled []history.LedgerRange
		m            sync.Mutex
	)
	result := &mockSystem{}
	result.On("ReingestRange", mock.AnythingOfType("[]history.LedgerRange"), false, false).Run(
		func(args mock.Arguments) {
			m.Lock()
			defer m.Unlock()
			rangesCalled = append(rangesCalled, args.Get(0).([]history.LedgerRange)...)
			// simulate call
			time.Sleep(time.Millisecond * time.Duration(10+rand.Int31n(50)))
		}).Return(error(nil))
	result.On("RebuildTradeAggregationBuckets", uint32(1), uint32(2050)).Return(nil).Once()
	factory := func(c Config) (System, error) {
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange([]history.LedgerRange{{1, 2050}})
	assert.NoError(t, err)

	sort.Slice(rangesCalled, func(i, j int) bool {
		return rangesCalled[i].StartSequence < rangesCalled[j].StartSequence
	})
	expected := []history.LedgerRange{
		{StartSequence: 1, EndSequence: 640}, {StartSequence: 641, EndSequence: 1280}, {StartSequence: 1281, EndSequence: 1920}, {StartSequence: 1921, EndSequence: 2050},
	}
	assert.Equal(t, expected, rangesCalled)

	rangesCalled = nil
	system, err = newParallelSystems(config, 1, factory)
	assert.NoError(t, err)
	result.On("RebuildTradeAggregationBuckets", uint32(1), uint32(1024)).Return(nil).Once()
	err = system.ReingestRange([]history.LedgerRange{{1, 1024}})
	result.AssertExpectations(t)
	expected = []history.LedgerRange{
		{StartSequence: 1, EndSequence: 1024},
	}
	assert.NoError(t, err)
	assert.Equal(t, expected, rangesCalled)
}

func TestParallelReingestRangeError(t *testing.T) {
	config := Config{}
	result := &mockSystem{}
	// Fail on the second range
	result.On("ReingestRange", []history.LedgerRange{{641, 1280}}, false, false).Return(errors.New("failed because of foo")).Once()
	result.On("ReingestRange", mock.AnythingOfType("[]history.LedgerRange"), false, false).Return(nil)
	result.On("RebuildTradeAggregationBuckets", uint32(1), uint32(641)).Return(nil).Once()

	factory := func(c Config) (System, error) {
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange([]history.LedgerRange{{1, 2050}})
	result.AssertExpectations(t)
	assert.Error(t, err)
	assert.Equal(t, "job failed, recommended restart range: [641, 2050]: error when processing [641, 1280] range: failed because of foo", err.Error())
}

func TestParallelReingestRangeErrorInEarlierJob(t *testing.T) {
	config := Config{}
	var wg sync.WaitGroup
	wg.Add(1)
	result := &mockSystem{}
	// Fail on an lower subrange after the first error
	result.On("ReingestRange", []history.LedgerRange{{641, 1280}}, false, false).Run(func(mock.Arguments) {
		// Wait for a more recent range to error
		wg.Wait()
		// This sleep should help making sure the result of this range is processed later than the one below
		// (there are no guarantees without instrumenting ReingestRange(), but that's too complicated)
		time.Sleep(50 * time.Millisecond)
	}).Return(errors.New("failed because of foo")).Once()
	result.On("ReingestRange", []history.LedgerRange{{1281, 1920}}, false, false).Run(func(mock.Arguments) {
		wg.Done()
	}).Return(errors.New("failed because of bar")).Once()
	result.On("ReingestRange", mock.AnythingOfType("[]history.LedgerRange"), false, false).Return(error(nil))
	result.On("RebuildTradeAggregationBuckets", uint32(1), uint32(641)).Return(nil).Once()

	factory := func(c Config) (System, error) {
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange([]history.LedgerRange{{1, 2050}})
	result.AssertExpectations(t)
	assert.Error(t, err)
	assert.Equal(t, "job failed, recommended restart range: [641, 2050]: error when processing [641, 1280] range: failed because of foo", err.Error())

}
