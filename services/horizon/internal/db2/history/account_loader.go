package history

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sort"
	"strings"

	"github.com/lib/pq"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// FutureAccountID represents a future history account.
// A FutureAccountID is created by an AccountLoader and
// the account id is available after calling Exec() on
// the AccountLoader.
type FutureAccountID struct {
	address string
	loader  *AccountLoader
}

// Value implements the database/sql/driver Valuer interface.
func (a FutureAccountID) Value() (driver.Value, error) {
	return a.loader.GetNow(a.address)
}

// AccountLoader will map account addresses to their history
// account ids. If there is no existing mapping for a given address,
// the AccountLoader will insert into the history_accounts table to
// establish a mapping.
type AccountLoader struct {
	sealed bool
	set    set.Set[string]
	ids    map[string]int64
	stats  LoaderStats
}

var errSealed = errors.New("cannot register more entries to loader after calling Exec()")

// NewAccountLoader will construct a new AccountLoader instance.
func NewAccountLoader() *AccountLoader {
	return &AccountLoader{
		sealed: false,
		set:    set.Set[string]{},
		ids:    map[string]int64{},
		stats:  LoaderStats{},
	}
}

// GetFuture registers the given account address into the loader and
// returns a FutureAccountID which will hold the history account id for
// the address after Exec() is called.
func (a *AccountLoader) GetFuture(address string) FutureAccountID {
	if a.sealed {
		panic(errSealed)
	}

	a.set.Add(address)
	return FutureAccountID{
		address: address,
		loader:  a,
	}
}

// GetNow returns the history account id for the given address.
// GetNow should only be called on values which were registered by
// GetFuture() calls. Also, Exec() must be called before any GetNow
// call can succeed.
func (a *AccountLoader) GetNow(address string) (int64, error) {
	if !a.sealed {
		return 0, fmt.Errorf(`invalid account loader state,  
		Exec was not called yet to properly seal and resolve %v id`, address)
	}
	if internalID, ok := a.ids[address]; !ok {
		return 0, fmt.Errorf(`account loader address %q was not found`, address)
	} else {
		return internalID, nil
	}
}

// LoaderStats describes the result of executing a history lookup id loader
type LoaderStats struct {
	// Total is the number of elements registered to the loader
	Total int
	// Inserted is the number of elements inserted into the lookup table
	Inserted int
}

// Exec will look up all the history account ids for the addresses registered in the loader.
// If there are no history account ids for a given set of addresses, Exec will insert rows
// into the history_accounts table to establish a mapping between address and history account id.
func (a *AccountLoader) Exec(ctx context.Context, session db.SessionInterface) error {
	a.sealed = true
	if len(a.set) == 0 {
		return nil
	}
	q := &Q{session}
	addresses := make([]string, 0, len(a.set))
	for address := range a.set {
		addresses = append(addresses, address)
	}
	// sort entries before inserting rows to prevent deadlocks on acquiring a ShareLock
	// https://github.com/stellar/go/issues/2370
	sort.Strings(addresses)

	var accounts []Account
	err := bulkInsert(
		ctx,
		q,
		"history_accounts",
		[]columnValues{
			{
				name:    "address",
				dbType:  "character varying(64)",
				objects: addresses,
			},
		},
		&accounts,
	)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		a.ids[account.Address] = account.ID
		a.stats.Inserted++
	}
	a.stats.Total += len(accounts)

	remaining := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if _, ok := a.ids[address]; ok {
			continue
		}
		remaining = append(remaining, address)
	}
	if len(remaining) > 0 {
		var remainingAccounts []Account
		err = bulkGet(
			ctx,
			q,
			"history_accounts",
			[]columnValues{
				{
					name:    "address",
					dbType:  "character varying(64)",
					objects: remaining,
				},
			},
			&remainingAccounts,
		)
		if err != nil {
			return err
		}
		for _, account := range remainingAccounts {
			a.ids[account.Address] = account.ID
		}
		a.stats.Total += len(remainingAccounts)
	}

	return nil
}

// Stats returns the number of addresses registered in the loader and the number of addresses
// inserted into the history_accounts table.
func (a *AccountLoader) Stats() LoaderStats {
	return a.stats
}

func (a *AccountLoader) Name() string {
	return "AccountLoader"
}

type columnValues struct {
	name    string
	dbType  string
	objects []string
}

func bulkInsert(ctx context.Context, q *Q, table string, fields []columnValues, response interface{}) error {
	unnestPart := make([]string, 0, len(fields))
	insertFieldsPart := make([]string, 0, len(fields))
	pqArrays := make([]interface{}, 0, len(fields))

	// In the code below we are building the bulk insert query which looks like:
	//
	// WITH rows AS
	//		(SELECT
	//			/* unnestPart */
	//			unnest(?::type1[]), /* field1 */
	//			unnest(?::type2[]), /* field2 */
	//			...
	//		)
	//	INSERT INTO table (
	//		/* insertFieldsPart */
	//		field1,
	//		field2,
	//		...
	//	)
	//	SELECT * FROM rows ON CONFLICT (field1, field2, ...) DO NOTHING RETURNING *
	//
	// Using unnest allows to get around the maximum limit of 65,535 query parameters,
	// see https://www.postgresql.org/docs/12/limits.html and
	// https://klotzandrew.com/blog/postgres-passing-65535-parameter-limit/
	//
	// Without using unnest we would have to use multiple insert statements to insert
	// all the rows for large datasets.
	for _, field := range fields {
		unnestPart = append(
			unnestPart,
			fmt.Sprintf("unnest(?::%s[]) /* %s */", field.dbType, field.name),
		)
		insertFieldsPart = append(
			insertFieldsPart,
			field.name,
		)
		pqArrays = append(
			pqArrays,
			pq.Array(field.objects),
		)
	}
	columns := strings.Join(insertFieldsPart, ",")

	sql := `
	WITH rows AS
		(SELECT ` + strings.Join(unnestPart, ",") + `)
	INSERT INTO ` + table + `
		(` + columns + `)
	SELECT * FROM rows
	ON CONFLICT (` + columns + `) DO NOTHING
	RETURNING *`

	return q.SelectRaw(
		ctx,
		response,
		sql,
		pqArrays...,
	)
}

func bulkGet(ctx context.Context, q *Q, table string, fields []columnValues, response interface{}) error {
	unnestPart := make([]string, 0, len(fields))
	columns := make([]string, 0, len(fields))
	pqArrays := make([]interface{}, 0, len(fields))

	// In the code below we are building the bulk get query which looks like:
	//
	//	SELECT * FROM table WHERE (field1, field2, ...) IN
	//		(SELECT
	//			/* unnestPart */
	//			unnest(?::type1[]), /* field1 */
	//			unnest(?::type2[]), /* field2 */
	//			...
	//		)
	//
	// Using unnest allows to get around the maximum limit of 65,535 query parameters,
	// see https://www.postgresql.org/docs/12/limits.html and
	// https://klotzandrew.com/blog/postgres-passing-65535-parameter-limit/
	//
	// Without using unnest we would have to use multiple select statements to obtain
	// all the rows for large datasets.
	for _, field := range fields {
		unnestPart = append(
			unnestPart,
			fmt.Sprintf("unnest(?::%s[]) /* %s */", field.dbType, field.name),
		)
		columns = append(
			columns,
			field.name,
		)
		pqArrays = append(
			pqArrays,
			pq.Array(field.objects),
		)
	}
	sql := `SELECT * FROM ` + table + ` WHERE (` + strings.Join(columns, ",") + `) IN 
	(SELECT ` + strings.Join(unnestPart, ",") + `)`

	return q.SelectRaw(
		ctx,
		response,
		sql,
		pqArrays...,
	)
}

// AccountLoaderStub is a stub wrapper around AccountLoader which allows
// you to manually configure the mapping of addresses to history account ids
type AccountLoaderStub struct {
	Loader *AccountLoader
}

// NewAccountLoaderStub returns a new AccountLoaderStub instance
func NewAccountLoaderStub() AccountLoaderStub {
	return AccountLoaderStub{Loader: NewAccountLoader()}
}

// Insert updates the wrapped AccountLoader so that the given account
// address is mapped to the provided history account id
func (a AccountLoaderStub) Insert(address string, id int64) {
	a.Loader.sealed = true
	a.Loader.ids[address] = id
}
