package crypto

import (
	"strings"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/stellar/go/support/errors"
)

func newInsecureEncrypterDecrypter(tinkKeysetJSON string) (Encrypter, Decrypter, error) {
	khPriv, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(tinkKeysetJSON)))
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting key handle for private key")
	}

	hd, err := hybrid.NewHybridDecrypt(khPriv)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting hybrid decryption primitive")
	}

	khPub, err := khPriv.Public()
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting key handle for public key")
	}

	he, err := hybrid.NewHybridEncrypt(khPub)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting hybrid encryption primitive")
	}

	return he, hd, nil
}
