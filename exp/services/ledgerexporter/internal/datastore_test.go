package ledgerexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvalidStore(t *testing.T) {
	_, err := NewDataStore(context.Background(), DataStoreConfig{Type: "unknown"}, "test", LedgerBatchConfig{})
	require.Error(t, err)
}
