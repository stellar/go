package ledgerexporter

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/datastore"
)

// Uploader is responsible for uploading data to a storage destination.
type Uploader interface {
	Run(ctx context.Context) error
	Upload(ctx context.Context, metaArchive *LedgerMetaArchive) error
}

type uploader struct {
	dataStore     datastore.DataStore
	metaArchiveCh chan *LedgerMetaArchive
}

func NewUploader(destination datastore.DataStore, metaArchiveCh chan *LedgerMetaArchive) Uploader {
	return &uploader{
		dataStore:     destination,
		metaArchiveCh: metaArchiveCh,
	}
}

// Upload uploads the serialized binary data of ledger TxMeta to the specified destination.
// TODO: Add retry logic.
func (u *uploader) Upload(ctx context.Context, metaArchive *LedgerMetaArchive) error {
	logger.Infof("Uploading: %s", metaArchive.GetObjectKey())

	err := u.dataStore.PutFileIfNotExists(ctx, metaArchive.GetObjectKey(),
		&XDRGzipEncoder{XdrPayload: &metaArchive.data})
	if err != nil {
		return errors.Wrapf(err, "error uploading %s", metaArchive.GetObjectKey())
	}
	return nil
}

// TODO: make it configurable
var uploaderShutdownWaitTime = 10 * time.Second

// Run starts the uploader, continuously listening for LedgerMetaArchive objects to upload.
func (u *uploader) Run(ctx context.Context) error {
	uploadCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		logger.Info("Context done, waiting for remaining uploads to complete...")
		// wait for a few seconds to upload remaining objects from metaArchiveCh
		<-time.After(uploaderShutdownWaitTime)
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
				return nil
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
