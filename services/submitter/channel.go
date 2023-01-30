package main

import (
	"sync"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
)

// Channel contains current state of the channel account and provides methods to reload and read sequence number
type Channel struct {
	Seed string

	mutex          sync.Mutex
	accountID      string
	sequenceNumber int64
}

// ReloadState loads the current state of the channel account using given horizon client
func (ch *Channel) LoadState(client horizonclient.ClientInterface) (accountID string, sequenceNumber uint64, err error) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	kp, err := keypair.Parse(ch.Seed)
	if err != nil {
		return
	}

	account, err := client.AccountDetail(horizonclient.AccountRequest{AccountID: kp.Address()})
	if err != nil {
		return
	}

	ch.accountID = account.ID
	ch.sequenceNumber = account.Sequence
	return
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
