package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/stellar/go/support/storage"
)

// Uploader is responsible for uploading data to a storage destination.
type Uploader interface {
	Run(ctx context.Context) error
	Upload(metaObject *LedgerCloseMetaObject) error
}

type uploader struct {
	destination    storage.Storage
	exportObjectCh chan *LedgerCloseMetaObject
}

func NewUploader(destination storage.Storage, exportObjectCh chan *LedgerCloseMetaObject) Uploader {
	return &uploader{
		destination:    destination,
		exportObjectCh: exportObjectCh,
	}
}

// Upload uploads the serialized binary data of ledger TxMeta
// to the specified destination
// TODO: Add retry logic.
func (u *uploader) Upload(metaObject *LedgerCloseMetaObject) error {
	logger.Infof("Uploading: %s", metaObject.objectKey)

	blob, err := metaObject.data.MarshalBinary()
	if err != nil {
		return err
	}

	err = u.destination.PutFile(metaObject.objectKey, io.NopCloser(bytes.NewReader(blob)))
	if err != nil {
		logger.Errorf("Error uploading %s: %v", metaObject.objectKey, err)
	}
	return nil
}

// Run starts the uploader goroutine
func (u *uploader) Run(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			// Drain the channel
			for exportObj := range u.exportObjectCh {
				err := u.Upload(exportObj)
				if err != nil {
					logger.Errorf("Error uploading %s. %v", err, exportObj.objectKey)
				}
			}
			logger.Info("Uploader stopped due to context cancellation.")
			return nil
		case metaObject, ok := <-u.exportObjectCh:
			if !ok {
				//The channel is closed
				return fmt.Errorf("export object channel closed. Uploader exiting")
			}
			//Upload the received LedgerCloseMetaObject.
			err := u.Upload(metaObject)
			if err != nil {
				return errors.Wrapf(err, "error uploading %s", metaObject.objectKey)
			}
		}
	}
}
