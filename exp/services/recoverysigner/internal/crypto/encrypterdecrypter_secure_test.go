package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecureEncrypterDecrypter(t *testing.T) {
	ksPriv := generateHybridKeysetEncrypted(t)
	enc, dec, err := newSecureEncrypterDecrypter(mockKMSClient{}, "aws-kms://key-uri", ksPriv)
	require.NoError(t, err)
	assert.NotNil(t, enc)
	assert.NotNil(t, dec)

	enc, dec, err = newSecureEncrypterDecrypter(mockKMSClient{}, "mock-key-uri", "")
	assert.Error(t, err)
	assert.Nil(t, enc)
	assert.Nil(t, dec)
}

func TestSecureEncrypterDecrypter_encryptDecrypt(t *testing.T) {
	ksPriv := generateHybridKeysetEncrypted(t)
	enc, dec, err := newSecureEncrypterDecrypter(mockKMSClient{}, "mock-key-uri", ksPriv)
	require.NoError(t, err)

	plaintext := []byte("secure message")
	contextInfo := []byte("context info")
	ciphertext, err := enc.Encrypt(plaintext, contextInfo)
	require.NoError(t, err)

	plaintext2, err := dec.Decrypt(ciphertext, contextInfo)
	require.NoError(t, err)
	assert.Equal(t, plaintext, plaintext2)

	// context info not matching will result in a failed decryption
	_, err = dec.Decrypt(ciphertext, []byte("wrong info"))
	assert.Error(t, err)
}
