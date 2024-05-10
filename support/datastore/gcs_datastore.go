package datastore

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"cloud.google.com/go/storage"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
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

// GetFile retrieves a file from the GCS bucket.
func (b GCSDataStore) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	filePath = path.Join(b.prefix, filePath)
	// setting ReadCompressed(true) will avoid transcoding of compressed files by including
	// an "Accept-Encoding: gzip" header in the request:
	// https://github.com/googleapis/google-cloud-go/blob/main/storage/http_client.go#L1307-L1309
	// https://cloud.google.com/storage/docs/transcoding#decompressive_transcoding
	// This will ensure that the reader performs CRC validation upon finishing the download:
	// https://pkg.go.dev/cloud.google.com/go/storage#Reader
	r, err := b.bucket.Object(filePath).ReadCompressed(true).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, os.ErrNotExist
		}
		if gcsError, ok := err.(*googleapi.Error); ok {
			log.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
		}
		return nil, errors.Wrapf(err, "error retrieving file: %s", filePath)
	}
	log.Infof("File retrieved successfully: %s", filePath)
	return r, nil
}

// PutFileIfNotExists uploads a file to GCS only if it doesn't already exist.
func (b GCSDataStore) PutFileIfNotExists(ctx context.Context, filePath string, in io.WriterTo) (bool, error) {
	err := b.putFile(ctx, filePath, in, &storage.Conditions{DoesNotExist: true})
	if err != nil {
		if gcsError, ok := err.(*googleapi.Error); ok {
			switch gcsError.Code {
			case http.StatusPreconditionFailed:
				log.Infof("Precondition failed: %s already exists in the bucket", filePath)
				return false, nil // Treat as success
			default:
				log.Errorf("GCS error: %s %s", gcsError.Message, gcsError.Body)
			}
		}
		return false, errors.Wrapf(err, "error uploading file:  %s", filePath)
	}
	log.Infof("File uploaded successfully: %s", filePath)
	return true, nil
}

// PutFile uploads a file to GCS
func (b GCSDataStore) PutFile(ctx context.Context, filePath string, in io.WriterTo) error {
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
	buf := &bytes.Buffer{}
	if _, err := in.WriteTo(buf); err != nil {
		return errors.Wrapf(err, "failed to write file: %s", filePath)
	}
	w.SendCRC32C = true
	// we must set CRC32C before invoking w.Write() for the first time
	w.CRC32C = crc32.Checksum(buf.Bytes(), crc32.MakeTable(crc32.Castagnoli))
	if _, err := in.WriteTo(buf); err != nil {
		return errors.Wrapf(err, "failed to put file: %s", filePath)
	}
	return w.Close()
}
