package history

import (
	"encoding/json"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
)

func TestGetRemovedOffers(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	address := "GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON"
	accounIDs, err := q.CreateAccounts([]string{address}, 1)
	tt.Assert.NoError(err)

	builder := q.NewEffectBatchInsertBuilder(2)
	sequence := int32(56)
	details, err := json.Marshal(map[string]interface{}{
		"offer_id": xdr.Int64(12345),
	})
	tt.Assert.NoError(err)

	otherSequence := int32(100)
	otherDetails, err := json.Marshal(map[string]interface{}{
		"offer_id": xdr.Int64(3456),
	})
	tt.Assert.NoError(err)

	err = builder.Add(
		accounIDs[address],
		toid.New(sequence, 1, 1).ToInt64(),
		1,
		EffectOfferRemoved,
		details,
	)
	tt.Assert.NoError(err)

	err = builder.Add(
		accounIDs[address],
		toid.New(otherSequence, 1, 1).ToInt64(),
		1,
		EffectOfferRemoved,
		otherDetails,
	)
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)

	removedOffers, err := q.GetRemovedOffers(uint32(sequence) - 1)
	tt.Assert.NoError(err)
	sort.Slice(removedOffers, func(i, j int) bool {
		return removedOffers[i] < removedOffers[j]
	})
	assert.Equal(t, []xdr.Int64{3456, 12345}, removedOffers)

	removedOffers, err = q.GetRemovedOffers(uint32(sequence))
	tt.Assert.NoError(err)
	assert.Equal(t, []xdr.Int64{3456}, removedOffers)

	removedOffers, err = q.GetRemovedOffers(uint32(otherSequence))
	tt.Assert.NoError(err)
	assert.Len(t, removedOffers, 0)
}
