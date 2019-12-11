package history

import (
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var (
	ledger = int64(4294967296) // ledger sequence 1
	tx     = int64(4096)       // tx index 1
	op     = int64(1)          // op index 1
)

func TestTransactionOperationID(t *testing.T) {
	tt := assert.New(t)
	transaction, err := buildTransaction(
		1,
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:          1,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 1,
	}

	tt.Equal(ledger+tx+op, operation.ID())
}

func TestTransactionOperationTransactionID(t *testing.T) {
	tt := assert.New(t)
	transaction, err := buildTransaction(
		1,
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:          1,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 1,
	}

	tt.Equal(ledger+tx, operation.TransactionID())
}

func TestOperationTransactionSourceAccount(t *testing.T) {
	testCases := []struct {
		desc          string
		sourceAccount string
		expected      string
	}{
		{
			desc:          "Source account is same as transaction",
			sourceAccount: "",
			expected:      "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
		},
		{
			desc:          "Source account is different to transaction",
			sourceAccount: "GDMQUXK7ZUCWM5472ZU3YLDP4BMJLQQ76DEMNYDEY2ODEEGGRKLEWGW2",
			expected:      "GDMQUXK7ZUCWM5472ZU3YLDP4BMJLQQ76DEMNYDEY2ODEEGGRKLEWGW2",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tt := assert.New(t)
			transaction, err := buildTransaction(
				1,
				"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
			)
			tt.NoError(err)
			op := transaction.Envelope.Tx.Operations[0]
			if len(tc.sourceAccount) > 0 {
				sourceAccount := xdr.MustAddress(tc.sourceAccount)
				op.SourceAccount = &sourceAccount
			}

			operation := TransactionOperation{
				Index:          1,
				Transaction:    transaction,
				Operation:      op,
				LedgerSequence: 1,
			}

			tt.Equal(tc.expected, operation.SourceAccount().Address())
		})
	}
}

func TestTransactionOperationType(t *testing.T) {
	tt := assert.New(t)
	transaction, err := buildTransaction(
		1,
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:          1,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 1,
	}

	tt.Equal(xdr.OperationTypePayment, operation.OperationType())
}

func TestTransactionOperationDetails(t *testing.T) {
	tt := assert.New(t)
	transaction, err := buildTransaction(
		1,
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:          1,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 1,
	}

	expected := map[string]interface{}{
		"asset_type": "native",
		"from":       "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
		"to":         "GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A",
		"amount":     "32.7000000",
	}

	tt.Equal(expected, operation.Details())
}

func buildTransaction(index uint32, envelope string) (io.LedgerTransaction, error) {
	transaction := io.LedgerTransaction{
		Index:    1,
		Envelope: xdr.TransactionEnvelope{},
	}
	err := xdr.SafeUnmarshalBase64(
		envelope,
		&transaction.Envelope,
	)
	if err != nil {
		return transaction, err
	}

	return transaction, nil
}
