package ingest

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
)

// GenesisChange returns the Change occurring at the genesis ledger (ledgerseq = 1)..
func GenesisChange(networkPassPhrase string) Change {
	rootKeyPair := keypair.Root(networkPassPhrase)

	rootAccountEntry := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId: xdr.MustAddress(rootKeyPair.Address()),
				// 100B
				Balance:    amount.MustParse("100000000000"),
				SeqNum:     0,
				Thresholds: xdr.Thresholds{1, 0, 0, 0},
			},
		},
	}

	return Change{
		Type: rootAccountEntry.Data.Type,
		Post: &rootAccountEntry,
	}
}
