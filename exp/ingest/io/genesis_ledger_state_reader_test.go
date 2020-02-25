package io

import (
	"io"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestGenesisLeaderStateReader(t *testing.T) {
	stateReader := GenesisLedgerStateReader{
		NetworkPassphrase: "Public Global Stellar Network ; September 2015",
	}

	ledgerEntryChange, err := stateReader.Read()
	assert.NoError(t, err)
	assert.Equal(t, xdr.LedgerEntryTypeAccount, ledgerEntryChange.Type)
	assert.Equal(t, xdr.Uint32(1), ledgerEntryChange.Post.LastModifiedLedgerSeq)
	account := ledgerEntryChange.Post.Data.MustAccount()
	assert.Equal(t, "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7", account.AccountId.Address())
	assert.Equal(t, xdr.SequenceNumber(0), account.SeqNum)
	assert.Equal(t, xdr.Int64(1000000000000000000), account.Balance)
	assert.Equal(t, xdr.Thresholds{1, 0, 0, 0}, account.Thresholds)

	_, err = stateReader.Read()
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
}
