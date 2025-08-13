package galexie

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/support/datastore"
)

func TestWriteManifest(t *testing.T) {
	type tc struct {
		name           string
		manifest       Manifest
		putFileSuccess bool
		putFileErr     error
		expectedErr    error
	}

	cases := []tc{
		{
			name: "succeeds when manifest is written successfully",
			manifest: Manifest{
				NetworkPassphrase:   "Test Passphrase",
				Version:             "1.0",
				Compression:         "zst",
				LedgersPerBatch:     100,
				BatchesPerPartition: 10,
			},
			putFileSuccess: true,
			expectedErr:    nil,
		},
		{
			name: "fails when manifest file already exists",
			manifest: Manifest{
				NetworkPassphrase:   "Test Passphrase",
				Version:             "1.0",
				Compression:         "zst",
				LedgersPerBatch:     100,
				BatchesPerPartition: 10,
			},
			putFileSuccess: false,
			expectedErr:    fmt.Errorf("manifest file %q already exists", manifestFilename),
		},
		{
			name: "fails when PutFileIfNotExists returns an error",
			manifest: Manifest{
				NetworkPassphrase:   "Test Passphrase",
				Version:             "1.0",
				Compression:         "zst",
				LedgersPerBatch:     100,
				BatchesPerPartition: 10,
			},
			putFileErr:  fmt.Errorf("data store error"),
			expectedErr: fmt.Errorf("failed to write manifest file %q: data store error", manifestFilename),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ds := new(datastore.MockDataStore)

			var capturedData bytes.Buffer
			ds.On("PutFileIfNotExists", mock.Anything, manifestFilename, mock.Anything, map[string]string{
				"Content-Type": "application/json",
			}).Run(func(args mock.Arguments) {
				if buffer, ok := args[2].(io.WriterTo); ok {
					_, err := buffer.WriteTo(&capturedData)
					require.NoError(t, err)
				} else {
					require.Fail(t, "expected io.WriterTo")
				}
			}).Return(tt.putFileSuccess, tt.putFileErr).Once()

			err := WriteManifest(context.Background(), ds, tt.manifest)

			if tt.expectedErr != nil {
				require.ErrorContains(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				expectedJSON := fmt.Sprintf(`{"networkPassphrase":"%s","version":"%s","compression":"%s","ledgersPerBatch":%d,"batchesPerPartition":%d}`,
					tt.manifest.NetworkPassphrase,
					tt.manifest.Version,
					tt.manifest.Compression,
					tt.manifest.LedgersPerBatch,
					tt.manifest.BatchesPerPartition,
				)
				require.JSONEq(t, expectedJSON, capturedData.String())
			}
			ds.AssertExpectations(t)
		})
	}
}
