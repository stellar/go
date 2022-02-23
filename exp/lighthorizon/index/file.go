package index

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/stellar/go/support/log"
)

type FileIndexStore struct {
	mutex   sync.RWMutex
	indexes map[string]*CheckpointIndex
	dir     string
}

func NewFileIndexStore(dir string) (*FileIndexStore, error) {
	return &FileIndexStore{
		indexes: map[string]*CheckpointIndex{},
		dir:     dir,
	}, nil
}

func (s *FileIndexStore) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var written uint64

	for id, index := range s.indexes {
		path := filepath.Join(s.dir, id[:3], id)
		err := os.MkdirAll(filepath.Dir(path), fs.ModeDir|0755)
		if err != nil {
			log.Errorf("Unable to mkdir %s, %v", filepath.Dir(path), err)
			continue
		}
		err = ioutil.WriteFile(path, index.Buffer().Bytes(), 0644)
		if err != nil {
			log.Errorf("Unable to save %s, %v", id, err)
			continue
		}

		nwritten := atomic.AddUint64(&written, 1)
		if nwritten%1000 == 0 {
			log.Infof("Writing indexes... %d/%d %.2f%%", nwritten, len(s.indexes), (float64(nwritten)/float64(len(s.indexes)))*100)
		}
	}

	// clear indexes to save memory
	s.indexes = map[string]*CheckpointIndex{}

	return nil
}

func (s *FileIndexStore) AddParticipantsToIndexes(checkpoint uint32, indexFormat string, participants []string) error {
	for _, participant := range participants {
		ind, err := s.getCreateIndex(fmt.Sprintf(indexFormat, participant))
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

func (s *FileIndexStore) getCreateIndex(id string) (*CheckpointIndex, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	ind, ok := s.indexes[id]
	if ok {
		return ind, nil
	}

	// Check if index exists in S3
	log.Debugf("Opening index: %v", id)
	b, err := ioutil.ReadFile(filepath.Join(s.dir, id[:3], id))
	if os.IsNotExist(err) {
		ind = &CheckpointIndex{}
	} else if err != nil {
		return nil, err
	} else {
		ind, err = NewCheckpointIndexFromBytes(b)
		if err != nil {
			return nil, err
		}
	}

	s.indexes[id] = ind

	return ind, nil
}

func (s *FileIndexStore) NextActive(indexId string, afterCheckpoint uint32) (uint32, error) {
	ind, err := s.getCreateIndex(indexId)
	if err != nil {
		return 0, err
	}
	return ind.NextActive(afterCheckpoint)
}
