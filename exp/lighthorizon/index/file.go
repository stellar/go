package index

import (
	"bytes"
	"compress/gzip"
	"io"
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
	indexes map[string]map[string]*CheckpointIndex
	dir     string
}

func NewFileIndexStore(dir string) (*FileIndexStore, error) {
	return &FileIndexStore{
		indexes: map[string]map[string]*CheckpointIndex{},
		dir:     dir,
	}, nil
}

func (s *FileIndexStore) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var written uint64

	for account, indexes := range s.indexes {
		if len(indexes) == 0 {
			continue
		}

		path := filepath.Join(s.dir, account[:3], account)

		err := os.MkdirAll(filepath.Dir(path), fs.ModeDir|0755)
		if err != nil {
			log.Errorf("Unable to mkdir %s, %v", filepath.Dir(path), err)
			continue
		}

		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)

		for id, index := range indexes {
			zw.Name = id
			_, err := io.Copy(zw, index.Buffer())
			if err != nil {
				log.Errorf("Unable to serialize %s/%s: %v", account, id, err)
				continue
			}

			if err := zw.Close(); err != nil {
				log.Errorf("Unable to serialize %s/%s: %v", account, id, err)
				continue
			}

			zw.Reset(&buf)
		}

		err = ioutil.WriteFile(path, buf.Bytes(), 0644)
		if err != nil {
			log.Errorf("Unable to save %s: %v", account, err)
			continue
		}

		nwritten := atomic.AddUint64(&written, 1)
		if nwritten%1000 == 0 {
			log.Infof("Writing indexes... %d/%d %.2f%%", nwritten, len(s.indexes), (float64(nwritten)/float64(len(s.indexes)))*100)
		}
	}

	// clear indexes to save memory
	s.indexes = map[string]map[string]*CheckpointIndex{}

	return nil
}

func (s *FileIndexStore) AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error {
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

func (s *FileIndexStore) getCreateIndex(account, id string) (*CheckpointIndex, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	accountIndexes, ok := s.indexes[account]
	if !ok {
		accountIndexes = map[string]*CheckpointIndex{}
	}
	ind, ok := accountIndexes[id]
	if ok {
		return ind, nil
	}

	// Check if index exists in S3
	log.Debugf("Opening index: %s/%s", account, id)
	b, err := ioutil.ReadFile(filepath.Join(s.dir, account[:3], account))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Got all bunch of indexes for account, parse 'em
	if err == nil {
		reader := bytes.NewReader(b)
		zr, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		for {
			zr.Multistream(false)

			if _, err := io.Copy(&buf, zr); err != nil {
				return nil, err
			}

			ind, err = NewCheckpointIndexFromBytes(buf.Bytes())
			if err != nil {
				return nil, err
			}
			buf.Reset()

			accountIndexes[zr.Name] = ind

			err = zr.Reset(reader)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
		}

		if err := zr.Close(); err != nil {
			return nil, err
		}
	}

	ind, ok = accountIndexes[id]
	if !ok {
		ind = &CheckpointIndex{}
		accountIndexes[id] = ind
	}
	s.indexes[account] = accountIndexes

	return ind, nil
}

func (s *FileIndexStore) NextActive(account, indexId string, afterCheckpoint uint32) (uint32, error) {
	ind, err := s.getCreateIndex(account, indexId)
	if err != nil {
		return 0, err
	}
	return ind.NextActive(afterCheckpoint)
}
