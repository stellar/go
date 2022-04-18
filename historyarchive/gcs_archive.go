// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"context"
	"io"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"cloud.google.com/go/storage"
)

type GCSArchiveBackend struct {
	ctx    context.Context
	client *storage.Client
	bucket *storage.BucketHandle
	prefix string
}

func (b *GCSArchiveBackend) Exists(pth string) (bool, error) {
	log.WithField("path", path.Join(b.prefix, pth)).Trace("gcs: check exists")
	_, err := b.Size(pth)
	return err == nil, err
}

func (b *GCSArchiveBackend) Size(pth string) (int64, error) {
	pth = path.Join(b.prefix, pth)
	log.WithField("path", pth).Trace("gcs: get size")
	attrs, err := b.bucket.Object(pth).Attrs(context.Background())
	if err == storage.ErrObjectNotExist {
		err = os.ErrNotExist
	}
	if err != nil {
		return 0, err
	}
	return attrs.Size, nil
}

func (b *GCSArchiveBackend) GetFile(pth string) (io.ReadCloser, error) {
	pth = path.Join(b.prefix, pth)
	log.WithField("path", pth).Trace("gcs: get file")
	r, err := b.bucket.Object(pth).NewReader(context.Background())
	if err == storage.ErrObjectNotExist {
		// TODO: Check this is right
		err = os.ErrNotExist
	}
	return r, nil
}

func (b *GCSArchiveBackend) PutFile(pth string, in io.ReadCloser) error {
	pth = path.Join(b.prefix, pth)
	log.WithField("path", pth).Trace("gcs: get file")
	w := b.bucket.Object(pth).NewWriter(context.Background())
	if _, err := io.Copy(w, in); err != nil {
		return err
	}
	in.Close()
	return w.Close()
}

func (b *GCSArchiveBackend) ListFiles(pth string) (chan string, chan error) {
	prefix := path.Join(b.prefix, pth)
	ch := make(chan string)
	errs := make(chan error)

	go func() {
		log.WithField("path", pth).Trace("gcs: list files")
		defer close(ch)
		defer close(errs)

		iter := b.bucket.Objects(context.Background(), &storage.Query{Prefix: prefix})
		for {
			object, err := iter.Next()
			if err == iterator.Done {
				return
			} else if err != nil {
				errs <- err
			} else {
				// TODO: Check Name is right
				ch <- object.Name
			}
		}
	}()

	return ch, errs
}

func (b *GCSArchiveBackend) CanListFiles() bool {
	log.Trace("gcs: can list files")
	return true
}

func (b *GCSArchiveBackend) Close() error {
	log.Trace("gcs: close")
	return b.client.Close()
}

func makeGCSBackend(bucketName string, prefix string, opts ConnectOptions) (ArchiveBackend, error) {
	log.WithFields(log.Fields{
		"bucket":   bucketName,
		"prefix":   prefix,
		"endpoint": opts.GCSEndpoint,
	}).Debug("gcs: making backend")

	var options []option.ClientOption
	if opts.GCSEndpoint != "" {
		options = append(options, option.WithEndpoint(opts.GCSEndpoint))
	}

	client, err := storage.NewClient(opts.Context, options...)
	if err != nil {
		return nil, err
	}

	// Check the bucket exists
	bucket := client.Bucket(bucketName)
	if _, err := bucket.Attrs(opts.Context); err != nil {
		return nil, err
	}

	backend := GCSArchiveBackend{
		ctx:    opts.Context,
		client: client,
		bucket: bucket,
		prefix: prefix,
	}
	return &backend, nil
}
