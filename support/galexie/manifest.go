package galexie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/stellar/go/support/datastore"
)

const (
	manifestFilename = ".config.json"
	ManifestVersion  = "1.0"
)

// Manifest represents the persisted configuration stored in the object store.
type Manifest struct {
	NetworkPassphrase   string `json:"networkPassphrase"`
	Version             string `json:"version"`
	Compression         string `json:"compression"`
	LedgersPerBatch     uint32 `json:"ledgersPerBatch"`
	BatchesPerPartition uint32 `json:"batchesPerPartition"`
}

func WriteManifest(ctx context.Context, dataStore datastore.DataStore, manifest Manifest) error {
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	ok, err := dataStore.PutFileIfNotExists(ctx, manifestFilename, bytes.NewReader(data), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to write manifest file %q: %w", manifestFilename, err)
	}
	if !ok {
		return fmt.Errorf("manifest file %q already exists", manifestFilename)
	}
	return nil
}

func GetManifest(ctx context.Context, dataStore datastore.DataStore) (Manifest, error) {
	reader, err := dataStore.GetFile(ctx, manifestFilename)
	if err != nil {
		return Manifest{}, fmt.Errorf("unable to open manifest file %q: %w", manifestFilename, err)
	}
	defer reader.Close()

	var manifest Manifest
	if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("invalid JSON in manifest file %q: %w", manifestFilename, err)
	}

	return manifest, nil
}
