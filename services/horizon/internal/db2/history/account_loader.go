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
	"github.com/stellar/go/support/ordered"
)

const loaderLookupBatchSize = 50000

var errSealed = errors.New("cannot register more entries to loader after calling Exec()")

type loader[K comparable, V any] struct {
	sealed         bool
	set            set.Set[K]
	ids            map[K]int64
	sort           func(keys []K)
	fetchAndUpdate func(ctx context.Context, q *Q, keys []K) error
	insert         func(ctx context.Context, q *Q, keys []K) error
	newFuture      func(key K) V
}

// GetFuture registers the given key into the loader and
// returns a future which will hold the history id for
// the key after Exec() is called.
func (l *loader[K, V]) GetFuture(key K) V {
	if l.sealed {
		panic(errSealed)
	}

	l.set.Add(key)
	return l.newFuture(key)
}

// GetNow returns the history id for the given key.
// GetNow should only be called on values which were registered by
// GetFuture() calls. Also, Exec() must be called before any GetNow
// call can succeed.
func (l *loader[K, V]) GetNow(key K) int64 {
	if id, ok := l.ids[key]; !ok {
		panic(fmt.Errorf("key %v not present", key))
	} else {
		return id
	}
}

func (l *loader[K, V]) lookupKeys(ctx context.Context, q *Q, keys []K) error {
	for i := 0; i < len(keys); i += loaderLookupBatchSize {
		end := ordered.Min(len(keys), i+loaderLookupBatchSize)

		if err := l.fetchAndUpdate(ctx, q, keys[i:end]); err != nil {
			return err
		}
	}
	return nil
}

// Exec will look up all the history ids for the keys registered in the loader.
// If there are no history ids for a given set of keys, Exec will insert rows
// into the history table.
func (l *loader[K, V]) Exec(ctx context.Context, session db.SessionInterface) error {
	l.sealed = true
	if len(l.set) == 0 {
		return nil
	}
	q := &Q{session}
	keys := make([]K, 0, len(l.set))
	for address := range l.set {
		keys = append(keys, address)
	}
	// sort entries before inserting rows to prevent deadlocks on acquiring a ShareLock
	// https://github.com/stellar/go/issues/2370
	l.sort(keys)

	if err := l.lookupKeys(ctx, q, keys); err != nil {
		return err
	}

	insert := 0
	for _, address := range keys {
		if _, ok := l.ids[address]; ok {
			continue
		}
		keys[insert] = address
		insert++
	}
	if insert == 0 {
		return nil
	}
	keys = keys[:insert]

	err := l.insert(ctx, q, keys)
	if err != nil {
		return err
	}

	return l.lookupKeys(ctx, q, keys)
}

type bulkInsertField struct {
	name    string
	dbType  string
	objects []string
}

func bulkInsert(ctx context.Context, q *Q, table string, conflictFields []string, fields []bulkInsertField) error {
	unnestPart := make([]string, 0, len(fields))
	insertFieldsPart := make([]string, 0, len(fields))
	pqArrays := make([]interface{}, 0, len(fields))

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

	sql := `
	WITH r AS
		(SELECT ` + strings.Join(unnestPart, ",") + `)
	INSERT INTO ` + table + `
		(` + strings.Join(insertFieldsPart, ",") + `)
	SELECT * from r
	ON CONFLICT (` + strings.Join(conflictFields, ",") + `) DO NOTHING`

	_, err := q.ExecRaw(
		context.WithValue(ctx, &db.QueryTypeContextKey, db.UpsertQueryType),
		sql,
		pqArrays...,
	)
	return err
}

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
	return a.loader.GetNow(a.address), nil
}

// AccountLoader will map account addresses to their history
// account ids. If there is no existing mapping for a given address,
// the AccountLoader will insert into the history_accounts table to
// establish a mapping.
type AccountLoader struct {
	loader[string, FutureAccountID]
}

// NewAccountLoader will construct a new AccountLoader instance.
func NewAccountLoader() *AccountLoader {
	l := &AccountLoader{
		loader: loader[string, FutureAccountID]{
			sealed: false,
			set:    set.Set[string]{},
			ids:    map[string]int64{},
			sort:   sort.Strings,
			insert: func(ctx context.Context, q *Q, keys []string) error {
				return bulkInsert(
					ctx,
					q,
					"history_accounts",
					[]string{"address"},
					[]bulkInsertField{
						{
							name:    "address",
							dbType:  "character varying(64)",
							objects: keys,
						},
					},
				)
			},
		},
	}
	l.fetchAndUpdate = func(ctx context.Context, q *Q, keys []string) error {
		var accounts []Account
		if err := q.AccountsByAddresses(ctx, &accounts, keys); err != nil {
			return errors.Wrap(err, "could not select accounts")
		}

		for _, account := range accounts {
			l.ids[account.Address] = account.ID
		}
		return nil
	}
	l.newFuture = func(key string) FutureAccountID {
		return FutureAccountID{
			address: key,
			loader:  l,
		}
	}

	return l
}
