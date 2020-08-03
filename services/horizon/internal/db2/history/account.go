package history

import (
	"sort"

	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// AccountByAddress loads a row from `history_accounts`, by address
func (q *Q) AccountByAddress(dest interface{}, addy string) error {
	sql := selectAccount.Limit(1).Where("ha.address = ?", addy)
	return q.Get(dest, sql)
}

// AccountsByAddresses loads a rows from `history_accounts`, by addresses
func (q *Q) AccountsByAddresses(dest interface{}, addresses []string) error {
	sql := selectAccount.Where(map[string]interface{}{
		"ha.address": addresses, // ha.address IN (...)
	})
	return q.Select(dest, sql)
}

// CreateAccounts creates rows in the history_accounts table for a given list of addresses.
// CreateAccounts returns a mapping of account address to its corresponding id in the history_accounts table
func (q *Q) CreateAccounts(addresses []string, batchSize int) (map[string]int64, error) {
	builder := &db.BatchInsertBuilder{
		Table:        q.GetTable("history_accounts"),
		MaxBatchSize: batchSize,
		Suffix:       "ON CONFLICT (address) DO NOTHING",
	}

	// sort assets before inserting rows into history_assets to prevent deadlocks on acquiring a ShareLock
	// https://github.com/stellar/go/issues/2370
	sort.Strings(addresses)
	for _, address := range addresses {
		err := builder.Row(map[string]interface{}{
			"address": address,
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not insert history_accounts row")
		}
	}

	err := builder.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "could not exec asset insert builder")
	}

	var accounts []Account
	addressToID := map[string]int64{}
	const selectBatchSize = 10000

	for i := 0; i < len(addresses); i += selectBatchSize {
		end := i + selectBatchSize
		if end > len(addresses) {
			end = len(addresses)
		}
		subset := addresses[i:end]

		if err := q.AccountsByAddresses(&accounts, subset); err != nil {
			return nil, errors.Wrap(err, "could not select accounts")
		}

		for _, account := range accounts {
			addressToID[account.Address] = account.ID
		}
	}

	return addressToID, nil
}

var selectAccount = sq.Select("ha.*").From("history_accounts ha")
