package derive

import (
	"fmt"

	"github.com/stellar/go/keypair"
	"github.com/tyler-smith/go-bip32"
)

const StellarCoinType = 148

// GetKeyPairs generate key pairs using BIP-44 derivation for Stellar Lumens.
// Returns BIP-44 derivation path for each key pair, key pairs and error.
func GetKeyPairs(rootKey *bip32.Key, startID, count uint32) ([]string, []*keypair.Full, error) {
	currentKey, err := rootKey.NewChildKey(bip32.FirstHardenedChild + 44)
	if err != nil {
		return nil, nil, err
	}

	derivationPath := []uint32{
		bip32.FirstHardenedChild + StellarCoinType,
		bip32.FirstHardenedChild + 0,
		0,
	}

	pathString := "m/44'"
	for len(derivationPath) > 0 {
		if derivationPath[0] >= bip32.FirstHardenedChild {
			pathString += fmt.Sprintf("/%d'", derivationPath[0]-bip32.FirstHardenedChild)
		} else {
			pathString += fmt.Sprintf("/%d", derivationPath[0])
		}
		currentKey, err = currentKey.NewChildKey(derivationPath[0])
		if err != nil {
			return nil, nil, err
		}
		derivationPath = derivationPath[1:]
	}

	paths := make([]string, count)
	keypairs := make([]*keypair.Full, count)
	currentID := startID
	for i := uint32(0); i < count; i++ {
		address, err := currentKey.NewChildKey(currentID)
		if err != nil {
			return nil, nil, err
		}

		var seed [32]byte
		copy(seed[:], address.Key[:])

		paths[i] = fmt.Sprintf("%s/%d", pathString, currentID)
		keypairs[i], err = keypair.FromRawSeed(seed)
		if err != nil {
			return nil, nil, err
		}

		currentID++
	}

	return paths, keypairs, nil
}
