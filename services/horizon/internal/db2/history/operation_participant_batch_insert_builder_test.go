package history

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestAddOperationParticipants(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	builder := q.NewOperationParticipantBatchInsertBuilder(1)
	transaction := buildLedgerTransaction(
		t,
		testTransaction{
			index:         1,
			envelopeXDR:   "AAAAAGk/nUZSIwC34ltdN0iqxq+m+0UAWAilH1lZDMM07nODAAAAZAALscMAAAABAAAAAQAAAAAAAAAAAAAAAF35J+sAAAABAAAACjEyMDgwNDc1NDIAAAAAAAEAAAAAAAAAAQAAAABbpsBvIu34ZyHMCzALP5ZzWU604GJX6h9tyk49T5gDwAAAAAAAAAACVAvkAAAAAAAAAAABNO5zgwAAAEDktE4HWENBP01or+tVTmLlDM5J4rwvt0qUZ0wB6fZKbevr+j8Y2eem0lQPjAqk/jdL/KkpFantFd+NKgK+48YO",
			resultXDR:     "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
			metaXDR:       "AAAAAQAAAAIAAAADAAuxxAAAAAAAAAAAaT+dRlIjALfiW103SKrGr6b7RQBYCKUfWVkMwzTuc4MAAAAXSHbnnAALscMAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAuxxAAAAAAAAAAAaT+dRlIjALfiW103SKrGr6b7RQBYCKUfWVkMwzTuc4MAAAAXSHbnnAALscMAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAABAAAAAMAC7GlAAAAAAAAAABbpsBvIu34ZyHMCzALP5ZzWU604GJX6h9tyk49T5gDwAAAAAAX14OcAABp9wAAT4AAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAC7HEAAAAAAAAAABbpsBvIu34ZyHMCzALP5ZzWU604GJX6h9tyk49T5gDwAAAAAJr42ecAABp9wAAT4AAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMAC7HEAAAAAAAAAABpP51GUiMAt+JbXTdIqsavpvtFAFgIpR9ZWQzDNO5zgwAAABdIduecAAuxwwAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAC7HEAAAAAAAAAABpP51GUiMAt+JbXTdIqsavpvtFAFgIpR9ZWQzDNO5zgwAAABT0awOcAAuxwwAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			feeChangesXDR: "AAAAAgAAAAMAC7HDAAAAAAAAAABpP51GUiMAt+JbXTdIqsavpvtFAFgIpR9ZWQzDNO5zgwAAABdIdugAAAuxwwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAC7HEAAAAAAAAAABpP51GUiMAt+JbXTdIqsavpvtFAFgIpR9ZWQzDNO5zgwAAABdIduecAAuxwwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			hash:          "152dafa4699b954272d9896939d2ecd99e39a809713acb99680380e7f0074f89",
		},
	)

	sequence := int32(56)

	err := builder.Add(transaction, uint32(sequence))
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)

	type hop struct {
		OperationID int64 `db:"history_operation_id"`
		AccountID   int64 `db:"history_account_id"`
	}

	ops := []hop{}
	err = q.Select(&ops, sq.Select(
		"hopp.history_operation_id, "+
			"hopp.history_account_id").
		From("exp_history_operation_participants hopp"),
	)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 2)

		op := ops[0]
		tt.Assert.Equal(int64(240518172673), op.OperationID)
		tt.Assert.Equal(int64(1), op.AccountID)

		op = ops[1]
		tt.Assert.Equal(int64(240518172673), op.OperationID)
		tt.Assert.Equal(int64(2), op.AccountID)
	}
}
