package index

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/stellar/go/support/log"
)

type FileBackend struct {
	dir      string
	parallel uint32
}

func NewFileStore(dir string, parallel uint32) (Store, error) {
	backend, err := NewFileBackend(dir, parallel)
	if err != nil {
		return nil, err
	}
	return NewStore(backend)
}

func NewFileBackend(dir string, parallel uint32) (*FileBackend, error) {
	return &FileBackend{
		dir:      dir,
		parallel: parallel,
	}, nil
}

func (s *FileBackend) Flush(indexes map[string]map[string]*CheckpointIndex) error {
	return parallelFlush(s.parallel, indexes, s.writeBatch)
}

func (s *FileBackend) FlushAccounts(accounts []string) error {
	path := filepath.Join(s.dir, "accounts")

	accountsString := strings.Join(accounts, "\n")
	return os.WriteFile(path, []byte(accountsString), fs.ModeDir|0755)
}

func (s *FileBackend) writeBatch(b *batch) error {
	if len(b.indexes) == 0 {
		return nil
	}

	path := filepath.Join(s.dir, b.account[:3], b.account)

	err := os.MkdirAll(filepath.Dir(path), fs.ModeDir|0755)
	if err != nil {
		log.Errorf("Unable to mkdir %s, %v", filepath.Dir(path), err)
		return nil
	}

	f, err := os.Create(path)
	if err != nil {
		log.Errorf("Unable to create %s: %v", path, err)
		return nil
	}
	defer f.Close()

	if _, err := writeGzippedTo(f, b.indexes); err != nil {
		log.Errorf("Unable to serialize %s: %v", b.account, err)
		return nil
	}

	return nil
}

func (s *FileBackend) Read(account string) (map[string]*CheckpointIndex, error) {
	log.Debugf("Opening index: %s", account)
	b, err := os.Open(filepath.Join(s.dir, account[:3], account))
	if err != nil {
		return nil, err
	}
	defer b.Close()
	indexes, _, err := readGzippedFrom(b)
	if err != nil {
		log.Errorf("Unable to parse %s: %v", account, err)
		return nil, os.ErrNotExist
	}
	return indexes, nil
}

func (s *FileBackend) ReadAccounts() ([]string, error) {
	panic("TODO")
}
