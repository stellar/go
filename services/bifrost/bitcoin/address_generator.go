package bitcoin

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/stellar/go/support/errors"
	"github.com/tyler-smith/go-bip32"
)

// TODO should we use account hardened key and then use it to generate change and index keys?
// That way we can create lot more accounts than 0x80000000-1.
func NewAddressGenerator(masterPublicKeyString string) (*AddressGenerator, error) {
	deserializedMasterPublicKey, err := bip32.B58Deserialize(masterPublicKeyString)
	if err != nil {
		return nil, errors.Wrap(err, "Error deserializing master public key")
	}

	if deserializedMasterPublicKey.IsPrivate {
		return nil, errors.New("Key is not a master public key")
	}

	return &AddressGenerator{deserializedMasterPublicKey}, nil
}

func (g *AddressGenerator) Generate(index uint32) (string, error) {
	if g.masterPublicKey == nil {
		return "", errors.New("No master public key set")
	}

	accountKey, err := g.masterPublicKey.NewChildKey(index)
	if err != nil {
		return "", errors.Wrap(err, "Error creating new child key")
	}

	address, err := btcutil.NewAddressPubKey(accountKey.Key, &chaincfg.MainNetParams)
	if err != nil {
		return "", errors.Wrap(err, "Error creating address for new child key")
	}

	return address.AddressPubKeyHash().EncodeAddress(), nil
}
