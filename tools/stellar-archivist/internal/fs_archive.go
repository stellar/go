// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package archivist

import (
	"io"
	"os"
	"path"
	"path/filepath"
)

type FsArchiveBackend struct {
	prefix string
}

func (b *FsArchiveBackend) GetFile(pth string) (io.ReadCloser, error) {
	return os.Open(path.Join(b.prefix, pth))
}

func (b *FsArchiveBackend) Exists(pth string) bool {
	pth = path.Join(b.prefix, pth)
	_, err := os.Stat(pth)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func (b *FsArchiveBackend) PutFile(pth string, in io.ReadCloser) error {
	dir := path.Join(b.prefix, path.Dir(pth))
	if !b.Exists(dir) {
		if e := os.MkdirAll(dir, 0755); e != nil {
			return e
		}
	}

	pth = path.Join(b.prefix, pth)
	out, e := os.Create(pth)
	if e != nil {
		return e
	}
	defer in.Close()
	defer out.Close()
	_, e = io.Copy(out, in)
	return e
}

func (b *FsArchiveBackend) ListFiles(pth string) (chan string, chan error) {
	ch := make(chan string)
	errs := make(chan error)
	go func() {
		filepath.Walk(path.Join(b.prefix, pth),
			func(p string, info os.FileInfo, err error) error {
				if err != nil {
					errs <- err
					return nil
				}
				if info != nil && !info.IsDir() {
					ch <- p
				}
				return nil
			})
		close(ch)
		close(errs)
	}()
	return ch, errs
}

func (b *FsArchiveBackend) CanListFiles() bool {
	return true
}

func MakeFsBackend(pth string, opts ConnectOptions) ArchiveBackend {
	return &FsArchiveBackend{
		prefix: pth,
	}
}
