package adapters

import (
	"encoding/json"
	"net/http"
	"net/url"
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
//
// TODO: Instead of making an unreliable network request, we should just create
// a "known" transaction on-the-fly. We could just hard-code the fields from
// Horizon, anyway.
func TestTransactionAdapter(t *testing.T) {
	const URL = "https://horizon.stellar.org/accounts/GBFHFINUD6NVGSX33PY25DDRCABN3H2JTDMLUEXAUEJVV22HTXVGLEZD/transactions?cursor=179530990183178241&limit=1&order=desc"

	parsed, err := url.Parse(URL)
	require.NoError(t, err)

	resp, err := http.Get(URL)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	page := protocol.TransactionsPage{}
	decoder := json.NewDecoder(resp.Body)
	require.NoError(t, decoder.Decode(&page))
	require.Len(t, page.Embedded.Records, 1)
	expectedTx := page.Embedded.Records[0]

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

	result, err := PopulateTransaction(parsed, &tx, xdr.NewEncodingBuffer())
	require.NoError(t, err)
	assert.Equal(t, expectedTx, result)
}
