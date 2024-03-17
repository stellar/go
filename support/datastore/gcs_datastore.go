package datastore

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"cloud.google.com/go/storage"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// GCSDataStore implements DataStore for GCS
type GCSDataStore struct {
	client *storage.Client
	bucket *storage.BucketHandle
	prefix string
}

// GetFile retrieves a file from the GCS bucket.
func (b *GCSDataStore) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	filePath = path.Join(b.prefix, filePath)
	r, err := b.bucket.Object(filePath).NewReader(ctx)
	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			log.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
		}
		return nil, errors.Wrapf(err, "error retrieving file: %s", filePath)
	}
	log.Infof("File retrieved successfully: %s", filePath)
	return r, nil
}

// PutFileIfNotExists uploads a file to GCS only if it doesn't already exist.
func (b *GCSDataStore) PutFileIfNotExists(ctx context.Context, filePath string, in io.WriterTo) error {
	err := b.putFile(ctx, filePath, in, &storage.Conditions{DoesNotExist: true})
	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			switch gcsError.Code {
			case http.StatusPreconditionFailed:
				log.Infof("Precondition failed: %s already exists in the bucket", filePath)
				return nil // Treat as success
			default:
				log.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
			}
		}
		return errors.Wrapf(err, "error uploading file:  %s", filePath)
	}
	log.Infof("File uploaded successfully: %s", filePath)
	return nil
}

// PutFile uploads a file to GCS
func (b *GCSDataStore) PutFile(ctx context.Context, filePath string, in io.WriterTo) error {
	err := b.putFile(ctx, filePath, in, nil) // No conditions for regular PutFile

	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			log.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
		}
		return errors.Wrapf(err, "error uploading file: %v", filePath)
	}
	log.Infof("File uploaded successfully: %s", filePath)
	return nil
}

// Size retrieves the size of a file in the GCS bucket.
func (b *GCSDataStore) Size(ctx context.Context, pth string) (int64, error) {
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
func (b *GCSDataStore) Exists(ctx context.Context, pth string) (bool, error) {
	_, err := b.Size(ctx, pth)
	return err == nil, err
}

// Close closes the GCS client connection.
func (b *GCSDataStore) Close() error {
	return b.client.Close()
}

func (b *GCSDataStore) putFile(ctx context.Context, filePath string, in io.WriterTo, conditions *storage.Conditions) error {
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

func (b *GCSDataStore) ListObjects(ctx context.Context, path string) ([]string, error) {
	var objectNames []string

	o := b.bucket.Objects(ctx, nil)
	for {
		attrs, err := o.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		objectNames = append(objectNames, attrs.Name)
	}

	return nil, nil
}
