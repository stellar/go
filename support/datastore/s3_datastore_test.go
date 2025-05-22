package datastore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS3DataStore_MissingParams(t *testing.T) {
	ctx := context.Background()
	schema := DataStoreSchema{LedgersPerFile: 64}

	// Missing bucket
	paramsMissingBucket := map[string]string{"region": "us-east-1"}
	_, err := NewS3DataStore(ctx, paramsMissingBucket, schema)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter: destination_bucket")

	// Missing region
	paramsMissingRegion := map[string]string{"destination_bucket": "test-bucket"}
	_, err = NewS3DataStore(ctx, paramsMissingRegion, schema)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter: region")
}

// Note: Testing successful creation requires mocking AWS config loading or having valid
// dummy credentials available in the test environment (e.g., via env vars).
// This basic test focuses on parameter validation.
// More comprehensive tests would mock the S3 client interactions.

func TestS3DataStore_GetSchema(t *testing.T) {
	// This test verifies GetSchema returns the schema passed during construction.
	// We create a dummy store instance directly to avoid AWS config issues.
	schema := DataStoreSchema{LedgersPerFile: 128, FilesPerPartition: 5}
	store := &S3DataStore{
		schema: schema, // Set schema directly for testing
	}

	retrievedSchema := store.GetSchema()
	assert.Equal(t, schema, retrievedSchema)
}

func TestS3DataStore_FullPath(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		objectPath string
		expected   string
	}{
		{
			name:       "No prefix",
			prefix:     "",
			objectPath: "path/to/object",
			expected:   "path/to/object",
		},
		{
			name:       "With prefix",
			prefix:     "data/prefix",
			objectPath: "path/to/object",
			expected:   "data/prefix/path/to/object",
		},
		{
			name:       "Prefix with trailing slash",
			prefix:     "data/prefix/",
			objectPath: "path/to/object",
			expected:   "data/prefix/path/to/object",
		},
		{
			name:       "Object path with leading slash (cleaned by path.Join)",
			prefix:     "data/prefix",
			objectPath: "/path/to/object",
			expected:   "data/prefix/path/to/object", // path.Join handles the extra slash
		},
		{
			name:       "Empty object path",
			prefix:     "data/prefix",
			objectPath: "",
			expected:   "data/prefix",
		},
		{
			name:       "Prefix and object path are single components",
			prefix:     "prefix",
			objectPath: "object",
			expected:   "prefix/object",
		},
		{
			name:       "Prefix is empty, object path has leading slash",
			prefix:     "",
			objectPath: "/object",
			expected:   "object", // strings.TrimPrefix will remove it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &S3DataStore{prefix: tt.prefix}
			actual := store.fullPath(tt.objectPath)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
