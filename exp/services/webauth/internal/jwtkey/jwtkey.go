// Package jwtkey provide utility functions for generating, serializing and
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
	"fmt"
)

// GenerateKey is a convenience function for generating an ECDSA key for use as
// a JWT key. It uses the P256 curve. To use other curves use the crypto/ecdsa
// package directly.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return k, nil
}

// PrivateKeyToString converts a ECDSA private key into a ASN.1 DER and base64
// encoded string.
func PrivateKeyToString(k *ecdsa.PrivateKey) (string, error) {
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return "", fmt.Errorf("marshaling ECDSA private key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// PublicKeyToString converts a ECDSA public key into a ASN.1 DER and base64
// encoded string.
func PublicKeyToString(k *ecdsa.PublicKey) (string, error) {
	b, err := x509.MarshalPKIXPublicKey(k)
	if err != nil {
		return "", fmt.Errorf("marshaling ECDSA public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// PrivateKeyFromString converts a ECDSA private key from a ASN.1 DER and
// base64 encoded string into a type.
func PrivateKeyFromString(s string) (*ecdsa.PrivateKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding ECDSA private key: %w", err)
	}
	key, err := x509.ParseECPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling ECDSA private key: %w", err)
	}
	return key, nil
}

// PublicKeyFromString converts a ECDSA public key from a ASN.1 DER and base64
// encoded string into a type.
func PublicKeyFromString(s string) (*ecdsa.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding ECDSA public key: %w", err)
	}
	keyI, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling ECDSA public key: %w", err)
	}
	key, ok := keyI.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key not ECDSA key")
	}
	return key, nil
}
