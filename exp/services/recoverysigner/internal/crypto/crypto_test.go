package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncrypterDecrypter(t *testing.T) {
	ksPriv := generateHybridKeysetCleartext(t)
	enc, dec, err := NewEncrypterDecrypter("", ksPriv)
	require.NoError(t, err)
	assert.NotNil(t, enc)
	assert.NotNil(t, dec)
}

func TestNewEncrypterDecrypter_invalidKMSKeyURI(t *testing.T) {
	ksPriv := generateHybridKeysetCleartext(t)

	// URI with a valid prefix but bad invalid identifier
	enc, dec, err := NewEncrypterDecrypter("aws-kms://invalid-key-arn", ksPriv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
	assert.Nil(t, enc)
	assert.Nil(t, dec)

	// URI too short
	enc, dec, err = NewEncrypterDecrypter("aws-kms", ksPriv)
	require.Error(t, err)
	assert.EqualError(t, err, "invalid KMS key URI format")
	assert.Nil(t, enc)
	assert.Nil(t, dec)

	// URI with an invalid prefix
	enc, dec, err = NewEncrypterDecrypter("unknown-kms", ksPriv)
	require.Error(t, err)
	assert.EqualError(t, err, "unrecognized prefix in KMS Key URI")
	assert.Nil(t, enc)
	assert.Nil(t, dec)
}
