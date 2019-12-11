package history

import (
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var ledger = int64(4294967296) // ledger sequence 1
var tx = int64(4096)           // tx index 1
var op = int64(1)              // op index 1

func TestTransactionOperationID(t *testing.T) {
	tt := assert.New(t)
	transaction := io.LedgerTransaction{
		Index:    1,
		Envelope: xdr.TransactionEnvelope{},
	}
	err := xdr.SafeUnmarshalBase64(
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
		&transaction.Envelope,
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:       1,
		Transaction: transaction,
		Operation:   transaction.Envelope.Tx.Operations[0],
	}

	tt.Equal(ledger+tx+op, operation.ID(1))
}
