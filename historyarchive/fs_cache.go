// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"io"
	"os"
	"path"

	lru "github.com/hashicorp/golang-lru"
	"github.com/stellar/go/support/log"
)

// FsCacheBackend fronts another backend with a local filesystem cache
type FsCacheBackend struct {
	ArchiveBackend
	dir      string
	maxFiles int
	lru      *lru.Cache

	log *log.Entry
}

// MakeFsCacheBackend wraps an ArchiveBackend with a local filesystem cache in
// `dir`. If dir is blank, a temporary directory will be created. If `maxFiles`
// is zero, a default (90 days of ledgers) is used.
func MakeFsCacheBackend(upstream ArchiveBackend, dir string, maxFiles uint) (ArchiveBackend, error) {
	if dir == "" {
		tmp, err := os.MkdirTemp(os.TempDir(), "stellar-horizon-*")
		if err != nil {
			return nil, err
		}
		dir = tmp
	}
	if maxFiles == 0 {
		// A guess at a reasonable number of checkpoints. This is 90 days of
		// ledgers. (90*86_400)/(5*64) = 24_300
		maxFiles = 24_300
	}

	backendLog := log.
		WithField("subservice", "fs-cache").
		WithField("path", dir).
		WithField("size", maxFiles)
	backendLog.Info("Filesystem cache configured")

	backend := &FsCacheBackend{
		ArchiveBackend: upstream,
		dir:            dir,
		maxFiles:       int(maxFiles),
		log:            backendLog,
	}

	cache, err := lru.NewWithEvict(int(maxFiles), backend.onEviction)
	if err != nil {
		return nil, err
	}

	backend.lru = cache
	return backend, nil
}

func (b *FsCacheBackend) GetFile(filepath string) (io.ReadCloser, error) {
	L := b.log.WithField("key", filepath)
	localPath := path.Join(b.dir, filepath)

	if _, ok := b.lru.Get(localPath); !ok {
		// If it doesn't exist in the cache, it might still exist on the disk if
		// we've restarted from an existing directory.
		local, err := os.Open(localPath)
		if err == nil {
			L.Debug("found file on disk but not in cache, adding")
			b.lru.Add(localPath, struct{}{})
			return local, nil
		}

		b.log.WithField("key", filepath).
			Debug("retrieving file from remote backend")

		// Since it's not on-disk, pull it from the remote backend, shove it
		// into the cache, and write it to disk.
		remote, err := b.ArchiveBackend.GetFile(filepath)
		if err != nil {
			return remote, err
		}

		local, err = b.createLocal(filepath)
		if err != nil {
			// If there's some local FS error, we can still continue with the
			// remote version, so just log it and continue.
			L.WithError(err).Error("caching ledger failed")

			return remote, nil
		}

		return teeReadCloser(remote, local), nil
	}

	// The cache claims it exists, so just give it a read and send it.
	local, err := os.Open(localPath)
	if err != nil {
		// Uh-oh, the cache and the disk are not in sync somehow? Let's evict
		// this value and try again (recurse) w/ the remote version.
		L.WithError(err).Warn("opening cached ledger failed")
		b.lru.Remove(localPath)
		return b.GetFile(filepath)
	}

	L.Debug("Found file in cache")
	return local, nil
}

func (b *FsCacheBackend) Exists(filepath string) (bool, error) {
	localPath := path.Join(b.dir, filepath)
	b.log.WithField("key", filepath).Debug("checking existence")

	if _, ok := b.lru.Get(localPath); ok {
		// If the cache says it's there, we can definitively say that this path
		// exists, even if we'd fail to `os.Stat()/Read()/etc.` it locally.
		return true, nil
	}

	return b.ArchiveBackend.Exists(filepath)
}

func (b *FsCacheBackend) Size(filepath string) (int64, error) {
	localPath := path.Join(b.dir, filepath)
	L := b.log.WithField("key", filepath)

	L.Debug("retrieving size")
	if _, ok := b.lru.Get(localPath); ok {
		stats, err := os.Stat(localPath)
		if err == nil {
			L.Debugf("retrieved cached size: %d", stats.Size())
			return stats.Size(), nil
		}

		L.WithError(err).Debug("retrieving size of cached ledger failed")
		b.lru.Remove(localPath) // stale cache?
	}

	return b.ArchiveBackend.Size(filepath)
}

func (b *FsCacheBackend) PutFile(filepath string, in io.ReadCloser) error {
	L := log.WithField("key", filepath)
	L.Debug("putting file")

	// Best effort to tee the upload off to the local cache as well
	local, err := b.createLocal(filepath)
	if err != nil {
		L.WithError(err).Error("failed to put file locally")
	} else {
		// tee upload data into our local file
		in = teeReadCloser(in, local)
	}

	return b.ArchiveBackend.PutFile(filepath, in)
}

func (b *FsCacheBackend) Close() error {
	// This means nothing for the cached side of things unless we start storing
	// file handles in the cache.
	//
	// TODO: Actually, should we purge the cache here?
	return b.ArchiveBackend.Close()
}

func (b *FsCacheBackend) onEviction(key, value interface{}) {
	path := key.(string)
	if err := os.Remove(path); err != nil { // best effort removal
		b.log.WithError(err).
			WithField("key", path).
			Error("removal failed after cache eviction")
	}
}

func (b *FsCacheBackend) createLocal(filepath string) (*os.File, error) {
	localPath := path.Join(b.dir, filepath)
	if err := os.MkdirAll(path.Dir(localPath), 0755 /* drwxr-xr-x */); err != nil {
		return nil, err
	}

	local, err := os.Create(localPath) /* mode -rw-rw-rw- */
	if err != nil {
		return nil, err
	}

	b.lru.Add(localPath, struct{}{}) // just use the cache as an array
	return local, nil
}

// The below is a helper interface so that we can use io.TeeReader to write
// data locally immediately as we read it remotely.

type trc struct {
	io.Reader
	close func() error
}

func (t trc) Close() error {
	return t.close()
}

func teeReadCloser(r io.ReadCloser, w io.WriteCloser) io.ReadCloser {
	return trc{
		Reader: io.TeeReader(r, w),
		close: func() error {
			r.Close()
			return w.Close()
		},
	}
}
