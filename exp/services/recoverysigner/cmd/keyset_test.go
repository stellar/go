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

func TestRotateKeyset(t *testing.T) {
	public1, privateC1, _, err := createKeyset("")
	require.NoError(t, err)

	khPriv1, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(privateC1)))
	require.NoError(t, err)

	exportedPriv1 := &keyset.MemReaderWriter{}
	err = insecurecleartextkeyset.Write(khPriv1, exportedPriv1)
	require.NoError(t, err)
	assert.Len(t, exportedPriv1.Keyset.Key, 1)

	khPub1, err := keyset.ReadWithNoSecrets(keyset.NewJSONReader(strings.NewReader(public1)))
	require.NoError(t, err)

	public2, privateC2, _, err := rotateKeyset("", privateC1)
	require.NoError(t, err)

	khPriv2, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(privateC2)))
	require.NoError(t, err)

	exportedPriv2 := &keyset.MemReaderWriter{}
	err = insecurecleartextkeyset.Write(khPriv2, exportedPriv2)
	require.NoError(t, err)

	// rotation will add a new key to the keyset and make the new key the primary key
	assert.Len(t, exportedPriv2.Keyset.Key, 2)
	assert.NotEqual(t, exportedPriv2.Keyset.PrimaryKeyId, exportedPriv1.Keyset.PrimaryKeyId)

	khPub2, err := keyset.ReadWithNoSecrets(keyset.NewJSONReader(strings.NewReader(public2)))
	require.NoError(t, err)

	// verify that one is able to get the same keyset public with keyset private
	khPubv2, err := khPriv2.Public()
	require.NoError(t, err)
	assert.Equal(t, khPub2.String(), khPubv2.String())

	hd1, err := hybrid.NewHybridDecrypt(khPriv1)
	require.NoError(t, err)

	he1, err := hybrid.NewHybridEncrypt(khPub1)
	require.NoError(t, err)

	hd2, err := hybrid.NewHybridDecrypt(khPriv2)
	require.NoError(t, err)

	he2, err := hybrid.NewHybridEncrypt(khPub2)
	require.NoError(t, err)

	plaintext := []byte("secure message")
	contextInfo := []byte("context info")

	// verify that the new keyset private is able to decrypt what the new keyset public encrypts
	ciphertext, err := he2.Encrypt(plaintext, contextInfo)
	require.NoError(t, err)

	plaintext2, err := hd2.Decrypt(ciphertext, contextInfo)
	require.NoError(t, err)
	assert.Equal(t, plaintext, plaintext2)

	// context info not matching will result in a failed decryption
	_, err = hd2.Decrypt(ciphertext, []byte("wrong info"))
	assert.Error(t, err)

	// verify that the new keyset is able to decrypt what the old keyset encrypts
	ciphertext, err = he1.Encrypt(plaintext, contextInfo)
	require.NoError(t, err)

	plaintext2, err = hd2.Decrypt(ciphertext, contextInfo)
	require.NoError(t, err)
	assert.Equal(t, plaintext, plaintext2)

	// context info not matching will still result in a failed decryption
	_, err = hd2.Decrypt(ciphertext, []byte("wrong info"))
	assert.Error(t, err)

	// verify that the old keyset is not able to decrypt what the new keyset encrypts
	ciphertext, err = he2.Encrypt(plaintext, contextInfo)
	require.NoError(t, err)

	_, err = hd1.Decrypt(ciphertext, contextInfo)
	assert.Error(t, err)
}

func TestRotateKeyset_invalidKMSKeyURI(t *testing.T) {
	_, privateC, _, err := createKeyset("")
	require.NoError(t, err)

	_, _, _, err = rotateKeyset("invalid-uri", privateC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}

func TestRotateKeyset_noCurrentKeyset(t *testing.T) {
	_, _, _, err := rotateKeyset("", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting key handle for private key")
}
