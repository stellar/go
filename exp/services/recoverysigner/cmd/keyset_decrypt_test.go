package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptKeyset_invalidKMSKeyURI(t *testing.T) {
	_, _, err := decryptKeyset("invalid-uri", "keysetJSON")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}
