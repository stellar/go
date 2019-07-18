package history

import (
	"database/sql"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
)

const (
	lastLedgerKey = "exp_ingest_last_ledger"
)

// GetLastLedgerExpIngest returns the last ledger ingested by expingest system
// in Horizon. Returns 0 if no value has been previously set. This can be set
// using UpdateLastLedgerExpIngest.
func (q *Q) GetLastLedgerExpIngest() (uint32, error) {
	lastIngestedLedger, err := q.getValueFromStore(lastLedgerKey)
	if err != nil {
		return 0, err
	}

	if lastIngestedLedger == "" {
		return 0, nil
	} else {
		ledgerSequence, err := strconv.ParseUint(lastIngestedLedger, 10, 32)
		if err != nil {
			return 0, errors.Wrap(err, "Error converting lastIngestedLedger value")
		}

		return uint32(ledgerSequence), nil
	}
}

// UpdateLastLedgerExpIngest upsets the last ledger ingested by expingest system.
// Can be read using GetLastLedgerExpIngest.
func (q *Q) UpdateLastLedgerExpIngest(ledgerSequence uint32) error {
	return q.updateValueInStore(
		lastLedgerKey,
		strconv.FormatUint(uint64(ledgerSequence), 10),
	)
}

// getValueFromStore returns a value for a given key from KV store
func (q *Q) getValueFromStore(key string) (string, error) {
	query := sq.Select("key_value_store.value").
		From("key_value_store").
		Where("key_value_store.key = ?", key)

	var value string
	if err := q.Get(&value, query); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "could not get value")
	}

	return value, nil
}

// updateValueInStore updates a value for a given key in KV store
func (q *Q) updateValueInStore(key, value string) error {
	query := sq.Insert("key_value_store").
		Columns("key", "value").
		Values(key, value).
		Suffix("ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value")

	_, err := q.Exec(query)
	return err
}
