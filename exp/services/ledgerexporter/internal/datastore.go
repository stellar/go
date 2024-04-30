package ledgerexporter

import (
	"context"
	"io"

	"github.com/stellar/go/support/errors"
)

// DataStore defines an interface for interacting with data storage
type DataStore interface {
	GetFile(ctx context.Context, path string) (io.ReadCloser, error)
	PutFile(ctx context.Context, path string, in io.WriterTo) error
	PutFileIfNotExists(ctx context.Context, path string, in io.WriterTo) (bool, error)
	Exists(ctx context.Context, path string) (bool, error)
	Size(ctx context.Context, path string) (int64, error)
	Close() error
}

// NewDataStore factory, it creates a new DataStore based on the config type
func NewDataStore(ctx context.Context, datastoreConfig DataStoreConfig, network string) (DataStore, error) {
	switch datastoreConfig.Type {
	case "GCS":
		return NewGCSDataStore(ctx, datastoreConfig.Params, network)
	default:
		return nil, errors.Errorf("Invalid datastore type %v, not supported", datastoreConfig.Type)
	}
}
