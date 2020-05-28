package history

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestExpStateInvalid(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	valid, lastUpdate, err := q.GetExpStateInvalid()
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.Equal(t, time.Time{}, lastUpdate)

	assert.NoError(t, q.UpdateExpStateInvalid(true))

	valid, lastUpdate, err = q.GetExpStateInvalid()
	assert.NoError(t, err)
	assert.True(t, valid)
	assert.NotEqual(t, time.Time{}, lastUpdate)

	previousTime := lastUpdate
	time.Sleep(10 * time.Millisecond)
	assert.NoError(t, q.UpdateExpStateInvalid(false))

	valid, lastUpdate, err = q.GetExpStateInvalid()
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.True(t, lastUpdate.After(previousTime))

	assert.NoError(t, q.updateValueInStore(stateInvalid, "asdfsf"))
	_, _, err = q.GetExpStateInvalid()
	assert.EqualError(t, err, "Error converting invalid value: strconv.ParseBool: parsing \"asdfsf\": invalid syntax")
}
