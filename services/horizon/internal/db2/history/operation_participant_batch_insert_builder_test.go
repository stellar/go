package history

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
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
		From("exp_history_operation_participants hopp"),
	)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 1)

		op := ops[0]
		tt.Assert.Equal(int64(240518172673), op.OperationID)
		tt.Assert.Equal(int64(1), op.AccountID)
	}
}

func TestOperationParticipants(t *testing.T) {
	tt := assert.New(t)

	sequence := uint32(56)
	transaction := buildLedgerTransaction(
		t,
		testTransaction{
			index:         1,
			envelopeXDR:   "AAAAAONt/6wGI884Zi6sYDYC1GOV/drnh4OcRrTrqJPoOTUKAAAAZAAAABAAAAADAAAAAAAAAAAAAAABAAAAAAAAAAIAAAAAAAAAADuaygAAAAAABAjoBMEUiZNLUjsWXL1iK59D90Li4w56076b8HKxZfIAAAABRVVSAAAAAAAuwvNzNk9twbuJHUBqnX26GYI3MbCdpQU9t4n6EVRXsQAAAAA7msoAAAAAAAAAAAAAAAAB6Dk1CgAAAEB+7jxesBKKrF343onyycjp2tiQLZiGH2ETl+9fuOqotveY2rIgvt9ng+QJ2aDP3+PnDsYEa9ZUaA+Zne2nIGgE",
			resultXDR:     "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAEAAAAALsLzczZPbcG7iR1Aap19uhmCNzGwnaUFPbeJ+hFUV7EAAAAAAAAAAgAAAAFFVVIAAAAAAC7C83M2T23Bu4kdQGqdfboZgjcxsJ2lBT23ifoRVFexAAAAADuaygAAAAAAAAAAADuaygAAAAAABAjoBMEUiZNLUjsWXL1iK59D90Li4w56076b8HKxZfIAAAABRVVSAAAAAAAuwvNzNk9twbuJHUBqnX26GYI3MbCdpQU9t4n6EVRXsQAAAAA7msoAAAAAAA==",
			metaXDR:       "AAAAAQAAAAIAAAADAAAAFAAAAAAAAAAA423/rAYjzzhmLqxgNgLUY5X92ueHg5xGtOuok+g5NQoAAAACVAvi1AAAABAAAAACAAAAAQAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAFAAAAAAAAAAA423/rAYjzzhmLqxgNgLUY5X92ueHg5xGtOuok+g5NQoAAAACVAvi1AAAABAAAAADAAAAAQAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAACAAAAAMAAAATAAAAAQAAAAAECOgEwRSJk0tSOxZcvWIrn0P3QuLjDnrTvpvwcrFl8gAAAAFFVVIAAAAAAC7C83M2T23Bu4kdQGqdfboZgjcxsJ2lBT23ifoRVFexAAAAAHc1lAB//////////wAAAAEAAAAAAAAAAAAAAAEAAAAUAAAAAQAAAAAECOgEwRSJk0tSOxZcvWIrn0P3QuLjDnrTvpvwcrFl8gAAAAFFVVIAAAAAAC7C83M2T23Bu4kdQGqdfboZgjcxsJ2lBT23ifoRVFexAAAAALLQXgB//////////wAAAAEAAAAAAAAAAAAAAAMAAAATAAAAAgAAAAAuwvNzNk9twbuJHUBqnX26GYI3MbCdpQU9t4n6EVRXsQAAAAAAAAACAAAAAUVVUgAAAAAALsLzczZPbcG7iR1Aap19uhmCNzGwnaUFPbeJ+hFUV7EAAAAAAAAAADuaygAAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAIAAAACAAAAAC7C83M2T23Bu4kdQGqdfboZgjcxsJ2lBT23ifoRVFexAAAAAAAAAAIAAAADAAAAFAAAAAAAAAAA423/rAYjzzhmLqxgNgLUY5X92ueHg5xGtOuok+g5NQoAAAACVAvi1AAAABAAAAADAAAAAQAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAFAAAAAAAAAAA423/rAYjzzhmLqxgNgLUY5X92ueHg5xGtOuok+g5NQoAAAACGHEY1AAAABAAAAADAAAAAQAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAADAAAAEwAAAAAAAAAALsLzczZPbcG7iR1Aap19uhmCNzGwnaUFPbeJ+hFUV7EAAAACVAvi1AAAABAAAAADAAAAAgAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAQAAAAA7msoAAAAAAHc1lAAAAAAAAAAAAAAAAAEAAAAUAAAAAAAAAAAuwvNzNk9twbuJHUBqnX26GYI3MbCdpQU9t4n6EVRXsQAAAAKPpqzUAAAAEAAAAAMAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAABAAAAAAAAAAAAAAAAdzWUAAAAAAAAAAAA",
			feeChangesXDR: "AAAAAgAAAAMAAAATAAAAAAAAAADjbf+sBiPPOGYurGA2AtRjlf3a54eDnEa066iT6Dk1CgAAAAJUC+M4AAAAEAAAAAIAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAUAAAAAAAAAADjbf+sBiPPOGYurGA2AtRjlf3a54eDnEa066iT6Dk1CgAAAAJUC+LUAAAAEAAAAAIAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			hash:          "96415ac1d2f79621b26b1568f963fd8dd6c50c20a22c7428cefbfe9dee867588",
		},
	)

	participants, err := OperationsParticipants(transaction, sequence)
	tt.NoError(err)
	tt.Len(participants, 1)

	expected := []xdr.AccountId{
		xdr.MustAddress("GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD"),
		xdr.MustAddress("GACAR2AEYEKITE2LKI5RMXF5MIVZ6Q7XILROGDT22O7JX4DSWFS7FDDP"),
	}

	result := map[string]xdr.AccountId{}
	for _, addresses := range participants {
		for _, account := range addresses {
			result[account.Address()] = account
		}
	}
	for _, account := range expected {
		delete(result, account.Address())
	}

	tt.Empty(result)
}
