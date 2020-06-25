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
	public, privateC, privateE, err := createKeyset("", hybrid.ECIESHKDFAES128GCMKeyTemplate())
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
	_, _, _, err := createKeyset("invalid-uri", hybrid.ECIESHKDFAES128GCMKeyTemplate())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}

func TestRotateKeyset(t *testing.T) {
	keyTemplate := hybrid.ECIESHKDFAES128GCMKeyTemplate()
	public1, privateC1, _, err := createKeyset("", keyTemplate)
	require.NoError(t, err)

	khPriv1, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(privateC1)))
	require.NoError(t, err)

	exportedPriv1 := &keyset.MemReaderWriter{}
	err = insecurecleartextkeyset.Write(khPriv1, exportedPriv1)
	require.NoError(t, err)

	ksPriv1 := exportedPriv1.Keyset
	assert.Len(t, ksPriv1.Key, 1)
	assert.Equal(t, ksPriv1.PrimaryKeyId, ksPriv1.Key[0].KeyId)

	khPub1, err := keyset.ReadWithNoSecrets(keyset.NewJSONReader(strings.NewReader(public1)))
	require.NoError(t, err)

	public2, privateC2, privateE2, err := rotateKeyset("", privateC1, keyTemplate)
	require.NoError(t, err)
	assert.NotEmpty(t, public2)
	assert.NotEmpty(t, privateC2)
	assert.Empty(t, privateE2)

	khPriv2, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(privateC2)))
	require.NoError(t, err)

	exportedPriv2 := &keyset.MemReaderWriter{}
	err = insecurecleartextkeyset.Write(khPriv2, exportedPriv2)
	require.NoError(t, err)

	// rotation will add a new key to the keyset and make the new key the primary key
	ksPriv2 := exportedPriv2.Keyset
	assert.Len(t, ksPriv2.Key, 2)
	assert.Equal(t, ksPriv2.PrimaryKeyId, ksPriv2.Key[1].KeyId)
	assert.NotEqual(t, ksPriv2.PrimaryKeyId, ksPriv1.PrimaryKeyId)

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
	keyTemplate := hybrid.ECIESHKDFAES128GCMKeyTemplate()
	_, privateC, _, err := createKeyset("", keyTemplate)
	require.NoError(t, err)

	_, _, _, err = rotateKeyset("invalid-uri", privateC, keyTemplate)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}

func TestRotateKeyset_noEncryptionTinkKeyset(t *testing.T) {
	_, _, _, err := rotateKeyset("", "", hybrid.ECIESHKDFAES128GCMKeyTemplate())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting key handle for keyset private by reading a cleartext keyset")
}

func TestDecryptKeyset_invalidKMSKeyURI(t *testing.T) {
	// encrption-kms-key-uri is not configured
	_, _, err := decryptKeyset("", "keysetJSON")
	require.Error(t, err)
	assert.Equal(t, err, errNoKMSKeyURI)

	_, _, err = decryptKeyset("invalid-uri", "keysetJSON")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}

func TestEncryptKeyset_invalidKMSKeyURI(t *testing.T) {
	// encrption-kms-key-uri is not configured
	_, _, err := encryptKeyset("", "keysetJSON")
	require.Error(t, err)
	assert.Equal(t, err, errNoKMSKeyURI)

	_, _, err = encryptKeyset("invalid-uri", "keysetJSON")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initializing AWS KMS client")
}
