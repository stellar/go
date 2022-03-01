package index

import (
	"os"
	"sync"
)

type Store interface {
	NextActive(account, index string, afterCheckpoint uint32) (uint32, error)
	AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error
	Flush() error
}

type Backend interface {
	Flush(map[string]map[string]*CheckpointIndex) error
	Read(account string) (map[string]*CheckpointIndex, error)
}

type store struct {
	mutex   sync.RWMutex
	indexes map[string]map[string]*CheckpointIndex
	backend Backend
}

func NewStore(backend Backend) (Store, error) {
	return &store{
		indexes: map[string]map[string]*CheckpointIndex{},
		backend: backend,
	}, nil
}

func (s *store) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.backend.Flush(s.indexes); err != nil {
		return err
	}

	// clear indexes to save memory
	s.indexes = map[string]map[string]*CheckpointIndex{}

	return nil
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
