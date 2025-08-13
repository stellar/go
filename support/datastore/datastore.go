package datastore

import (
	"context"
	"fmt"
	"io"
	"time"
)

const (
	manifestFilename = ".config.json"
	Version          = "1.0"
)

// DataStoreConfig defines user-provided configuration used to initialize a DataStore.
type DataStoreConfig struct {
	Type              string            `toml:"type"`
	Params            map[string]string `toml:"params"`
	Schema            DataStoreSchema   `toml:"schema"`
	NetworkPassphrase string
	Compression       string
}

const listFilePathsMaxLimit = 1000

// DataStore defines an interface for interacting with data storage
type DataStore interface {
	GetFileMetadata(ctx context.Context, path string) (map[string]string, error)
	GetFileLastModified(ctx context.Context, filePath string) (time.Time, error)
	GetFile(ctx context.Context, path string) (io.ReadCloser, error)
	PutFile(ctx context.Context, path string, in io.WriterTo, metaData map[string]string) error
	PutFileIfNotExists(ctx context.Context, path string, in io.WriterTo, metaData map[string]string) (bool, error)
	Exists(ctx context.Context, path string) (bool, error)
	Size(ctx context.Context, path string) (int64, error)
	ListFilePaths(ctx context.Context, prefix string, limit int) ([]string, error)
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
		return nil, fmt.Errorf("invalid datastore type %v, not supported", datastoreConfig.Type)
	}
}
