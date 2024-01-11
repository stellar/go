package main

import (
	"bytes"
	"context"
	"io"

	"github.com/stellar/go/support/storage"
)

// Uploader is responsible for uploading data to a storage destination.
type Uploader struct {
	destination             storage.Storage
	ledgerCloseMetaObjectCh chan *LedgerCloseMetaObject
}

func NewUploader(destination storage.Storage, ledgerCloseMetaObjectCh chan *LedgerCloseMetaObject) *Uploader {
	return &Uploader{destination: destination, ledgerCloseMetaObjectCh: ledgerCloseMetaObjectCh}
}

// Upload uploads the serialized binary data of ledger TxMeta
// to the specified destination
func (u *Uploader) Upload(metaObject *LedgerCloseMetaObject) error {
	logger.Infof("Uploading: %s", metaObject.objectKey)

	blob, err := metaObject.data.MarshalBinary()
	if err != nil {
		return err
	}

	err = u.destination.PutFile(metaObject.objectKey, io.NopCloser(bytes.NewReader(blob)))
	if err != nil {
		logger.Errorf("Error uploading %v. %s", err, metaObject.objectKey)
	}
	return nil
}

// Run starts the uploader goroutine
func (u *Uploader) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("Uploader stopped due to context cancellation.")
			return
		case metaObject, ok := <-u.ledgerCloseMetaObjectCh:
			if !ok {
				// The channel is closed, indicating no more LedgerCloseMetaObjects will be sent.
				return
			}
			// Upload the received LedgerCloseMetaObject.
			err := u.Upload(metaObject)
			if err != nil {
				// Handle error
			}
		}
	}
}
