package ingest

import (
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/support/errors"
)

func TestCalculateParallelLedgerBatchSize(t *testing.T) {
	assert.Equal(t, uint32(6656), calculateParallelLedgerBatchSize(20096, 20096, 3))
	assert.Equal(t, uint32(4992), calculateParallelLedgerBatchSize(20096, 20096, 4))
	assert.Equal(t, uint32(4992), calculateParallelLedgerBatchSize(20096, 0, 4))
	assert.Equal(t, uint32(64), calculateParallelLedgerBatchSize(64, 256, 4))
	assert.Equal(t, uint32(64), calculateParallelLedgerBatchSize(64, 32, 4))
	assert.Equal(t, uint32(64), calculateParallelLedgerBatchSize(2, 256, 4))
	assert.Equal(t, uint32(64), calculateParallelLedgerBatchSize(20096, 64, 1))
}

type sorteableRanges []ledgerRange

func (s sorteableRanges) Len() int           { return len(s) }
func (s sorteableRanges) Less(i, j int) bool { return s[i].from < s[j].from }
func (s sorteableRanges) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func TestParallelReingestRange(t *testing.T) {
	config := Config{}
	var (
		rangesCalled sorteableRanges
		m            sync.Mutex
	)
	result := &mockSystem{}
	result.On("ReingestRange", mock.AnythingOfType("[][2]uint32"), mock.AnythingOfType("bool")).Run(
		func(args mock.Arguments) {
			m.Lock()
			defer m.Unlock()
			for _, pair := range args.Get(0).([][2]uint32) {
				rangesCalled = append(rangesCalled, ledgerRange{from: pair[0], to: pair[1]})
			}
			// simulate call
			time.Sleep(time.Millisecond * time.Duration(10+rand.Int31n(50)))
		}).Return(error(nil))
	factory := func(c Config) (System, error) {
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange([][2]uint32{{1, 2050}}, 258)
	assert.NoError(t, err)

	sort.Sort(rangesCalled)
	expected := sorteableRanges{
		{from: 1, to: 256}, {from: 257, to: 512}, {from: 513, to: 768}, {from: 769, to: 1024}, {from: 1025, to: 1280},
		{from: 1281, to: 1536}, {from: 1537, to: 1792}, {from: 1793, to: 2048}, {from: 2049, to: 2050},
	}
	assert.Equal(t, expected, rangesCalled)

	rangesCalled = nil
	system, err = newParallelSystems(config, 1, factory)
	assert.NoError(t, err)
	err = system.ReingestRange([][2]uint32{{1, 1024}}, 64)
	result.AssertExpectations(t)
	expected = sorteableRanges{
		{from: 1, to: 64}, {from: 65, to: 128}, {from: 129, to: 192}, {from: 193, to: 256}, {from: 257, to: 320},
		{from: 321, to: 384}, {from: 385, to: 448}, {from: 449, to: 512}, {from: 513, to: 576}, {from: 577, to: 640},
		{from: 641, to: 704}, {from: 705, to: 768}, {from: 769, to: 832}, {from: 833, to: 896}, {from: 897, to: 960},
		{from: 961, to: 1024},
	}
	assert.NoError(t, err)
	assert.Equal(t, expected, rangesCalled)
}

func TestParallelReingestRangeError(t *testing.T) {
	config := Config{}
	result := &mockSystem{}
	// Fail on the second range
	result.On("ReingestRange", [][2]uint32{{1537, 1792}}, mock.AnythingOfType("bool")).Return(errors.New("failed because of foo"))
	result.On("ReingestRange", mock.AnythingOfType("[][2]uint32"), mock.AnythingOfType("bool")).Return(error(nil))
	factory := func(c Config) (System, error) {
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange([][2]uint32{{1, 2050}}, 258)
	result.AssertExpectations(t)
	assert.Error(t, err)
	assert.Equal(t, "job failed, recommended restart range: [1537, 2050]: error when processing [1537, 1792] range: failed because of foo", err.Error())
}

func TestParallelReingestRangeErrorInEarlierJob(t *testing.T) {
	config := Config{}
	var wg sync.WaitGroup
	wg.Add(1)
	result := &mockSystem{}
	// Fail on an lower subrange after the first error
	result.On("ReingestRange", [][2]uint32{{1025, 1280}}, mock.AnythingOfType("bool")).Run(func(mock.Arguments) {
		// Wait for a more recent range to error
		wg.Wait()
		// This sleep should help making sure the result of this range is processed later than the one below
		// (there are no guarantees without instrumenting ReingestRange(), but that's too complicated)
		time.Sleep(50 * time.Millisecond)
	}).Return(errors.New("failed because of foo"))
	result.On("ReingestRange", [][2]uint32{{1537, 1792}}, mock.AnythingOfType("bool")).Run(func(mock.Arguments) {
		wg.Done()
	}).Return(errors.New("failed because of bar"))
	result.On("ReingestRange", mock.AnythingOfType("[][2]uint32"), mock.AnythingOfType("bool")).Return(error(nil))

	factory := func(c Config) (System, error) {
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange([][2]uint32{{1, 2050}}, 258)
	result.AssertExpectations(t)
	assert.Error(t, err)
	assert.Equal(t, "job failed, recommended restart range: [1025, 2050]: error when processing [1025, 1280] range: failed because of foo", err.Error())

}
