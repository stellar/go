package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInsecureEncrypterDecrypter(t *testing.T) {
	ksPriv := generateHybridKeysetCleartext(t)
	enc, dec, err := newInsecureEncrypterDecrypter(ksPriv)
	require.NoError(t, err)
	assert.NotNil(t, enc)
	assert.NotNil(t, dec)

	// no keyset is present
	enc, dec, err = newInsecureEncrypterDecrypter("")
	assert.Error(t, err)
	assert.Nil(t, enc)
	assert.Nil(t, dec)

	// encrypted keyset is preset
	ksPrivEncrypted := generateHybridKeysetEncrypted(t)
	enc, dec, err = newInsecureEncrypterDecrypter(ksPrivEncrypted)
	assert.Error(t, err)
	assert.Nil(t, enc)
	assert.Nil(t, dec)
}

func TestInsecureEncrypterDecrypter_encryptDecrypt(t *testing.T) {
	ksPriv := generateHybridKeysetCleartext(t)
	enc, dec, err := newInsecureEncrypterDecrypter(ksPriv)
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
