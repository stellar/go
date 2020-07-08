package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptKeyset_invalidKMSKeyURI(t *testing.T) {
	_, _, err := encryptKeyset("invalid-uri", "keysetJSON")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}
