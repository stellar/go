package internal

import (
	"sync"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/crypto/derivation"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
)

// Channel contains current state of the channel account and provides methods to reload and read sequence number
type Channel struct {
	Seed string

	mutex          sync.Mutex
	accountID      string
	sequenceNumber int64
}

// Derives n channel accounts from seed, starting at offset i
func DeriveChannelsFromSeed(seed string, n uint32, i uint32) (channels []*Channel, err error) {
	bytes, err := strkey.Decode(strkey.VersionByteSeed, seed)
	if err != nil {
		return channels, err
	}
	key, err := derivation.DeriveForPath(derivation.StellarPrimaryAccountPath, bytes)
	if err != nil {
		return channels, err
	}
	var j uint32
	for j = i; j < i+n; j++ {
		derivedKey, err := key.Derive(j + derivation.FirstHardenedIndex)
		if err != nil {
			return channels, err
		}
		derivedKeypair, err := keypair.FromRawSeed(derivedKey.RawSeed())
		if err != nil {
			return channels, err
		}
		channels = append(channels, &Channel{
			Seed: derivedKeypair.Seed(),
		})
	}
	return channels, nil
}

// ReloadState loads the current state of the channel account using given horizon client
func (ch *Channel) LoadState(client horizonclient.ClientInterface) (err error) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	kp, err := keypair.Parse(ch.Seed)
	if err != nil {
		return err
	}

	account, err := client.AccountDetail(horizonclient.AccountRequest{AccountID: kp.Address()})
	if err != nil {
		return err
	}

	ch.accountID = account.ID
	ch.sequenceNumber = account.Sequence
	return nil
}

// GetSequenceNumber increments sequence number in an atomic operation and returns a new sequence number
func (ch *Channel) GetSequenceNumber() int64 {
	ch.mutex.Lock()
	ch.sequenceNumber++
	sequenceNumberCopy := ch.sequenceNumber
	ch.mutex.Unlock()
	return sequenceNumberCopy
}

// GetAccountID returns channel's account ID
func (ch *Channel) GetAccountID() string {
	return ch.accountID
}
