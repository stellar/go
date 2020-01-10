package history

import (
	"testing"

	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/db"
)

func TestCheckExpOperationEffects(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sequence := int32(57)

	valid, err := q.CheckExpOperationEffects(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	effects := []Effect{
		Effect{
			Account:            "GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
			HistoryOperationID: toid.New(sequence, 1, 1).ToInt64(),
			DetailsString: null.StringFrom(
				"{\"starting_balance\":\"1000.0000000\"}",
			),
			Type:  EffectAccountCreated,
			Order: int32(1),
		},
		Effect{
			Account:            "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			HistoryOperationID: toid.New(sequence, 1, 1).ToInt64(),
			DetailsString: null.StringFrom(
				"{\"amount\":\"1000.0000000\",\"asset_type\":\"native\"}",
			),
			Type:  EffectAccountDebited,
			Order: int32(2),
		},
		Effect{
			Account:            "GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
			HistoryOperationID: toid.New(sequence, 1, 1).ToInt64(),
			DetailsString: null.StringFrom(
				"{\"public_key\": \"GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN\", \"weight\": 1}",
			),
			Type:  EffectSignerCreated,
			Order: int32(3),
		},
		Effect{
			Account: "GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y",
			DetailsString: null.StringFrom(
				"{\"amount\":\"10.0000000\",\"asset_type\":\"native\"}",
			),
			Type:               EffectAccountCredited,
			HistoryOperationID: toid.New(sequence, 2, 1).ToInt64(),
			Order:              int32(1),
		},
		Effect{
			Account: "GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y",
			DetailsString: null.StringFrom(
				"{\"amount\":\"10.0000000\",\"asset_type\": \"native\"}",
			),
			Type:               EffectAccountDebited,
			HistoryOperationID: toid.New(sequence, 2, 1).ToInt64(),
			Order:              int32(2),
		},
		Effect{
			Account:            "GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
			Type:               EffectSequenceBumped,
			HistoryOperationID: toid.New(sequence+1, 1, 1).ToInt64(),
			Order:              int32(1),
			DetailsString: null.StringFrom(
				"{\"new_seq\": 300000000000}",
			),
		},
	}

	addresses := []string{
		"GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		"GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
	}
	expAccounts, err := q.CreateExpAccounts(addresses)
	tt.Assert.NoError(err)

	for i, effect := range effects {
		effect.HistoryAccountID = expAccounts[effect.Account]
		effects[i] = effect
	}

	batch := q.NewEffectBatchInsertBuilder(0)
	for _, effect := range effects {
		tt.Assert.NoError(
			batch.Add(
				effect.HistoryAccountID,
				effect.HistoryOperationID,
				uint32(effect.Order),
				effect.Type,
				[]byte(effect.DetailsString.String),
			),
		)
	}
	tt.Assert.NoError(batch.Exec())

	valid, err = q.CheckExpOperationEffects(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	// addresses = append(addresses, "GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON")
	// transactionIDs = append(transactionIDs, toid.New(sequence, 3, 0).ToInt64())
	var accounts []Account
	tt.Assert.NoError(q.CreateAccounts(&accounts, addresses))
	accountsMap := map[string]int64{}
	for _, account := range accounts {
		accountsMap[account.Address] = account.ID
	}

	historyBatch := effectBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_effects"),
			MaxBatchSize: 0,
		},
	}

	for _, effect := range effects {
		tt.Assert.NoError(
			historyBatch.Add(
				accountsMap[effect.Account],
				effect.HistoryOperationID,
				uint32(effect.Order),
				effect.Type,
				[]byte(effect.DetailsString.String),
			),
		)
	}

	tt.Assert.NoError(historyBatch.Exec())

	valid, err = q.CheckExpOperationEffects(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	// Add a new operation effect to history_effects but not to
	// exp_history_effects, which should make the comparison fail
	tt.Assert.NoError(
		historyBatch.Add(
			accountsMap[addresses[0]],
			toid.New(sequence, 3, 1).ToInt64(),
			1,
			EffectSequenceBumped,
			[]byte("{\"new_seq\": 300000000000}"),
		),
	)

	tt.Assert.NoError(historyBatch.Exec())

	valid, err = q.CheckExpOperationEffects(sequence)
	tt.Assert.NoError(err)
	tt.Assert.False(valid)

	// Add previous effect to exp_history_effects, but make it different to
	// history_effects
	tt.Assert.NoError(
		batch.Add(
			expAccounts[addresses[0]],
			toid.New(sequence, 3, 1).ToInt64(),
			1,
			EffectSequenceBumped,
			[]byte("{\"new_seq\": 3000}"),
		),
	)

	tt.Assert.NoError(batch.Exec())

	valid, err = q.CheckExpOperationEffects(sequence)
	tt.Assert.NoError(err)
	tt.Assert.False(valid)
}
