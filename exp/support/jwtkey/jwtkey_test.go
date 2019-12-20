package jwtkey

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)
	assert.Equal(t, elliptic.P256(), key.Curve)
}

func TestToFromStringRoundTrip(t *testing.T) {
	testCases := []struct {
		Name  string
		Curve elliptic.Curve
	}{
		{Name: "P224", Curve: elliptic.P224()},
		{Name: "P256", Curve: elliptic.P256()},
		{Name: "P384", Curve: elliptic.P384()},
		{Name: "P521", Curve: elliptic.P521()},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			privateKey, err := ecdsa.GenerateKey(tc.Curve, rand.Reader)
			require.NoError(t, err)

			t.Run("private", func(t *testing.T) {
				privateKeyStr, err := PrivateKeyToString(privateKey)
				require.NoError(t, err)

				// Private key as string should be valid standard base64
				_, err = base64.StdEncoding.DecodeString(privateKeyStr)
				require.NoError(t, err)

				// Private key should decode back to the original
				privateKeyRoundTripped, err := PrivateKeyFromString(privateKeyStr)
				require.NoError(t, err)
				assert.Equal(t, privateKey, privateKeyRoundTripped)
			})

			publicKey := &privateKey.PublicKey

			t.Run("public", func(t *testing.T) {
				publicKeyStr, err := PublicKeyToString(publicKey)
				require.NoError(t, err)

				// Public key as string should be valid standard base64
				_, err = base64.StdEncoding.DecodeString(publicKeyStr)
				require.NoError(t, err)

				// Public key should decode back to the original
				publicKeyRoundTripped, err := PublicKeyFromString(publicKeyStr)
				require.NoError(t, err)
				assert.Equal(t, publicKey, publicKeyRoundTripped)
			})
		})
	}
}
