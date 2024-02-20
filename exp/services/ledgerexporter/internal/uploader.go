package exporter

import (
	"context"

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

func NewUploader(destination DataStore, metaArchiveCh chan *LedgerMetaArchive) Uploader {
	return &uploader{
		destination:   destination,
		metaArchiveCh: metaArchiveCh,
	}
}

// Upload uploads the serialized binary data of ledger TxMeta to the specified destination.
// TODO: Add retry logic.
func (u *uploader) Upload(metaArchive *LedgerMetaArchive) error {
	logger.Infof("Uploading: %s", metaArchive.GetObjectKey())

	err := u.destination.PutFileIfNotExists(metaArchive.GetObjectKey(), &XDRGzipEncoder{XdrPayload: &metaArchive.data})
	if err != nil {
		return errors.Wrapf(err, "error uploading %s", metaArchive.GetObjectKey())
	}
	return nil
}

// Run starts the uploader, continuously listening for LedgerMetaArchive objects to upload.
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
				logger.Info("Export object channel closed, stopping uploader")
				return nil
			}
			//Upload the received LedgerMetaArchive.
			err := u.Upload(metaObject)
			if err != nil {
				return errors.Wrapf(err, "error uploading %s", metaObject.objectKey)
			}
		}
	}
}
