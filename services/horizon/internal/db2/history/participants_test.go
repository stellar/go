package history

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
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

func TestCheckExpParticipants(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sequence := int32(20)

	valid, err := q.CheckExpParticipants(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	addresses := []string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		"GCYVFGI3SEQJGBNQQG7YCMFWEYOHK3XPVOVPA6C566PXWN4SN7LILZSM",
		"GBYSBDAJZMHL5AMD7QXQ3JEP3Q4GLKADWIJURAAHQALNAWD6Z5XF2RAC",
	}
	expAccounts, err := q.CreateExpAccounts(addresses)
	tt.Assert.NoError(err)

	transactionIDs := []int64{
		toid.New(sequence, 1, 0).ToInt64(),
		toid.New(sequence, 2, 0).ToInt64(),
		toid.New(sequence, 1, 0).ToInt64(),
		toid.New(sequence+1, 1, 0).ToInt64(),
	}

	batch := q.NewTransactionParticipantsBatchInsertBuilder(0)
	for i, address := range addresses {
		tt.Assert.NoError(
			batch.Add(transactionIDs[i], expAccounts[address]),
		)
	}
	tt.Assert.NoError(batch.Exec())

	valid, err = q.CheckExpParticipants(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	addresses = append(addresses, "GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON")
	transactionIDs = append(transactionIDs, toid.New(sequence, 3, 0).ToInt64())
	var accounts []Account
	tt.Assert.NoError(q.CreateAccounts(&accounts, addresses))
	accountsMap := map[string]int64{}
	for _, account := range accounts {
		accountsMap[account.Address] = account.ID
	}

	for i, address := range addresses {
		_, err := q.Exec(sq.Insert("history_transaction_participants").
			SetMap(map[string]interface{}{
				"history_transaction_id": transactionIDs[i],
				"history_account_id":     accountsMap[address],
			}))
		tt.Assert.NoError(err)

		valid, err = q.CheckExpParticipants(sequence)
		tt.Assert.NoError(err)
		// The first 3 transactions all belong to ledger `sequence`.
		// The 4th transaction belongs to the next ledger so it is
		// ignored by CheckExpParticipants.
		// The last transaction belongs to `sequence`, however, it is
		// not present in exp_history_transaction_participants so
		// we expect CheckExpParticipants to fail after the last
		// transaction is added to history_transaction_participants
		expected := i == 2 || i == 3
		tt.Assert.Equal(expected, valid)
	}
}
