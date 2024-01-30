package exporter

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Uploader is responsible for uploading data to a storage destination.
type Uploader interface {
	Run(ctx context.Context) error
	Upload(metaArchive *LedgerMetaArchive) error
}

type uploader struct {
	destination   DataStore
	metaArchiveCh chan *LedgerMetaArchive
}

// NewUploader creates a new Uploader
func NewUploader(destination DataStore, metaArchiveCh chan *LedgerMetaArchive) Uploader {
	return &uploader{
		destination:   destination,
		metaArchiveCh: metaArchiveCh,
	}
}

// Upload uploads the serialized binary data of ledger TxMeta to the specified destination.
// It includes retry logic with linear backoff to handle transient errors.
func (u *uploader) Upload(metaArchive *LedgerMetaArchive) error {
	logger.Infof("Uploading: %s", metaArchive.GetObjectKey())

	blob, err := metaArchive.GetBinaryData()
	if err != nil {
		return errors.Wrap(err, "failed to get binary data")
	}

	compressedBlob, err := Compress(blob)
	if err != nil {
		return errors.Wrap(err, "failed to compress data")
	}

	err = u.destination.PutFileIfNotExists(metaArchive.GetObjectKey(),
		io.NopCloser(bytes.NewReader(compressedBlob)))
	if err != nil {
		return errors.Wrapf(err, "error uploading %s", metaArchive.GetObjectKey())
	}
	return nil
}

// Run starts the uploader, continuously listening for ledger meta archive objects to upload.
func (u *uploader) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			// Drain the channel and upload pending objects before exiting.
			logger.Info("Stopping uploader, draining remaining objects from channel...")
			for obj := range u.metaArchiveCh {
				err := u.Upload(obj)
				if err != nil {
					logger.WithError(err).Errorf("Error uploading %s during shutdown", obj.objectKey)
				}
			}
			logger.WithError(ctx.Err()).Info("Uploader stopped")
			return ctx.Err()
		case metaObject, ok := <-u.metaArchiveCh:
			if !ok {
				return fmt.Errorf("export object channel closed, stopping uploader")
			}
			//Upload the received LedgerMetaArchive.
			err := u.Upload(metaObject)
			if err != nil {
				return errors.Wrapf(err, "error uploading %s", metaObject.objectKey)
			}
		}
	}
}
