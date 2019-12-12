package history

import (
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

	result := "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA="
	transaction, err := buildTransaction(
		1,
		"AAAAABpcjiETZ0uhwxJJhgBPYKWSVJy2TZ2LI87fqV1cUf/UAAAAZAAAADcAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAAAAAAAAAX14QAAAAAAAAAAAVxR/9QAAABAK6pcXYMzAEmH08CZ1LWmvtNDKauhx+OImtP/Lk4hVTMJRVBOebVs5WEPj9iSrgGT0EswuDCZ2i5AEzwgGof9Ag==",
		&result,
	)
	tt.Assert.NoError(err)

	operation := TransactionOperation{
		Index:          0,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 56,
	}
	err = builder.Add(operation)
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)

	ops := []Operation{}
	// Custom select for now  to make the test pass.
	// TODO: Add join on transactions
	err = q.Select(&ops, selectExpOperation)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 1)

		op := ops[0]
		tt.Assert.Equal(int64(240518172673), op.ID)
		tt.Assert.Equal(int64(240518172672), op.TransactionID)
		tt.Assert.Equal(int32(1), op.ApplicationOrder)
		tt.Assert.Equal(xdr.OperationTypePayment, op.Type)
		tt.Assert.Equal(
			"{\"to\": \"GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y\", \"from\": \"GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y\", \"amount\": \"10.0000000\", \"asset_type\": \"native\"}",
			op.DetailsString.String,
		)
		tt.Assert.Equal("GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y", op.SourceAccount)
	}
}
