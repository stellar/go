package datastore

import (
	"context"
	"io"
	"time"

	"github.com/stellar/go/support/errors"
)

type DataStoreConfig struct {
	Type   string            `toml:"type"`
	Params map[string]string `toml:"params"`
	Schema DataStoreSchema   `toml:"schema"`
}

// DataStore defines an interface for interacting with data storage
type DataStore interface {
	GetFileMetadata(ctx context.Context, path string) (map[string]string, error)
	GetFileLastModified(ctx context.Context, filePath string) (time.Time, error)
	GetFile(ctx context.Context, path string) (io.ReadCloser, error)
	PutFile(ctx context.Context, path string, in io.WriterTo, metaData map[string]string) error
	PutFileIfNotExists(ctx context.Context, path string, in io.WriterTo, metaData map[string]string) (bool, error)
	Exists(ctx context.Context, path string) (bool, error)
	Size(ctx context.Context, path string) (int64, error)
	GetSchema() DataStoreSchema
	Close() error
}

// NewDataStore factory, it creates a new DataStore based on the config type
func NewDataStore(ctx context.Context, datastoreConfig DataStoreConfig) (DataStore, error) {
	switch datastoreConfig.Type {
	case "GCS":
		return NewGCSDataStore(ctx, datastoreConfig)
	case "S3":
		return NewS3DataStore(ctx, datastoreConfig)

	default:
		return nil, errors.Errorf("Invalid datastore type %v, not supported", datastoreConfig.Type)
	}
}
