package exporter

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Uploader is responsible for uploading data to a storage destination.
type Uploader interface {
	Run(ctx context.Context) error
	Upload(ctx context.Context, metaArchive *LedgerMetaArchive) error
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
func (u *uploader) Upload(ctx context.Context, metaArchive *LedgerMetaArchive) error {
	logger.Infof("Uploading: %s", metaArchive.GetObjectKey())

	err := u.destination.PutFileIfNotExists(ctx, metaArchive.GetObjectKey(),
		&XDRGzipEncoder{XdrPayload: &metaArchive.data})
	if err != nil {
		return errors.Wrapf(err, "error uploading %s", metaArchive.GetObjectKey())
	}
	return nil
}

// Run starts the uploader, continuously listening for LedgerMetaArchive objects to upload.
func (u *uploader) Run(ctx context.Context) error {
	uploadCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		logger.Info("Context done, waiting for remaining uploads to complete...")
		// allow up to 10 seconds to upload remaining objects from metaArchiveCh
		<-time.After(10 * time.Second)
		logger.Info("Timeout reached, canceling remaining uploads...")
		cancel()
	}()

	for {
		select {
		case <-uploadCtx.Done():
			return uploadCtx.Err()

		case metaObject, ok := <-u.metaArchiveCh:
			if !ok {
				logger.Info("Meta archive channel closed, stopping uploader")
				return errors.New("Meta archive channel closed")
			}
			//Upload the received LedgerMetaArchive.
			err := u.Upload(uploadCtx, metaObject)
			if err != nil {
				return err
			}
			logger.Infof("Uploaded %s successfully", metaObject.objectKey)
		}
	}
}
