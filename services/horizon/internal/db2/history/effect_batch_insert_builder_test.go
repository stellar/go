package history

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
)

func TestAddEffect(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	address := "GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON"
	accounIDs, err := q.CreateAccounts([]string{address}, 1)
	tt.Assert.NoError(err)

	builder := q.NewEffectBatchInsertBuilder(2)
	sequence := int32(56)
	details, err := json.Marshal(map[string]string{
		"amount":     "1000.0000000",
		"asset_type": "native",
	})

	err = builder.Add(
		accounIDs[address],
		toid.New(sequence, 1, 1).ToInt64(),
		1,
		3,
		details,
	)
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)

	effects := []Effect{}
	tt.Assert.NoError(q.Effects().Select(&effects))
	tt.Assert.Len(effects, 1)

	effect := effects[0]
	tt.Assert.Equal(address, effect.Account)
	tt.Assert.Equal(int64(240518172673), effect.HistoryOperationID)
	tt.Assert.Equal(int32(1), effect.Order)
	tt.Assert.Equal(EffectType(3), effect.Type)
	tt.Assert.Equal("{\"amount\": \"1000.0000000\", \"asset_type\": \"native\"}", effect.DetailsString.String)
}
