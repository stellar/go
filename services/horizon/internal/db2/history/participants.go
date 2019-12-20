package history

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/db"
)

// QParticipants defines ingestion participant related queries.
type QParticipants interface {
	CheckExpParticipants(seq int32) (bool, error)
	CreateExpAccounts(addresses []string) (map[string]int64, error)
	NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) TransactionParticipantsBatchInsertBuilder
	NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder
}

// CreateExpAccounts creates rows in the exp_history_accounts table for a given list of addresses.
// CreateExpAccounts returns a mapping of account address to its corresponding id in the exp_history_accounts table
func (q *Q) CreateExpAccounts(addresses []string) (map[string]int64, error) {
	var accounts []Account
	sql := sq.Insert("exp_history_accounts").Columns("address")
	for _, address := range addresses {
		sql = sql.Values(address)
	}
	sql = sql.Suffix("ON CONFLICT (address) DO UPDATE SET address=EXCLUDED.address RETURNING *")

	err := q.Select(&accounts, sql)
	if err != nil {
		return nil, err
	}

	addressToID := map[string]int64{}
	for _, account := range accounts {
		addressToID[account.Address] = account.ID
	}
	return addressToID, nil
}

type transactionParticipantPair struct {
	ID      int64  `db:"history_transaction_id"`
	Address string `db:"address"`
}

func (q *Q) findTransactionParticipants(
	participantTable, accountTable string, seq int32,
) ([]transactionParticipantPair, error) {
	var participants []transactionParticipantPair
	from := toid.ID{LedgerSequence: int32(seq)}.ToInt64()
	to := toid.ID{LedgerSequence: int32(seq + 1)}.ToInt64()

	err := q.Select(
		&participants,
		sq.Select(
			"htp.history_transaction_id",
			"ha.address",
		).From(
			fmt.Sprintf("%s htp", participantTable),
		).Join(
			fmt.Sprintf("%s ha ON ha.id = htp.history_account_id", accountTable),
		).Where(
			"htp.history_transaction_id >= ? AND htp.history_transaction_id < ? ", from, to,
		).OrderBy(
			"htp.history_transaction_id asc, ha.address asc",
		),
	)

	return participants, err
}

type ingestionCheckFn func(*Q, int32) (bool, error)

var participantChecks = []ingestionCheckFn{
	checkExpTransactionParticipants,
	checkExpOperationParticipants,
}

// CheckExpParticipants checks that the participants in the
// experimental ingestion tables matches the participants in the
// legacy ingestion tables
func (q *Q) CheckExpParticipants(seq int32) (bool, error) {
	for _, checkFn := range participantChecks {
		if valid, err := checkFn(q, seq); err != nil {
			return false, err
		} else if !valid {
			return false, nil
		}
	}
	return true, nil
}

// checkExpTransactionParticipants checks that the participants in
// exp_history_transaction_participants for the given ledger matches
// the same participants in history_transaction_participants
func checkExpTransactionParticipants(q *Q, seq int32) (bool, error) {
	participants, err := q.findTransactionParticipants(
		"history_transaction_participants", "history_accounts", seq,
	)
	if err != nil {
		return false, err
	}

	expParticipants, err := q.findTransactionParticipants(
		"exp_history_transaction_participants", "exp_history_accounts", seq,
	)
	if err != nil {
		return false, err
	}

	// We only proceed with the comparison if we have data in both the
	// legacy ingestion system and the experimental ingestion system.
	// If there are no participants in either the legacy ingestion system or the
	// experimental ingestion system we skip the check.
	if len(participants) == 0 || len(expParticipants) == 0 {
		return true, nil
	}

	if len(participants) != len(expParticipants) {
		return false, nil
	}

	for i, participant := range participants {
		expParticipant := expParticipants[i]
		if participant != expParticipant {
			return false, nil
		}
	}

	return true, nil
}

// TransactionParticipantsBatchInsertBuilder is used to insert transaction participants into the
// exp_history_transaction_participants table
type TransactionParticipantsBatchInsertBuilder interface {
	Add(transactionID, accountID int64) error
	Exec() error
}

type transactionParticipantsBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewTransactionParticipantsBatchInsertBuilder constructs a new TransactionParticipantsBatchInsertBuilder instance
func (q *Q) NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) TransactionParticipantsBatchInsertBuilder {
	return &transactionParticipantsBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("exp_history_transaction_participants"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds a new transaction participant to the batch
func (i *transactionParticipantsBatchInsertBuilder) Add(transactionID, accountID int64) error {
	return i.builder.Row(map[string]interface{}{
		"history_transaction_id": transactionID,
		"history_account_id":     accountID,
	})
}

// Exec flushes all pending transaction participants to the db
func (i *transactionParticipantsBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
