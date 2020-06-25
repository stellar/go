package expingest

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
	factory := func(c Config) (System, error) {
		result := &mockSystem{}
		result.On("ReingestRange", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32"), mock.AnythingOfType("bool")).Run(
			func(args mock.Arguments) {
				r := ledgerRange{
					from: args.Get(0).(uint32),
					to:   args.Get(1).(uint32),
				}
				m.Lock()
				defer m.Unlock()
				rangesCalled = append(rangesCalled, r)
				// simulate call
				time.Sleep(time.Millisecond * time.Duration(10+rand.Int31n(50)))
			}).Return(error(nil))
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange(0, 2050, 258)
	assert.NoError(t, err)

	sort.Sort(rangesCalled)
	expected := sorteableRanges{
		{from: 0, to: 255}, {from: 256, to: 511}, {from: 512, to: 767}, {from: 768, to: 1023}, {from: 1024, to: 1279},
		{from: 1280, to: 1535}, {from: 1536, to: 1791}, {from: 1792, to: 2047}, {from: 2048, to: 2050},
	}
	assert.Equal(t, expected, rangesCalled)

}

func TestParallelReingestRangeError(t *testing.T) {
	config := Config{}
	factory := func(c Config) (System, error) {
		result := &mockSystem{}
		// Fail on the second range
		result.On("ReingestRange", uint32(1536), uint32(1791), mock.AnythingOfType("bool")).Return(errors.New("failed because of foo"))
		result.On("ReingestRange", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32"), mock.AnythingOfType("bool")).Return(error(nil))
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange(0, 2050, 258)
	assert.Error(t, err)
	assert.Equal(t, "job failed, recommended restart range: [1536, 2050]: error when processing [1536, 1791] range: failed because of foo", err.Error())

}

func TestParallelReingestRangeErrorInEarlierJob(t *testing.T) {
	config := Config{}
	var wg sync.WaitGroup
	wg.Add(1)
	factory := func(c Config) (System, error) {
		result := &mockSystem{}
		// Fail on the second range
		result.On("ReingestRange", uint32(1024), uint32(1279), mock.AnythingOfType("bool")).Run(func(mock.Arguments) {
			// Wait for a more recent range to error
			wg.Wait()
			// This sleep should help making sure the result of this range is processed later than the earlier ones
			// (there are no guarantees without instrumenting ReingestRange(), but that's too complicated)
			time.Sleep(50 * time.Millisecond)
		}).Return(errors.New("failed because of foo"))
		result.On("ReingestRange", uint32(1536), uint32(1791), mock.AnythingOfType("bool")).Run(func(mock.Arguments) {
			wg.Done()
		}).Return(errors.New("failed because of bar"))
		result.On("ReingestRange", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32"), mock.AnythingOfType("bool")).Return(error(nil))
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange(0, 2050, 258)
	assert.Error(t, err)
	assert.Equal(t, "job failed, recommended restart range: [1024, 2050]: error when processing [1024, 1279] range: failed because of foo", err.Error())

}
