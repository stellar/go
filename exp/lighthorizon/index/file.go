package index

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	xdr3 "github.com/stellar/go-xdr/xdr3"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
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

	f, err := os.OpenFile(path, os.O_CREATE|
		os.O_APPEND| // crucial! since we might flush from various sources
		os.O_WRONLY,
		0664) // rw-rw-r--

	if err != nil {
		return errors.Wrapf(err, "failed to open account file at %s", path)
	}

	defer f.Close()

	for _, account := range accounts {
		muxed := xdr.MuxedAccount{}
		if err := muxed.SetAddress(account); err != nil {
			return errors.Wrapf(err, "failed to encode %s", account)
		}

		raw, err := muxed.MarshalBinary()
		if err != nil {
			return errors.Wrapf(err, "failed to marshal %s", account)
		}

		_, err = f.Write(raw)
		if err != nil {
			return errors.Wrapf(err, "failed to write to %s", path)
		}
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

func (s *FileBackend) FlushTransactions(indexes map[string]*TrieIndex) error {
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
	path := filepath.Join(s.dir, "accounts")
	log.Debugf("Opening accounts list at %s", path)

	// We read the file in chunks with guarantees that we will always read on an
	// account boundary:
	//
	//   Accounts w/o IDs are always 36 bytes (4-byte type, 32-byte pubkey)
	//   Muxed accounts with IDs are always 44 bytes (36 + 8-byte ID)
	//
	// If we read 36*44=1584 bytes at a time, we are guaranteed to have a
	// complete set of accounts (no partial buffer). Then, we bump this by 4 to
	// read a sizeable amount into memory (the built-in buffered reader does
	// 4096 bytes at a time).
	//
	// This keeps minimal file data in memory.
	const chunkSize = 4 * 36 * 44

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, err
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", path)
	}

	// The capacity here is ballparked based on all of the values being
	// G-addresses (32 public key bytes) plus the key type (4 bytes).
	preallocationSize := chunkSize / 36
	info, err := os.Stat(path)
	if err == nil { // we can still safely continue w/ errors
		// Note that this will never be too large, but may be too small.
		preallocationSize = int(info.Size()) / 36
	}
	accounts := make([]string, 0, preallocationSize)

	for {
		buffer := [chunkSize]byte{}
		readBytes, err := f.Read(buffer[:])

		if err == io.EOF || readBytes <= 0 {
			break
		} else if err != nil {
			return nil, errors.Wrapf(err, "failed reading %s", path)
		}

		// We don't use UnmarshalBinary here because we need to know how much of
		// the buffer was read for each account.
		reader := bytes.NewReader(buffer[:readBytes])
		d := xdr3.NewDecoder(reader)

		for i := 0; i < readBytes; {
			muxed := xdr.MuxedAccount{}
			xdrBytesRead, err := muxed.DecodeFrom(d)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode account")
			}

			account, err := muxed.GetAddress()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get strkey")
			}

			accounts = append(accounts, account)
			i += xdrBytesRead
		}
	}

	// The account list is very unlikely to be unique (especially if it was made
	// w/ parallel flushes), so let's ensure that that's the case.
	count := 0
	accountMap := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		if _, ok := accountMap[account]; !ok {
			accountMap[account] = struct{}{}
			accounts[count] = account // save memory: shove uniques to front
			count++
		}
	}

	return accounts[:count], nil
}

func (s *FileBackend) ReadTransactions(prefix string) (*TrieIndex, error) {
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
	var index TrieIndex
	_, err = index.ReadFrom(zr)
	if err != nil {
		log.Errorf("Unable to parse %s: %v", prefix, err)
		return nil, os.ErrNotExist
	}
	return &index, nil
}
