package adapters

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/network"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

// TestTransactionAdapter confirms that the adapter correctly serializes a
// transaction to JSON by actually pulling a transaction from the
// known-to-be-true horizon.stellar.org, turning it into an "ingested"
// transaction, and serializing it.
func TestTransactionAdapter(t *testing.T) {
	f, err := os.Open(filepath.Join("./testdata", "transactions.json"))
	require.NoErrorf(t, err, "are fixtures missing?")

	page := protocol.TransactionsPage{}
	decoder := json.NewDecoder(f)
	require.NoError(t, decoder.Decode(&page))
	require.Len(t, page.Embedded.Records, 1)
	expectedTx := page.Embedded.Records[0]

	parsedUrl, err := url.Parse(page.Links.Self.Href)
	require.NoError(t, err)
	parsedToid, err := strconv.ParseInt(expectedTx.PagingToken(), 10, 64)
	require.NoError(t, err)
	expectedTxIndex := toid.Parse(parsedToid).TransactionOrder

	txEnv := xdr.TransactionEnvelope{}
	txResult := xdr.TransactionResult{}
	txMeta := xdr.TransactionMeta{}
	txFeeMeta := xdr.LedgerEntryChanges{}

	require.NoError(t, xdr.SafeUnmarshalBase64(expectedTx.EnvelopeXdr, &txEnv))
	require.NoError(t, xdr.SafeUnmarshalBase64(expectedTx.ResultMetaXdr, &txMeta))
	require.NoError(t, xdr.SafeUnmarshalBase64(expectedTx.ResultXdr, &txResult))
	require.NoError(t, xdr.SafeUnmarshalBase64(expectedTx.FeeMetaXdr, &txFeeMeta))

	closeTimestamp := expectedTx.LedgerCloseTime.UTC().Unix()

	tx := common.Transaction{
		LedgerTransaction: &archive.LedgerTransaction{
			Index:    0,
			Envelope: txEnv,
			Result: xdr.TransactionResultPair{
				TransactionHash: xdr.Hash{},
				Result:          txResult,
			},
			FeeChanges: txFeeMeta,
			UnsafeMeta: txMeta,
		},
		LedgerHeader: &xdr.LedgerHeader{
			LedgerSeq: xdr.Uint32(expectedTx.Ledger),
			ScpValue: xdr.StellarValue{
				CloseTime: xdr.TimePoint(closeTimestamp),
			},
		},
		TxIndex:           expectedTxIndex - 1, // TOIDs have a 1-based index
		NetworkPassphrase: network.PublicNetworkPassphrase,
	}

	result, err := PopulateTransaction(parsedUrl, &tx, xdr.NewEncodingBuffer())
	require.NoError(t, err)
	assert.Equal(t, expectedTx, result)
}
