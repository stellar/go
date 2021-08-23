package history

import (
	"encoding/json"
	"testing"

	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
)

func TestEffectsForLiquidityPool(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	// Insert Effect
	address := "GAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSTVY"
	muxedAddres := "MAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSAAAAAAAAAAE2LP26"
	accounIDs, err := q.CreateAccounts(tt.Ctx, []string{address}, 1)
	tt.Assert.NoError(err)

	builder := q.NewEffectBatchInsertBuilder(2)
	sequence := int32(56)
	details, err := json.Marshal(map[string]string{
		"amount":     "1000.0000000",
		"asset_type": "native",
	})
	opID := toid.New(sequence, 1, 1).ToInt64()
	err = builder.Add(tt.Ctx,
		accounIDs[address],
		null.StringFrom(muxedAddres),
		opID,
		1,
		3,
		details,
	)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)
	tt.Assert.NoError(err)

	// Insert Liquidity Pool history
	liquidityPoolID := "abcde"
	toInternalID, err := q.CreateHistoryLiquidityPools(tt.Ctx, []string{liquidityPoolID}, 2)
	tt.Assert.NoError(err)
	operationBuilder := q.NewOperationLiquidityPoolBatchInsertBuilder(2)
	tt.Assert.NoError(err)
	internalID, ok := toInternalID[liquidityPoolID]
	tt.Assert.True(ok)
	err = operationBuilder.Add(tt.Ctx, opID, internalID)
	tt.Assert.NoError(err)
	err = operationBuilder.Exec(tt.Ctx)
	tt.Assert.NoError(err)

	var result []Effect
	err = q.Effects().ForLiquidityPool(liquidityPoolID).Select(tt.Ctx, &result)
	tt.Assert.NoError(err)

	tt.Assert.Len(result, 1)
	tt.Assert.Equal(result[0].Account, address)

}
