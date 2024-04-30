package ledgerexporter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"cloud.google.com/go/storage"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/url"
)

// GCSDataStore implements DataStore for GCS
type GCSDataStore struct {
	client *storage.Client
	bucket *storage.BucketHandle
	prefix string
}

func NewGCSDataStore(ctx context.Context, params map[string]string, network string) (DataStore, error) {
	destinationBucketPath, ok := params["destination_bucket_path"]
	if !ok {
		return nil, errors.Errorf("Invalid GCS config, no destination_bucket_path")
	}

	// append the gcs:// scheme to enable usage of the url package reliably to
	// get parse bucket name which is first path segment as URL.Host
	gcsBucketURL := fmt.Sprintf("gcs://%s/%s", destinationBucketPath, network)
	parsed, err := url.Parse(gcsBucketURL)
	if err != nil {
		return nil, err
	}

	// Inside gcs, all paths start _without_ the leading /
	prefix := strings.TrimPrefix(parsed.Path, "/")
	bucketName := parsed.Host

	logger.Infof("creating GCS client for bucket: %s, prefix: %s", bucketName, prefix)

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

// GetFile retrieves a file from the GCS bucket.
func (b GCSDataStore) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	filePath = path.Join(b.prefix, filePath)
	r, err := b.bucket.Object(filePath).NewReader(ctx)
	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			logger.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
		}
		return nil, errors.Wrapf(err, "error retrieving file: %s", filePath)
	}
	logger.Infof("File retrieved successfully: %s", filePath)
	return r, nil
}

// PutFileIfNotExists uploads a file to GCS only if it doesn't already exist.
func (b GCSDataStore) PutFileIfNotExists(ctx context.Context, filePath string, in io.WriterTo) (bool, error) {
	err := b.putFile(ctx, filePath, in, &storage.Conditions{DoesNotExist: true})
	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			switch gcsError.Code {
			case http.StatusPreconditionFailed:
				logger.Infof("Precondition failed: %s already exists in the bucket", filePath)
				return false, nil // Treat as success
			default:
				logger.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
			}
		}
		return false, errors.Wrapf(err, "error uploading file:  %s", filePath)
	}
	logger.Infof("File uploaded successfully: %s", filePath)
	return true, nil
}

// PutFile uploads a file to GCS
func (b GCSDataStore) PutFile(ctx context.Context, filePath string, in io.WriterTo) error {
	err := b.putFile(ctx, filePath, in, nil) // No conditions for regular PutFile

	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			logger.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
		}
		return errors.Wrapf(err, "error uploading file: %v", filePath)
	}
	logger.Infof("File uploaded successfully: %s", filePath)
	return nil
}

// Size retrieves the size of a file in the GCS bucket.
func (b GCSDataStore) Size(ctx context.Context, pth string) (int64, error) {
	pth = path.Join(b.prefix, pth)
	attrs, err := b.bucket.Object(pth).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		err = os.ErrNotExist
	}
	if err != nil {
		return 0, err
	}
	return attrs.Size, nil
}

// Exists checks if a file exists in the GCS bucket.
func (b GCSDataStore) Exists(ctx context.Context, pth string) (bool, error) {
	_, err := b.Size(ctx, pth)

	if err == os.ErrNotExist {
		return false, nil
	}

	return err == nil, err
}

// Close closes the GCS client connection.
func (b GCSDataStore) Close() error {
	return b.client.Close()
}

func (b GCSDataStore) putFile(ctx context.Context, filePath string, in io.WriterTo, conditions *storage.Conditions) error {
	filePath = path.Join(b.prefix, filePath)
	o := b.bucket.Object(filePath)
	if conditions != nil {
		o = o.If(*conditions)
	}
	w := o.NewWriter(ctx)
	if _, err := in.WriteTo(w); err != nil {
		return errors.Wrapf(err, "failed to put file: %s", filePath)
	}
	return w.Close()
}
