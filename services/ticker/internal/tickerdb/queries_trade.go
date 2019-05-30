package tickerdb

import (
	"math"
	"strings"
	"time"
)

// BulkInsertTrades inserts a slice of trades in the database. Trades
// that are already in the database (i.e. horizon_id already exists)
// are ignored.
func (s *TickerSession) BulkInsertTrades(trades []Trade) (err error) {
	if len(trades) <= 50 {
		return performInsertTrades(s, trades)
	}

	chunks := chunkifyDBTrades(trades, 50)
	for _, chunk := range chunks {
		err = performInsertTrades(s, chunk)
		if err != nil {
			return
		}
	}

	return
}

// GetLastTrade returns the newest Trade object in the database.
func (s *TickerSession) GetLastTrade() (trade Trade, err error) {
	err = s.GetRaw(&trade, "SELECT * FROM trades ORDER BY ledger_close_time DESC LIMIT 1")
	return
}

// DeleteOldTrades deletes trades in the database older than minDate.
func (s *TickerSession) DeleteOldTrades(minDate time.Time) error {
	_, err := s.ExecRaw("DELETE FROM trades WHERE ledger_close_time < ?", minDate)
	return err
}

// chunkifyDBTrades transforms a slice into a slice of chunks (also slices) of chunkSize
// e.g.: Chunkify([b, c, d, e, f], 2) = [[b c] [d e] [f]]
func chunkifyDBTrades(sl []Trade, chunkSize int) [][]Trade {
	var chunkedSlice [][]Trade

	numChunks := int(math.Ceil(float64(len(sl)) / float64(chunkSize)))
	start := 0
	length := len(sl)

	for i := 0; i < numChunks; i++ {
		end := start + chunkSize

		if end > length {
			end = length
		}
		chunk := sl[start:end]
		chunkedSlice = append(chunkedSlice, chunk)
		start = end

	}

	return chunkedSlice
}

func performInsertTrades(s *TickerSession, trades []Trade) (err error) {
	var t Trade
	var placeholders string
	var dbValues []interface{}

	dbFields := getDBFieldTags(t, true)
	dbFieldsString := strings.Join(dbFields, ", ")

	for i, trade := range trades {
		v := getDBFieldValues(trade, true)
		placeholders += "(" + generatePlaceholders(v) + ")"
		dbValues = append(dbValues, v...)

		if i != len(trades)-1 {
			placeholders += ","
		}
	}

	qs := "INSERT INTO trades (" + dbFieldsString + ")"
	qs += " VALUES " + placeholders
	qs += " ON CONFLICT ON CONSTRAINT trades_horizon_id_key DO NOTHING;"

	_, err = s.ExecRaw(qs, dbValues...)
	return
}
