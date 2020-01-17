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
	err := builder.Add(240518172673, 1)
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
		From("history_operation_participants hopp"),
	)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 1)

		op := ops[0]
		tt.Assert.Equal(int64(240518172673), op.OperationID)
		tt.Assert.Equal(int64(1), op.AccountID)
	}
}
