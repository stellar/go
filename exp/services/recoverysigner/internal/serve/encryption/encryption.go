package encryption

import (
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"

	"github.com/pkg/errors"
)

func parsePublicKey(pk []byte) (crypto.PublicKey, error) {
	pub, err := x509.ParsePKIXPublicKey(pk)
	if err != nil {
		return nil, errors.Wrap(err, "parsing public key in PKIX")
	}

	switch t := pub.(type) {
	case *rsa.PublicKey:
		return t, nil
	case *dsa.PublicKey:
		return t, nil
	case *ecdsa.PublicKey:
		return t, nil
	case ed25519.PublicKey:
		return t, nil
	default:
		return nil, errors.Errorf("unknown key type %T", t)
	}
}
