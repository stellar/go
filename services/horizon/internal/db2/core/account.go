package core

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/xdr"
)

// IsAuthRequired returns true if the account has the "AUTH_REQUIRED" option
// turned on.
func (ac Account) IsAuthRequired() bool {
	return (ac.Flags & xdr.AccountFlagsAuthRequiredFlag) != 0
}

// IsAuthRevocable returns true if the account has the "AUTH_REVOCABLE" option
// turned on.
func (ac Account) IsAuthRevocable() bool {
	return (ac.Flags & xdr.AccountFlagsAuthRevocableFlag) != 0
}

// IsAuthImmutable returns true if the account has the "AUTH_IMMUTABLE" option
// turned on.
func (ac Account) IsAuthImmutable() bool {
	return (ac.Flags & xdr.AccountFlagsAuthImmutableFlag) != 0
}

// AccountByAddress loads a row from `accounts`, by address
func (q *Q) AccountByAddress(dest interface{}, addy string, protocolVersion int32) error {
	var selectQuery sq.SelectBuilder

	if protocolVersion >= 10 {
		selectQuery = selectAccount
	} else {
		selectQuery = selectAccountPreV10
	}

	sql := selectQuery.Limit(1).Where("accountid = ?", addy)

	return q.Get(dest, sql)
}

// SequencesForAddresses loads the current sequence number for every accountid
// specified in `addys`
func (q *Q) SequencesForAddresses(dest interface{}, addys []string) error {
	sql := sq.
		Select("seqnum as sequence", "accountid as address").
		From("accounts").
		Where(sq.Eq{"accountid": addys})

	return q.Select(dest, sql)
}

// SequenceProvider returns a new sequence provider.
func (q *Q) SequenceProvider() *SequenceProvider {
	return &SequenceProvider{Q: q}
}

// Get implements `txsub.SequenceProvider`
func (sp *SequenceProvider) Get(addys []string) (map[string]uint64, error) {
	rows := []struct {
		Address  string
		Sequence uint64
	}{}

	err := sp.Q.SequencesForAddresses(&rows, addys)
	if err != nil {
		return nil, err
	}

	results := make(map[string]uint64)
	for _, r := range rows {
		results[r.Address] = r.Sequence
	}
	return results, nil
}

var selectAccount = sq.Select(
	"a.accountid",
	"a.balance",
	"a.seqnum",
	"a.numsubentries",
	"a.inflationdest",
	"a.homedomain",
	"a.thresholds",
	"a.flags",
	// Liabilities can be NULL so can error without `coalesce`:
	// `Invalid value for xdr.Int64`
	"coalesce(a.buyingliabilities, 0) as buyingliabilities",
	"coalesce(a.sellingliabilities, 0) as sellingliabilities",
).From("accounts a")

var selectAccountPreV10 = sq.Select(
	"a.accountid",
	"a.balance",
	"a.seqnum",
	"a.numsubentries",
	"a.inflationdest",
	"a.homedomain",
	"a.thresholds",
	"a.flags",
).From("accounts a")
