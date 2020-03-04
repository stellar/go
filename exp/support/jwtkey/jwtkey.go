// Package jwtkey provides utility functions for generating, serializing and
// deserializing JWT ECDSA keys.
//
// TODO: Replace EC function usages with PKCS8 functions for supporting ECDSA
// and RSA keys instead of only supporting ECDSA. The fact this package only
// supports ECDSA is unnecessary.
package jwtkey

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"

	"github.com/stellar/go/support/errors"
)

// GenerateKey is a convenience function for generating an ECDSA key for use as
// a JWT key. It uses the P256 curve. To use other curves use the crypto/ecdsa
// package directly.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generating ECDSA key")
	}
	return k, nil
}

// PrivateKeyToString converts a ECDSA private key into a ASN.1 DER and base64
// encoded string.
func PrivateKeyToString(k *ecdsa.PrivateKey) (string, error) {
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return "", errors.Wrap(err, "marshaling ECDSA private key")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// PublicKeyToString converts a ECDSA public key into a ASN.1 DER and base64
// encoded string.
func PublicKeyToString(k *ecdsa.PublicKey) (string, error) {
	b, err := x509.MarshalPKIXPublicKey(k)
	if err != nil {
		return "", errors.Wrap(err, "marshaling ECDSA public key")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// PrivateKeyFromString converts a ECDSA private key from a ASN.1 DER and
// base64 encoded string into a type.
func PrivateKeyFromString(s string) (*ecdsa.PrivateKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.Wrap(err, "base64 decoding ECDSA private key")
	}
	key, err := x509.ParseECPrivateKey(keyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling ECDSA private key")
	}
	return key, nil
}

// PublicKeyFromString converts a ECDSA public key from a ASN.1 DER and base64
// encoded string into a type.
func PublicKeyFromString(s string) (*ecdsa.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.Wrap(err, "base64 decoding ECDSA public key")
	}
	keyI, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling ECDSA public key")
	}
	key, ok := keyI.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.Wrap(err, "public key not ECDSA key")
	}
	return key, nil
}
