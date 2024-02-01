package exporter

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"

	"google.golang.org/api/googleapi"

	"cloud.google.com/go/storage"
	"github.com/stellar/go/support/errors"
)

// GCSDataStore implements DataStore for GCS
type GCSDataStore struct {
	client *storage.Client
	bucket *storage.BucketHandle
	prefix string
}

// GetFile retrieves a file from the GCS bucket.
func (b *GCSDataStore) GetFile(filePath string) (io.ReadCloser, error) {
	filePath = path.Join(b.prefix, filePath)
	r, err := b.bucket.Object(filePath).NewReader(context.Background())
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
func (b *GCSDataStore) PutFileIfNotExists(filePath string, in io.ReadCloser) error {
	err := b.putFile(filePath, in, &storage.Conditions{DoesNotExist: true})
	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			switch gcsError.Code {
			case http.StatusPreconditionFailed:
				logger.Infof("Precondition failed: %s already exists in the bucket", filePath)
				return nil // Treat as success
			default:
				logger.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
			}
		}
		return errors.Wrapf(err, "error uploading file:  %s", filePath)
	}
	logger.Infof("File uploaded successfully: %s", filePath)
	return nil
}

// PutFile uploads a file to GCS
func (b *GCSDataStore) PutFile(filePath string, in io.ReadCloser) error {
	err := b.putFile(filePath, in, nil) // No conditions for regular PutFile

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
func (b *GCSDataStore) Size(pth string) (int64, error) {
	pth = path.Join(b.prefix, pth)
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
	_, err := b.Size(pth)
	return err == nil, err
}

// Close closes the GCS client connection.
func (b *GCSDataStore) Close() error {
	return b.client.Close()
}

func (b *GCSDataStore) putFile(filePath string, in io.ReadCloser, conditions *storage.Conditions) error {
	filePath = path.Join(b.prefix, filePath)
	o := b.bucket.Object(filePath)
	if conditions != nil {
		o = o.If(*conditions)
	}
	w := o.NewWriter(context.Background())
	if _, err := io.Copy(w, in); err != nil {
		_ = in.Close()
		return errors.Wrapf(err, "failed to put file: %s", filePath)
	}
	return w.Close()
}
