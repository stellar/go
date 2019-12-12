package history

import (
	"encoding/hex"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestAddOperation(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	builder := q.NewOperationBatchInsertBuilder(1)

	transactionResult := "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA="
	transaction, err := buildTransaction(
		1,
		"AAAAABpcjiETZ0uhwxJJhgBPYKWSVJy2TZ2LI87fqV1cUf/UAAAAZAAAADcAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAAAAAAAAAX14QAAAAAAAAAAAVxR/9QAAABAK6pcXYMzAEmH08CZ1LWmvtNDKauhx+OImtP/Lk4hVTMJRVBOebVs5WEPj9iSrgGT0EswuDCZ2i5AEzwgGof9Ag==",
		&transactionResult,
		nil,
		nil,
	)

	err = xdr.SafeUnmarshalBase64(
		"AAAAAQAAAAIAAAADAAAAOAAAAAAAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAACVAvjnAAAADcAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAOAAAAAAAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAACVAvjnAAAADcAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA==",
		&transaction.Meta,
	)
	tt.Assert.NoError(err)

	transactionHash := "2a805712c6d10f9e74bb0ccf54ae92a2b4b1e586451fe8133a2433816f6b567c"

	_, err = hex.Decode(transaction.Result.TransactionHash[:], []byte(transactionHash))
	tt.Assert.NoError(err)

	sequence := int32(56)

	insertTransaction(tt, q, "exp_history_transactions", transaction, sequence)

	operation := TransactionOperation{
		Index:          0,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: uint32(sequence),
	}

	err = builder.Add(operation)
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)

	ops := []Operation{}
	err = q.Select(&ops, selectExpOperation)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 1)

		op := ops[0]
		tt.Assert.Equal(int64(240518172673), op.ID)
		tt.Assert.Equal(int64(240518172672), op.TransactionID)
		tt.Assert.Equal(transactionHash, op.TransactionHash)
		tt.Assert.Equal(transactionResult, op.TxResult)
		tt.Assert.Equal(int32(1), op.ApplicationOrder)
		tt.Assert.Equal(xdr.OperationTypePayment, op.Type)
		tt.Assert.Equal(
			"{\"to\": \"GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y\", \"from\": \"GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y\", \"amount\": \"10.0000000\", \"asset_type\": \"native\"}",
			op.DetailsString.String,
		)
		tt.Assert.Equal("GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y", op.SourceAccount)
		tt.Assert.Equal(true, *op.TransactionSuccessful)
	}
}
