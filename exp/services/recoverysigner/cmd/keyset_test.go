package cmd

import (
	"strings"
	"testing"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateKeyset(t *testing.T) {
	public, privateC, privateE, err := createKeyset("")
	require.NoError(t, err)
	assert.NotEmpty(t, public)
	assert.NotEmpty(t, privateC)
	assert.Empty(t, privateE)

	khPriv, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(privateC)))
	require.NoError(t, err)

	khPub, err := keyset.ReadWithNoSecrets(keyset.NewJSONReader(strings.NewReader(public)))
	require.NoError(t, err)

	// verify that one is able to get the same keyset public with keyset private
	khPubv, err := khPriv.Public()
	require.NoError(t, err)
	assert.Equal(t, khPub.String(), khPubv.String())

	hd, err := hybrid.NewHybridDecrypt(khPriv)
	require.NoError(t, err)

	he, err := hybrid.NewHybridEncrypt(khPub)
	require.NoError(t, err)

	// verify that one can use keyset private to decrypt what keyset public
	// encrypts
	plaintext := []byte("secure message")
	contextInfo := []byte("context info")
	ciphertext, err := he.Encrypt(plaintext, contextInfo)
	require.NoError(t, err)

	plaintext2, err := hd.Decrypt(ciphertext, contextInfo)
	require.NoError(t, err)
	assert.Equal(t, plaintext, plaintext2)

	// context info not matching will result in a failed decryption
	_, err = hd.Decrypt(ciphertext, []byte("wrong info"))
	assert.Error(t, err)
}

func TestCreateKeyset_invalidKMSKeyURI(t *testing.T) {
	_, _, _, err := createKeyset("invalid-uri")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}
