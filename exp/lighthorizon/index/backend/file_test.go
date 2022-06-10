package index

import (
	"math/rand"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
)

func TestSimpleFileStore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a large (beyond a single chunk) list of arbitrary accounts, some
	// regular and some muxed.
	accountList := make([]string, 123)
	for i, _ := range accountList {
		var err error
		var muxed xdr.MuxedAccount
		address := keypair.MustRandom().Address()

		if rand.Intn(2) == 1 {
			muxed, err = xdr.MuxedAccountFromAccountId(address, 12345678)
			require.NoErrorf(t, err, "shouldn't happen")
		} else {
			muxed = xdr.MustMuxedAddress(address)
		}

		accountList[i] = muxed.Address()
	}

	require.Len(t, accountList, 123)

	file, err := NewFileBackend(tmpDir, 1)
	require.NoError(t, err)

	require.NoError(t, file.FlushAccounts(accountList))

	accounts, err := file.ReadAccounts()
	require.NoError(t, err)
	require.Equal(t, accountList, accounts)
}
