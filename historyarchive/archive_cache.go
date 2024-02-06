package historyarchive

import (
	"fmt"
	"io"
	"os"
	"path"

	lru "github.com/hashicorp/golang-lru"
	log "github.com/stellar/go/support/log"
)

type CacheOptions struct {
	Cache bool

	Path     string
	MaxFiles uint
	Log      *log.Entry
}

type ArchiveBucketCache struct {
	path string
	lru  *lru.Cache
	log  *log.Entry
}

// MakeArchiveBucketCache creates a cache on the disk at the given path that
// acts as an LRU cache, mimicking a particular upstream.
func MakeArchiveBucketCache(opts CacheOptions) (*ArchiveBucketCache, error) {
	log_ := opts.Log
	if opts.Log == nil {
		log_ = log.WithField("subservice", "fs-cache")
	}
	log_ = log_.
		WithField("path", opts.Path).
		WithField("cap", opts.MaxFiles)

	if _, err := os.Stat(opts.Path); err == nil || os.IsExist(err) {
		log_.Warnf("Cache directory already exists, removing")
		os.RemoveAll(opts.Path)
	}

	backend := &ArchiveBucketCache{
		path: opts.Path,
		log:  log_,
	}

	cache, err := lru.NewWithEvict(int(opts.MaxFiles), backend.onEviction)
	if err != nil {
		return &ArchiveBucketCache{}, err
	}
	backend.lru = cache

	log_.Info("Bucket cache initialized")
	return backend, nil
}

// GetFile retrieves the file contents from the local cache if present.
// Otherwise, it returns the same result as the upstream, adding that result
// into the local cache if possible. It returns a 3-tuple of a reader (which may
// be nil on an error), an indication of whether or not it was *found* in the
// cache, and any error.
func (abc *ArchiveBucketCache) GetFile(
	filepath string,
	upstream ArchiveBackend,
) (io.ReadCloser, bool, error) {
	L := abc.log.WithField("key", filepath)
	localPath := path.Join(abc.path, filepath)

	// If the lockfile exists, we should defer to the remote source but *not*
	// update the cache, as it means there's an in-progress sync of the same
	// file.
	_, statErr := os.Stat(NameLockfile(localPath))
	if statErr == nil {
		L.Info("Incomplete file in on-disk cache: deferring")
		reader, err := upstream.GetFile(filepath)
		return reader, false, err
	} else if _, ok := abc.lru.Get(localPath); !ok {
		L.Info("File does not exist in the cache: downloading")

		// Since it's not on-disk, pull it from the remote backend, shove it
		// into the cache, and write it to disk.
		remote, err := upstream.GetFile(filepath)
		if err != nil {
			L.WithError(err).Warn("Upstream download failed")
			return remote, false, err
		}

		local, err := abc.createLocal(filepath)
		if err != nil {
			// If there's some local FS error, we can still continue with the
			// remote version, so just log it and continue.
			L.WithError(err).Warn("Creating cache file failed")
			return remote, false, nil
		}

		// We only add it to the cache after the final close call.
		return teeReadCloser(remote, local, func() error {
			// Integrity check: stream in the file from the disk and check that
			// the hash matches the filename for integrity purposes.
			// f, err := os.Open(localPath)
			// if err != nil {
			// 	gReader, gErr := gzip.NewReader(f)
			// 	if gErr != nil {

			// 	}

			// 	h := sha256.New()
			// 	buf := [1024]byte{}
			// 	for {
			// 		nRead, err := gReader.Read(buf[:])
			// 		if err != io.EOF {
			// 			break
			// 		}
			// 		h.Write(buf[:nRead])
			// 	}

			// 	// checksum :=
			// 	// h.Sum()

			// } else {

			// }

			// gzip.NewReader()

			// Basic sanity check: does the upstream size match the on-disk
			// size? If not, something messed up during the fetch and we can't
			// use this.
			stat, statErr := os.Stat(localPath)
			if statErr != nil {
				L.WithError(statErr).Warnf("Couldn't stat cached file")
				abc.onEviction(localPath, nil)
				return statErr
			}

			upSize, sizeErr := upstream.Size(filepath)
			if sizeErr != nil {
				L.WithError(sizeErr).
					Warn("Couldn't fetch size from upstream")
				abc.onEviction(localPath, nil)
				return sizeErr
			} else if stat.Size() != upSize {
				sizeErr = fmt.Errorf("upstream size (%d) doesn't match cache (%d)", upSize, stat.Size())
				L.WithError(sizeErr).Warn("Couldn't confirm cached file integrity")
				abc.onEviction(localPath, nil)
				return sizeErr
			}

			L.Infof("Successfully cached %.2fKiB file", float32(upSize)/1024)
			abc.lru.Add(localPath, struct{}{}) // just use the cache as an array
			L.Debug("Removing lockfile")
			return os.Remove(NameLockfile(localPath))
		}), false, nil
	}

	L.Info("Found file in cache")
	// The cache claims it exists, so just give it a read and send it.
	local, err := os.Open(localPath)
	if err != nil {
		// Uh-oh, the cache and the disk are not in sync somehow? Let's evict
		// this value and try again (recurse) w/ the remote version.
		L.WithError(err).Warn("Opening cached file failed")
		abc.lru.Remove(localPath)
		return abc.GetFile(filepath, upstream)
	}

	return local, true, nil
}

func (abc *ArchiveBucketCache) Exists(filepath string) bool {
	L := abc.log.WithField("key", filepath)
	localPath := path.Join(abc.path, filepath)

	// First, check if the file exists in the cache.
	if abc.lru.Contains(localPath) {
		return true
	}

	// If it doesn't, it may still exist on the disk which is still a cheaper
	// check than going upstream.
	//
	// Note that this means the cache and disk are out of sync (perhaps due to
	// other archives using the same cache location) so we can update it. This
	// situation is well-handled by `GetFile`.
	_, statErr := os.Stat(localPath)
	if statErr == nil || os.IsExist(statErr) {
		L.Info("File found cached on disk")
		abc.lru.Add(localPath, struct{}{})
		return true
	}

	return false
}

// Close purges the cache and cleans up the filesystem.
func (abc *ArchiveBucketCache) Close() error {
	abc.lru.Purge()
	return os.RemoveAll(abc.path)
}

// Evict removes a file from the cache and the filesystem.
func (abc *ArchiveBucketCache) Evict(filepath string) {
	log.WithField("key", filepath).Info("Evicting file from the disk")
	abc.lru.Remove(path.Join(abc.path, filepath))
}

func (abc *ArchiveBucketCache) onEviction(key, _ interface{}) {
	path := key.(string)
	os.Remove(NameLockfile(path))           // just in case
	if err := os.Remove(path); err != nil { // best effort removal
		abc.log.WithError(err).
			WithField("key", path).
			Warn("Removal failed after cache eviction")
	}
}

func (abc *ArchiveBucketCache) createLocal(filepath string) (*os.File, error) {
	localPath := path.Join(abc.path, filepath)
	if err := os.MkdirAll(path.Dir(localPath), 0755 /* drwxr-xr-x */); err != nil {
		return nil, err
	}

	// First, create the lockfile to avoid contention.
	lockfile, err := os.Create(NameLockfile(localPath))
	if err != nil {
		return nil, err
	}

	local, err := os.Create(localPath) /* mode -rw-rw-rw- */
	if err != nil {
		// Be sure to clean up the lockfile, ignoring errors.
		lockfile.Close()
		os.Remove(NameLockfile(localPath))
		return nil, err
	}

	return local, nil
}

func NameLockfile(file string) string {
	return file + ".lock"
}

type cacheReader struct {
	file   *os.File
	reader io.Reader

	closed bool
}

var _ io.ReadCloser = &cacheReader{}
var _ io.WriteCloser = &cacheReader{}

func newCacheReader(upstream io.Reader, path string) (*cacheReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &cacheReader{
		reader: upstream,
		file:   file,
		closed: false,
	}, nil
}

func (cr *cacheReader) Read([]byte) (int, error) {

	return 0, nil
}

func (cr *cacheReader) Write([]byte) (int, error) {
	return 0, nil
}

func (cr *cacheReader) Close() error {
	return nil
}

// The below is a helper interface so that we can use io.TeeReader to write
// data locally immediately as we read it remotely.

type trc struct {
	io.Reader
	close  func() error
	closed bool // prevents a double-close
}

func (t trc) Close() error {
	if t.closed {
		return nil
	}

	return t.close()
}

func teeReadCloser(r io.ReadCloser, w io.WriteCloser, onClose func() error) io.ReadCloser {
	closer := trc{
		Reader: io.TeeReader(r, w),
		closed: false,
	}
	closer.close = func() error {
		if closer.closed {
			return nil
		}

		// Always run all closers but return the first possible error.

		err1 := r.Close()
		// Ensure that we flush to disk before closing
		// err2 := w.Sync()
		// if err2 == nil {
		err2 := w.Close()
		// }
		err3 := onClose()

		closer.closed = true
		if err1 != nil {
			return err1
		} else if err2 != nil {
			return err2
		}
		return err3
	}

	return closer
}
