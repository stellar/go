// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"io"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

// FsCacheBackend fronts another backend with a local filesystem cache
type FsCacheBackend struct {
	ArchiveBackend
	dir string
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
	return local, err
}

func (b *FsCacheBackend) Exists(pth string) (bool, error) {
	log.WithField("path", pth).Trace("fs-cache: check exists")
	if _, err := os.Stat(path.Join(b.dir, pth)); err == nil {
		return true, nil
	}
	return b.ArchiveBackend.Exists(pth)
}

func (b *FsCacheBackend) Size(pth string) (int64, error) {
	log.WithField("path", pth).Trace("fs-cache: check exists")
	fi, err := os.Stat(path.Join(b.dir, pth))
	if err == nil {
		log.WithField("path", pth).WithField("size", fi.Size()).Trace("fs-cache: got size")
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

// MakeFsCacheBackend, wraps an ArchiveBackend with a local filesystem cache in
// `dir`. If dir is blank, a temporary directory will be created.
func MakeFsCacheBackend(upstream ArchiveBackend, dir string) (ArchiveBackend, error) {
	if dir == "" {
		tmp, err := os.MkdirTemp(os.TempDir(), "stellar-horizon-*")
		if err != nil {
			return nil, err
		}
		dir = tmp
	}
	return &FsCacheBackend{
		ArchiveBackend: upstream,
		dir:            dir,
	}, nil
}
