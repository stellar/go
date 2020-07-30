package history

import (
	"testing"
	"time"

	"github.com/stellar/go/services/horizon/internal/db2"
	strtime "github.com/stellar/go/support/time"
	"github.com/stretchr/testify/assert"
)

func TestTradeAggregationsLimitRangeAscending(t *testing.T) {
	q := TradeAggregationsQ{
		resolution: 3600 * 1000,
		pagingParams: db2.PageQuery{
			Order: db2.OrderAscending,
			Limit: 200,
		},
	}

	query, args, err := q.GetSql().ToSql()
	assert.NoError(t, err)
	assert.Contains(t, query, "LIMIT 200")
	startTime := args[2].(time.Time)
	endTime := args[3].(time.Time)
	assert.Equal(t, int64(0), startTime.Unix()) // startTime
	// resolution * limit but in seconds
	assert.Equal(t, int64(3600*200), endTime.Unix()) // endTime
}

func TestTradeAggregationsLimitRangeDescending(t *testing.T) {
	q := TradeAggregationsQ{
		resolution: 3600 * 1000,
		pagingParams: db2.PageQuery{
			Order: db2.OrderDescending,
			Limit: 200,
		},
	}

	query, args, err := q.GetSql().ToSql()
	assert.NoError(t, err)
	assert.Contains(t, query, "LIMIT 200")
	startTime := args[2].(time.Time)
	endTime := args[3].(time.Time)
	assert.WithinDuration(t, time.Now(), endTime, time.Second) // endTime
	expectedStartTime := endTime.Unix() - int64(3600*200)
	assert.Equal(t, expectedStartTime, startTime.Unix()) // startTime
}

func TestTradeAggregationsRangeSmallerThanLimit(t *testing.T) {
	q := &TradeAggregationsQ{
		resolution: 3600 * 1000,
		pagingParams: db2.PageQuery{
			Order: db2.OrderAscending,
			Limit: 200,
		},
	}

	startTime := time.Now()
	endTime := startTime.Add(5 * time.Hour)

	var err error
	q, err = q.WithStartTime(strtime.MillisFromSeconds(startTime.Unix()))
	assert.NoError(t, err)
	q, err = q.WithEndTime(strtime.MillisFromSeconds(endTime.Unix()))
	assert.NoError(t, err)

	// Load adjusted: WithStartTime and WithEndTime round down to resolution hours
	startTime = q.startTime.ToTime()
	endTime = q.endTime.ToTime()

	query, args, err := q.GetSql().ToSql()
	assert.NoError(t, err)
	assert.Contains(t, query, "LIMIT 200")
	actualStartTime := args[2].(time.Time)
	actualEndTime := args[3].(time.Time)
	assert.Equal(t, startTime.Unix(), actualStartTime.Unix())
	assert.Equal(t, endTime.Unix(), actualEndTime.Unix())
}
