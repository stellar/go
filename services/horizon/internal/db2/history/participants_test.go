package history

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/test"
)

func assertAccountsContainAddresses(tt *test.T, accounts map[string]int64, addresses []string) {
	tt.Assert.Len(accounts, len(addresses))
	set := map[int64]bool{}
	for _, address := range addresses {
		accountID, ok := accounts[address]
		tt.Assert.True(ok)
		tt.Assert.False(set[accountID])
		set[accountID] = true
	}
}

func TestCreateExpAccounts(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	addresses := []string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	}
	accounts, err := q.CreateExpAccounts(addresses)
	tt.Assert.NoError(err)
	tt.Assert.Len(accounts, 2)
	assertAccountsContainAddresses(tt, accounts, addresses)

	dupAccounts, err := q.CreateExpAccounts([]string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	})
	tt.Assert.NoError(err)
	tt.Assert.Equal(accounts, dupAccounts)

	addresses = []string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		"GCYVFGI3SEQJGBNQQG7YCMFWEYOHK3XPVOVPA6C566PXWN4SN7LILZSM",
		"GBYSBDAJZMHL5AMD7QXQ3JEP3Q4GLKADWIJURAAHQALNAWD6Z5XF2RAC",
	}
	accounts, err = q.CreateExpAccounts(addresses)
	tt.Assert.NoError(err)
	assertAccountsContainAddresses(tt, accounts, addresses)
	for address, accountID := range dupAccounts {
		id, ok := accounts[address]
		tt.Assert.True(ok)
		tt.Assert.Equal(id, accountID)
	}
}

type transactionParticipant struct {
	TransactionID int64 `db:"history_transaction_id"`
	AccountID     int64 `db:"history_account_id"`
}

func getTransactionParticipants(tt *test.T, q *Q) []transactionParticipant {
	var participants []transactionParticipant
	sql := sq.Select("history_transaction_id", "history_account_id").
		From("exp_history_transaction_participants").
		OrderBy("(history_transaction_id, history_account_id) asc")

	err := q.Select(&participants, sql)
	if err != nil {
		tt.T.Fatal(err)
	}

	return participants
}

func TestTransactionParticipantsBatch(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	batch := q.NewTransactionParticipantsBatchInsertBuilder(0)

	transactionID := int64(1)
	otherTransactionID := int64(2)
	accountID := int64(100)

	for i := int64(0); i < 3; i++ {
		tt.Assert.NoError(batch.Add(transactionID, accountID+i))
	}

	tt.Assert.NoError(batch.Add(otherTransactionID, accountID))
	tt.Assert.NoError(batch.Exec())

	participants := getTransactionParticipants(tt, q)
	tt.Assert.Equal(
		[]transactionParticipant{
			transactionParticipant{TransactionID: 1, AccountID: 100},
			transactionParticipant{TransactionID: 1, AccountID: 101},
			transactionParticipant{TransactionID: 1, AccountID: 102},
			transactionParticipant{TransactionID: 2, AccountID: 100},
		},
		participants,
	)
}
