package datastore

import (
	"context"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/url"
	"google.golang.org/api/option"
)

// DataStore defines an interface for interacting with data storage
type DataStore interface {
	GetFile(ctx context.Context, path string) (io.ReadCloser, error)
	PutFile(ctx context.Context, path string, in io.WriterTo) error
	PutFileIfNotExists(ctx context.Context, path string, in io.WriterTo) error
	Exists(ctx context.Context, path string) (bool, error)
	Size(ctx context.Context, path string) (int64, error)
	Close() error
	ListObjects(ctx context.Context, path string) ([]string, error)
}

// NewDataStore creates a new DataStore based on the destination URL.
// Currently, only accepts GCS URLs.
func NewDataStore(ctx context.Context, destinationURL string) (DataStore, error) {
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

	log.Infof("creating GCS client for bucket: %s, prefix: %s", bucketName, prefix)

	var options []option.ClientOption
	client, err := storage.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}

	// Check the bucket exists
	bucket := client.Bucket(bucketName)
	if _, err := bucket.Attrs(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve bucket attributes")
	}

	return &GCSDataStore{client: client, bucket: bucket, prefix: prefix}, nil
}
