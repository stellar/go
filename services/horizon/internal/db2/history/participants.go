package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/db"
)

// QParticipants defines ingestion participant related queries.
type QParticipants interface {
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
