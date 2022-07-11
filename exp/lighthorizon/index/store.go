package index

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	backend "github.com/stellar/go/exp/lighthorizon/index/backend"
	types "github.com/stellar/go/exp/lighthorizon/index/types"
	"github.com/stellar/go/support/log"
)

type Store interface {
	NextActive(account, index string, afterCheckpoint uint32) (uint32, error)
	TransactionTOID(hash [32]byte) (int64, error)

	AddTransactionToIndexes(txnTOID int64, hash [32]byte) error
	AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error
	AddParticipantsToIndexesNoBackend(checkpoint uint32, index string, participants []string) error
	AddParticipantToIndexesNoBackend(participant string, indexes types.NamedIndices)

	Flush() error
	FlushAccounts() error
	ClearMemory(bool)

	Read(account string) (types.NamedIndices, error)
	ReadAccounts() ([]string, error)
	ReadTransactions(prefix string) (*types.TrieIndex, error)

	MergeTransactions(prefix string, other *types.TrieIndex) error

	RegisterMetrics(registry *prometheus.Registry)
}

type StoreConfig struct {
	// init time config
	Url     string
	Workers uint32

	// runtime config
	ClearMemoryOnFlush bool

	// logging & metrics
	Log     *log.Entry // TODO: unused for now
	Metrics *prometheus.Registry
}

type store struct {
	mutex  sync.RWMutex
	config StoreConfig

	// data
	indexes   map[string]types.NamedIndices
	txIndexes map[string]*types.TrieIndex
	backend   backend.Backend

	// metrics
	indexWorkingSet     prometheus.Gauge
	indexWorkingSetTime prometheus.Gauge // to check if the above takes too long lmao
}

func NewStore(backend backend.Backend, config StoreConfig) (Store, error) {
	result := &store{
		indexes:   map[string]types.NamedIndices{},
		txIndexes: map[string]*types.TrieIndex{},
		backend:   backend,

		config: config,

		indexWorkingSet: newHorizonLiteGauge("working_set",
			"Approximately how much memory (kiB) are indices using?"),
		indexWorkingSetTime: newHorizonLiteGauge("working_set_time",
			"How long did it take (μs) to calculate the working set size?"),
	}
	result.RegisterMetrics(config.Metrics)

	return result, nil
}

func (s *store) accounts() []string {
	accounts := make([]string, 0, len(s.indexes))
	for account := range s.indexes {
		accounts = append(accounts, account)
	}
	return accounts
}

func (s *store) FlushAccounts() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.backend.FlushAccounts(s.accounts())
}

func (s *store) Read(account string) (types.NamedIndices, error) {
	return s.backend.Read(account)
}

func (s *store) ReadAccounts() ([]string, error) {
	return s.backend.ReadAccounts()
}

func (s *store) ReadTransactions(prefix string) (*types.TrieIndex, error) {
	return s.getCreateTrieIndex(prefix)
}

func (s *store) MergeTransactions(prefix string, other *types.TrieIndex) error {
	defer s.approximateWorkingSet()

	index, err := s.getCreateTrieIndex(prefix)
	if err != nil {
		return err
	}
	if err := index.Merge(other); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.txIndexes[prefix] = index
	return nil
}

func (s *store) approximateWorkingSet() {
	if s.config.Metrics == nil {
		return
	}

	start := time.Now()
	approx := float64(0)

	for _, indices := range s.indexes {
		firstIndexSize := 0
		for _, index := range indices {
			firstIndexSize = index.Size()
			break
		}

		// There may be multiple indices for each account, but we can do a rough
		// approximation for now by just assuming they're all around the same
		// size.
		approx += float64(len(indices) * firstIndexSize)
	}

	for _, trie := range s.txIndexes {
		// FIXME: Is this too slow? We probably want a TrieIndex.Size() method,
		// but that's not trivial to determine for a trie.
		trie.Iterate(func(key, value []byte) {
			approx += float64(len(key) + len(value))
		})
	}

	s.indexWorkingSet.Set(approx / 1024)                                 // kiB
	s.indexWorkingSetTime.Set(float64(time.Since(start).Microseconds())) // μs
}

func (s *store) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.approximateWorkingSet()

	if err := s.backend.Flush(s.indexes); err != nil {
		return err
	}

	if err := s.backend.FlushAccounts(s.accounts()); err != nil {
		return err
	} else if s.config.ClearMemoryOnFlush {
		s.indexes = map[string]types.NamedIndices{}
	}

	if err := s.backend.FlushTransactions(s.txIndexes); err != nil {
		return err
	} else if s.config.ClearMemoryOnFlush {
		s.txIndexes = map[string]*types.TrieIndex{}
	}

	return nil
}

func (s *store) ClearMemory(doClear bool) {
	s.config.ClearMemoryOnFlush = doClear
}

func (s *store) AddTransactionToIndexes(txnTOID int64, hash [32]byte) error {
	index, err := s.getCreateTrieIndex(hex.EncodeToString(hash[:1]))
	if err != nil {
		return err
	}

	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, uint64(txnTOID))

	// We don't have to re-calculate the whole working set size for metrics
	// since we're adding a known size.
	if _, replaced := index.Upsert(hash[1:], value); !replaced {
		s.indexWorkingSet.Add(float64(len(hash) - 1 + len(value)))
	}

	return nil
}

func (s *store) TransactionTOID(hash [32]byte) (int64, error) {
	index, err := s.getCreateTrieIndex(hex.EncodeToString(hash[:1]))
	if err != nil {
		return 0, err
	}

	value, ok := index.Get(hash[1:])
	if !ok {
		return 0, io.EOF
	}
	return int64(binary.BigEndian.Uint64(value)), nil
}

// AddParticipantsToIndexesNoBackend is a temp version of
// AddParticipantsToIndexes that skips backend downloads and it used in AWS
// Batch. Refactoring required to make it better.
func (s *store) AddParticipantsToIndexesNoBackend(checkpoint uint32, index string, participants []string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.approximateWorkingSet()

	var err error
	for _, participant := range participants {
		if _, ok := s.indexes[participant]; !ok {
			s.indexes[participant] = map[string]*types.CheckpointIndex{}
		}

		ind, ok := s.indexes[participant][index]
		if !ok {
			ind = &types.CheckpointIndex{}
			s.indexes[participant][index] = ind
		}

		if innerErr := ind.SetActive(checkpoint); innerErr != nil {
			err = innerErr
		}
		// don't break early, instead try to save as many participants as we can
	}

	return err
}

func (s *store) AddParticipantToIndexesNoBackend(participant string, indexes types.NamedIndices) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.approximateWorkingSet()

	s.indexes[participant] = indexes
}

func (s *store) AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error {
	defer s.approximateWorkingSet()

	for _, participant := range participants {
		ind, err := s.getCreateIndex(participant, index)
		if err != nil {
			return err
		}
		err = ind.SetActive(checkpoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *store) getCreateIndex(account, id string) (*types.CheckpointIndex, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.approximateWorkingSet()

	// Check if we already have it loaded
	accountIndexes, ok := s.indexes[account]
	if !ok {
		accountIndexes = types.NamedIndices{}
	}
	ind, ok := accountIndexes[id]
	if ok {
		return ind, nil
	}

	// Check if index exists in backend
	found, err := s.backend.Read(account)
	if err == nil {
		accountIndexes = found
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	ind, ok = accountIndexes[id]
	if !ok {
		// Not found anywhere, make a new one.
		ind = &types.CheckpointIndex{}
		accountIndexes[id] = ind
	}

	// We don't want to replace the entire index map in memory (even though we
	// read all of it from disk), just the one we loaded from disk. Otherwise,
	// we lose in-memory changes to unrelated indices.
	if memoryIndices, ok := s.indexes[account]; ok { // account exists in-mem
		if memoryIndex, ok2 := memoryIndices[id]; ok2 { // id exists in-mem
			if memoryIndex != accountIndexes[id] { // not using in-mem already
				memoryIndex.Merge(ind)
				s.indexes[account][id] = memoryIndex
			}
		}
	} else {
		s.indexes[account] = accountIndexes
	}

	return ind, nil
}

func (s *store) NextActive(account, indexId string, afterCheckpoint uint32) (uint32, error) {
	defer s.approximateWorkingSet()

	ind, err := s.getCreateIndex(account, indexId)
	if err != nil {
		return 0, err
	}
	return ind.NextActive(afterCheckpoint)
}

func (s *store) getCreateTrieIndex(prefix string) (*types.TrieIndex, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.approximateWorkingSet()

	// Check if we already have it loaded
	index, ok := s.txIndexes[prefix]
	if ok {
		return index, nil
	}

	// Check if index exists in backend
	found, err := s.backend.ReadTransactions(prefix)
	if err == nil {
		s.txIndexes[prefix] = found
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	index, ok = s.txIndexes[prefix]
	if !ok {
		// Not found anywhere, make a new one.
		index = &types.TrieIndex{}
		s.txIndexes[prefix] = index
	}

	return index, nil
}

func (s *store) RegisterMetrics(registry *prometheus.Registry) {
	s.config.Metrics = registry

	if registry != nil {
		registry.Register(s.indexWorkingSet)
		registry.Register(s.indexWorkingSetTime)
	}
}

func newHorizonLiteGauge(name, help string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon_lite",
		Subsystem: "index_store",
		Name:      name,
		Help:      help,
	})
}
