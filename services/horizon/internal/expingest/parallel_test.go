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
		shutdowns    int
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
		result.On("Shutdown").Run(func(mock.Arguments) {
			m.Lock()
			defer m.Unlock()
			shutdowns++
		})
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange(0, 2050, 258)
	assert.NoError(t, err)

	sort.Sort(rangesCalled)
	expected := sorteableRanges{
		{from: 0, to: 256}, {from: 256, to: 512}, {from: 512, to: 768}, {from: 768, to: 1024}, {from: 1024, to: 1280},
		{from: 1280, to: 1536}, {from: 1536, to: 1792}, {from: 1792, to: 2048}, {from: 2048, to: 2050},
	}
	assert.Equal(t, expected, rangesCalled)
	system.Shutdown()
	assert.Equal(t, 3, shutdowns)

}

func TestParallelReingestRangeError(t *testing.T) {
	config := Config{}
	var (
		shutdowns int
		m         sync.Mutex
	)
	factory := func(c Config) (System, error) {
		result := &mockSystem{}
		// Fail on the second range
		result.On("ReingestRange", uint32(1536), uint32(1792), mock.AnythingOfType("bool")).Return(errors.New("failed because of foo"))
		result.On("ReingestRange", mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32"), mock.AnythingOfType("bool")).Return(error(nil))
		result.On("Shutdown").Run(func(mock.Arguments) {
			m.Lock()
			defer m.Unlock()
			shutdowns++
		})
		return result, nil
	}
	system, err := newParallelSystems(config, 3, factory)
	assert.NoError(t, err)
	err = system.ReingestRange(0, 2050, 258)
	assert.Error(t, err)
	assert.Equal(t, "in subrange 1536 to 1792: failed because of foo", err.Error())

	system.Shutdown()
	assert.Equal(t, 3, shutdowns)

}
