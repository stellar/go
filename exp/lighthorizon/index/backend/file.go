package index

import (
	"bufio"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	types "github.com/stellar/go/exp/lighthorizon/index/types"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

type FileBackend struct {
	dir      string
	parallel uint32
}

func NewFileBackend(dir string, parallel uint32) (*FileBackend, error) {
	if parallel <= 0 {
		parallel = 1
	}

	return &FileBackend{
		dir:      dir,
		parallel: parallel,
	}, nil
}

func (s *FileBackend) Flush(indexes map[string]types.NamedIndices) error {
	return parallelFlush(s.parallel, indexes, s.writeBatch)
}

func (s *FileBackend) FlushAccounts(accounts []string) error {
	path := filepath.Join(s.dir, "accounts")

	f, err := os.OpenFile(path, os.O_CREATE|
		os.O_APPEND| // crucial! since we might flush from various sources
		os.O_WRONLY,
		0664) // rw-rw-r--

	if err != nil {
		return errors.Wrapf(err, "failed to open account file at %s", path)
	}

	defer f.Close()

	// We write one account at a time because writes that occur within a single
	// `write()` syscall are thread-safe. A larger write might be split into
	// many calls and thus get interleaved, so we play it safe.
	for _, account := range accounts {
		f.Write([]byte(account + "\n"))
	}

	return nil
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

func (s *FileBackend) FlushTransactions(indexes map[string]*types.TrieIndex) error {
	// TODO: Parallelize this
	for key, index := range indexes {
		path := filepath.Join(s.dir, "tx", key)

		err := os.MkdirAll(filepath.Dir(path), fs.ModeDir|0755)
		if err != nil {
			log.Errorf("Unable to mkdir %s, %v", filepath.Dir(path), err)
			continue
		}

		f, err := os.Create(path)
		if err != nil {
			log.Errorf("Unable to create %s: %v", path, err)
			continue
		}

		zw := gzip.NewWriter(f)
		if _, err := index.WriteTo(zw); err != nil {
			log.Errorf("Unable to serialize %s: %v", path, err)
			f.Close()
			continue
		}

		if err := zw.Close(); err != nil {
			log.Errorf("Unable to serialize %s: %v", path, err)
			f.Close()
			continue
		}

		if err := f.Close(); err != nil {
			log.Errorf("Unable to save %s: %v", path, err)
		}
	}
	return nil
}

func (s *FileBackend) Read(account string) (types.NamedIndices, error) {
	log.Debugf("Opening index: %s", account)
	b, err := os.Open(filepath.Join(s.dir, account[:3], account))
	if err != nil {
		return nil, err
	}
	defer b.Close()

	indexes, _, err := readGzippedFrom(bufio.NewReader(b))
	if err != nil {
		log.Errorf("Unable to parse %s: %v", account, err)
		return nil, os.ErrNotExist
	}
	return indexes, nil
}

func (s *FileBackend) ReadAccounts() ([]string, error) {
	path := filepath.Join(s.dir, "accounts")
	log.Debugf("Opening accounts list at %s", path)

	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open %s", path)
	}

	const gAddressSize = 56

	// We ballpark the capacity assuming all of the values being G-addresses.
	preallocationSize := 100 * gAddressSize // default to 100 lines
	info, err := os.Stat(path)
	if err == nil { // we can still safely continue w/ errors
		// Note that this will never be too large, but may be too small.
		preallocationSize = int(info.Size()) / (gAddressSize + 1) // +1 for \n
	}
	accountMap := set.NewSet[string](preallocationSize)
	accounts := make([]string, 0, preallocationSize)

	reader := bufio.NewReaderSize(f, 100*gAddressSize) // reasonable buffer size
	for {
		line, err := reader.ReadString(byte('\n'))
		if err == io.EOF {
			break
		} else if err != nil {
			return accounts, errors.Wrapf(err, "failed to read %s", path)
		}

		account := line[:len(line)-1] // trim newline

		// The account list is very unlikely to be unique (especially if it was made
		// w/ parallel flushes), so let's ensure that that's the case.
		if !accountMap.Contains(account) {
			accountMap.Add(account)
			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

func (s *FileBackend) ReadTransactions(prefix string) (*types.TrieIndex, error) {
	log.Debugf("Opening index: %s", prefix)
	b, err := os.Open(filepath.Join(s.dir, "tx", prefix))
	if err != nil {
		return nil, err
	}
	defer b.Close()
	zr, err := gzip.NewReader(b)
	if err != nil {
		log.Errorf("Unable to parse %s: %v", prefix, err)
		return nil, os.ErrNotExist
	}
	defer zr.Close()
	var index types.TrieIndex
	_, err = index.ReadFrom(zr)
	if err != nil {
		log.Errorf("Unable to parse %s: %v", prefix, err)
		return nil, os.ErrNotExist
	}
	return &index, nil
}
