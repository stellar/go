package exporter

import (
	"context"
	"google.golang.org/api/googleapi"
	"io"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/url"
	"google.golang.org/api/option"
)

// DataStore defines an interface for interacting with data storage
type DataStore interface {
	GetFile(path string) (io.ReadCloser, error)
	PutFile(key string, closer io.ReadCloser) error
	PutFileIfNotExists(string, io.ReadCloser) error
	Exists(path string) (bool, error)
	Size(path string) (int64, error)
	Close() error
}

// GCSDataStore implements DataStore for GCS
type GCSDataStore struct {
	client *storage.Client
	bucket *storage.BucketHandle
	prefix string
}

// NewDataStore creates a new DataStore based on the destination URL.
// Currently, only accepts GCS URLs.
func NewDataStore(destinationURL string) (DataStore, error) {
	parsed, err := url.Parse(destinationURL)
	if err != nil {
		return nil, err
	}

	pth := parsed.Path
	if parsed.Scheme != "gcs" {
		return nil, errors.Errorf("Invalid destination URL %s. Expected GCS URL ", destinationURL)
	}

	// Inside gcs, all paths start _without_ the leading /
	pth = strings.TrimPrefix(pth, "/")
	bucketName := parsed.Host
	prefix := pth

	log.WithFields(log.Fields{
		"bucket": bucketName,
		"prefix": prefix,
	}).Debug("gcs: making backend")

	var options []option.ClientOption
	client, err := storage.NewClient(context.Background(), options...)
	if err != nil {
		return nil, err
	}

	// Check the bucket exists
	bucket := client.Bucket(bucketName)
	if _, err := bucket.Attrs(context.Background()); err != nil {
		return nil, err
	}

	return &GCSDataStore{client: client, bucket: bucket, prefix: prefix}, nil
}

// GetFile retrieves a file from the GCS bucket.
func (b *GCSDataStore) GetFile(filePath string) (io.ReadCloser, error) {
	filePath = path.Join(b.prefix, filePath)
	log.WithField("path", filePath).Trace("gcs: get file")

	r, err := b.bucket.Object(filePath).NewReader(context.Background())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file: %s", filePath)
	}

	return r, nil
}

// PutFileIfNotExists uploads a file to GCS only if it doesn't already exist.
func (b *GCSDataStore) PutFileIfNotExists(pth string, in io.ReadCloser) error {
	err := b.putFile(pth, in, &storage.Conditions{DoesNotExist: true})
	if e, ok := err.(*googleapi.Error); ok {
		if e.Code == 412 {
			log.WithField("path", pth).Info("File already exists in the bucket, skipping upload.")
			return nil // Treat as success since the file is already present
		}
	}
	return err
}

// PutFile uploads a file to GCS
func (b *GCSDataStore) PutFile(pth string, in io.ReadCloser) error {
	return b.putFile(pth, in, nil) // No conditions for regular PutFile
}

// Size retrieves the size of a file in the GCS bucket.
func (b *GCSDataStore) Size(pth string) (int64, error) {
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

// Exists checks if a file exists in the GCS bucket.
func (b *GCSDataStore) Exists(pth string) (bool, error) {
	log.WithField("path", path.Join(b.prefix, pth)).Trace("gcs: check exists")
	_, err := b.Size(pth)
	return err == nil, err
}

// Close closes the GCS client connection.
func (b *GCSDataStore) Close() error {
	log.Trace("gcs: close")
	return b.client.Close()
}

func (b *GCSDataStore) putFile(filePath string, in io.ReadCloser, conditions *storage.Conditions) error {
	filePath = path.Join(b.prefix, filePath)
	log.WithField("path", filePath).Trace("gcs: put file")

	o := b.bucket.Object(filePath)
	if conditions != nil {
		o = o.If(*conditions)
	}
	w := o.NewWriter(context.Background())
	if _, err := io.Copy(w, in); err != nil {
		in.Close()
		return errors.Wrapf(err, "failed to put file: %s", filePath)
	}

	return w.Close()
}
