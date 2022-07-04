// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"container/heap"
	"io"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

// FsCacheBackend fronts another backend with a local filesystem cache
type FsCacheBackend struct {
	ArchiveBackend
	dir string
	//lint:ignore U1000 Ignore unused temporarily
	knownFiles lruCache
	maxFiles   int
	lru        lruCache
}

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

func (b *FsCacheBackend) GetFile(pth string) (r io.ReadCloser, err error) {
	localPath := path.Join(b.dir, pth)
	local, err := os.Open(localPath)
	if err == nil {
		b.updateLRU(localPath)
		return local, nil
	}
	if !os.IsNotExist(err) {
		// Some local fs error.. log and continue?
		log.WithField("path", pth).WithError(err).Error("fs-cache: get file")
	}

	remote, err := b.ArchiveBackend.GetFile(pth)
	if err != nil {
		return remote, err
	}
	local, err = b.createLocal(pth)
	if err != nil {
		// Some local fs error.. log and continue?
		log.WithField("path", pth).WithError(err).Error("fs-cache: get file")
		return remote, nil
	}
	return teeReadCloser(remote, local), nil
}

func (b *FsCacheBackend) createLocal(pth string) (*os.File, error) {
	localPath := path.Join(b.dir, pth)

	if err := os.MkdirAll(path.Dir(localPath), 0755); err != nil {
		return nil, err
	}

	local, err := os.Create(localPath)
	if err != nil {
		return nil, err
	}
	b.updateLRU(localPath)
	return local, err
}

func (b *FsCacheBackend) Exists(pth string) (bool, error) {
	localPath := path.Join(b.dir, pth)
	log.WithField("path", pth).Trace("fs-cache: check exists")
	if _, err := os.Stat(localPath); err == nil {
		b.updateLRU(localPath)
		return true, nil
	}
	return b.ArchiveBackend.Exists(pth)
}

func (b *FsCacheBackend) Size(pth string) (int64, error) {
	localPath := path.Join(b.dir, pth)
	log.WithField("path", pth).Trace("fs-cache: check exists")
	fi, err := os.Stat(localPath)
	if err == nil {
		log.WithField("path", pth).WithField("size", fi.Size()).Trace("fs-cache: got size")
		b.updateLRU(localPath)
		return fi.Size(), nil
	}
	log.WithField("path", pth).WithError(err).Error("fs-cache: get size")
	return b.ArchiveBackend.Size(pth)
}

func (b *FsCacheBackend) PutFile(pth string, in io.ReadCloser) error {
	log.WithField("path", pth).Trace("fs-cache: put file")
	in = b.tryLocalPutFile(pth, in)
	return b.ArchiveBackend.PutFile(pth, in)
}

// Best effort to tee the upload off to the local cache as well
func (b *FsCacheBackend) tryLocalPutFile(pth string, in io.ReadCloser) io.ReadCloser {
	local, err := b.createLocal(pth)
	if err != nil {
		log.WithField("path", pth).WithError(err).Error("fs-cache: put file")
		return in
	}

	// tee upload data into our local file
	return teeReadCloser(in, local)
}

func (b *FsCacheBackend) updateLRU(pth string) {
	b.lru.bump(pth)
	for i := b.lru.Len(); i > b.maxFiles; i-- {
		item := b.lru.Pop().(*lruCacheItem)
		if err := os.Remove(item.path); err != nil {
			log.WithField("path", item.path).WithError(err).Error("fs-cache: evict")
		}
	}
}

func (b *FsCacheBackend) Close() error {
	return b.ArchiveBackend.Close()
}

// MakeFsCacheBackend, wraps an ArchiveBackend with a local filesystem cache in
// `dir`. If dir is blank, a temporary directory will be created.
func MakeFsCacheBackend(upstream ArchiveBackend, dir string, maxFiles uint) (ArchiveBackend, error) {
	if dir == "" {
		tmp, err := os.MkdirTemp(os.TempDir(), "stellar-horizon-*")
		if err != nil {
			return nil, err
		}
		dir = tmp
		log.WithField("dir", dir).Info("fs-cache: temp dir")
	}
	if maxFiles == 0 {
		// A guess at a reasonable number of checkpoints. This is 90 days of
		// ledgers. (90*86_400)/(5*64) = 24_300
		maxFiles = 24_300
	}
	// Add 10 here, cause we need a bit of spare room for pending evictions.
	var lru lruCache
	heap.Init(&lru)
	return &FsCacheBackend{
		ArchiveBackend: upstream,
		dir:            dir,
		maxFiles:       int(maxFiles),
		lru:            lru,
	}, nil
}

// lruCache is a heap-based LRU cache that we use to limit the on-disk size
type lruCache []*lruCacheItem

type lruCacheItem struct {
	path       string
	lastUsedAt time.Time
	index      int
}

func (c lruCache) Len() int { return len(c) }

func (c lruCache) Less(i, j int) bool {
	// We want Pop to give us the oldest, so we use before than here.
	return c[i].lastUsedAt.Before(c[j].lastUsedAt)
}

func (c lruCache) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
	c[i].index = i
	c[j].index = j
}

func (c *lruCache) Push(x interface{}) {
	n := len(*c)
	item := x.(*lruCacheItem)
	item.index = n
	*c = append(*c, item)
}

func (c *lruCache) Pop() interface{} {
	old := *c
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*c = old[0 : n-1]
	return item
}

func (c *lruCache) bump(pth string) {
	c.upsert(pth, time.Now())
}

// upsert modifies the priority and value of an item in the heap, or inserts it.
func (c *lruCache) upsert(pth string, lastUsedAt time.Time) {
	// Try to find by path and update
	for _, item := range *c {
		if item.path == pth {
			item.lastUsedAt = lastUsedAt
			heap.Fix(c, item.index)
			return
		}
	}
	// not found, add this item
	heap.Push(c, &lruCacheItem{
		path:       pth,
		lastUsedAt: lastUsedAt,
	})
}
