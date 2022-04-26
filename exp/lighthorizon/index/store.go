package index

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"sync"
)

type Store interface {
	NextActive(account, index string, afterCheckpoint uint32) (uint32, error)
	AddTransactionToIndexes(txnTOID int64, hash [32]byte) error
	TransactionTOID(hash [32]byte) (int64, error)
	AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error
	AddParticipantsToIndexesNoBackend(checkpoint uint32, index string, participants []string) error
	AddParticipantToIndexesNoBackend(participant string, indexes map[string]*CheckpointIndex)
	Flush() error
	FlushAccounts() error
	Read(account string) (map[string]*CheckpointIndex, error)
	ReadAccounts() ([]string, error)
	ReadTransactions(prefix string) (*TrieIndex, error)
	MergeTransactions(prefix string, other *TrieIndex) error
}

// TODO: Use a more standardized filesystem-style backend, so we can re-use
// code
type Backend interface {
	Flush(map[string]map[string]*CheckpointIndex) error
	FlushAccounts([]string) error
	Read(account string) (map[string]*CheckpointIndex, error)
	ReadAccounts() ([]string, error)
	FlushTransactions(map[string]*TrieIndex) error
	ReadTransactions(prefix string) (*TrieIndex, error)
}

type store struct {
	mutex     sync.RWMutex
	indexes   map[string]map[string]*CheckpointIndex
	txIndexes map[string]*TrieIndex
	backend   Backend
}

func NewStore(backend Backend) (Store, error) {
	return &store{
		indexes:   map[string]map[string]*CheckpointIndex{},
		txIndexes: map[string]*TrieIndex{},
		backend:   backend,
	}, nil
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

	if err := s.backend.FlushAccounts(s.accounts()); err != nil {
		return err
	}

	return nil
}

func (s *store) Read(account string) (map[string]*CheckpointIndex, error) {
	return s.backend.Read(account)
}

func (s *store) ReadAccounts() ([]string, error) {
	return s.backend.ReadAccounts()
}

func (s *store) ReadTransactions(prefix string) (*TrieIndex, error) {
	return s.getCreateTrieIndex(prefix)
}

func (s *store) MergeTransactions(prefix string, other *TrieIndex) error {
	index, err := s.getCreateTrieIndex(prefix)
	if err != nil {
		return err
	}
	if err := index.Merge(other); err != nil {
		return err
	}
	s.txIndexes[prefix] = index
	return nil
}

func (s *store) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.backend.Flush(s.indexes); err != nil {
		return err
	}

	// clear indexes to save memory
	s.indexes = map[string]map[string]*CheckpointIndex{}

	if err := s.backend.FlushTransactions(s.txIndexes); err != nil {
		return err
	}
	s.txIndexes = map[string]*TrieIndex{}

	return nil
}

func (s *store) AddTransactionToIndexes(txnTOID int64, hash [32]byte) error {
	index, err := s.getCreateTrieIndex(hex.EncodeToString(hash[:1]))
	if err != nil {
		return err
	}

	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, uint64(txnTOID))
	index.Upsert(hash[1:], value)

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

// AddParticipantsToIndexesNoBackend is a temp version of AddParticipantsToIndexes that
// skips backend downloads and it used in AWS Batch. Refactoring required to make it better.
func (s *store) AddParticipantsToIndexesNoBackend(checkpoint uint32, index string, participants []string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, participant := range participants {
		_, ok := s.indexes[participant]
		if !ok {
			s.indexes[participant] = map[string]*CheckpointIndex{}
		}

		ind, ok := s.indexes[participant][index]
		if !ok {
			ind = &CheckpointIndex{}
			s.indexes[participant][index] = ind
		}

		err := ind.SetActive(checkpoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *store) AddParticipantToIndexesNoBackend(participant string, indexes map[string]*CheckpointIndex) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.indexes[participant] = indexes
}

func (s *store) AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error {
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

func (s *store) getCreateIndex(account, id string) (*CheckpointIndex, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if we already have it loaded
	accountIndexes, ok := s.indexes[account]
	if !ok {
		accountIndexes = map[string]*CheckpointIndex{}
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
		ind = &CheckpointIndex{}
		accountIndexes[id] = ind
	}
	s.indexes[account] = accountIndexes

	return ind, nil
}

func (s *store) NextActive(account, indexId string, afterCheckpoint uint32) (uint32, error) {
	ind, err := s.getCreateIndex(account, indexId)
	if err != nil {
		return 0, err
	}
	return ind.NextActive(afterCheckpoint)
}

func (s *store) getCreateTrieIndex(prefix string) (*TrieIndex, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

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
		index = &TrieIndex{}
		s.txIndexes[prefix] = index
	}

	return index, nil
}
