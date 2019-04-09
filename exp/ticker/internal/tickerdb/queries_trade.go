package tickerdb

import (
	"fmt"
	"strings"
)

// BulkInsertTrades inserts a slice of trades in the database. Trades
// that are already in the database (i.e. horizon_id already exists)
// are ignored.
func (s *TickerSession) BulkInsertTrades(trades []Trade) (err error) {
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

	fmt.Println(qs)
	fmt.Println(dbValues)

	_, err = s.ExecRaw(qs, dbValues...)
	return
}
