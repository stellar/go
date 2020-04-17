package results

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txsub"
)

func TestResultProvider(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	rp := &DB{
		Core:    &core.Q{Session: tt.CoreSession()},
		History: &history.Q{Session: tt.HorizonSession()},
	}

	// Regression: ensure a transaction that is not ingested still returns the
	// result
	hash := "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"
	ret := rp.ResultByHash(tt.Ctx, hash)

	tt.Require.NoError(ret.Err)
	tt.Assert.Equal(hash, ret.Transaction.TransactionHash)
}

func TestResultProviderHorizonOnly(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	rp := &DB{
		Core:           &core.Q{Session: tt.CoreSession()},
		History:        &history.Q{Session: tt.HorizonSession()},
		SkipCoreChecks: true,
	}

	hash := "adf1efb9fd253f53cbbe6230c131d2af19830328e52b610464652d67d2fb7195"
	_, err := tt.CoreSession().ExecRaw("INSERT INTO txhistory VALUES ('" + hash + "', 5, 1, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAO5rKAAAAAAAAAAABVvwF9wAAAECDzqvkQBQoNAJifPRXDoLhvtycT3lFPCQ51gkdsFHaBNWw05S/VhW0Xgkr0CBPE4NaFV2Kmcs3ZwLmib4TRrML', 'I3Tpk0m57326ml2zM5t4/ajzR3exrzO6RorVwN+UbU0AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAQAAAAAAAAABAAAAAwAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/7UAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrNryTTUAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAA7msoAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');")
	tt.Require.NoError(err)

	ret := rp.ResultByHash(tt.Ctx, hash)

	tt.Require.Equal(ret.Err, txsub.ErrNoResults)
}

func TestResultFailed(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	rp := &DB{
		Core:    &core.Q{Session: tt.CoreSession()},
		History: &history.Q{Session: tt.HorizonSession()},
	}

	hash := "aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf"

	// Ignore core db results
	_, err := tt.CoreSession().ExecRaw(
		`DELETE FROM txhistory WHERE txid = ?`,
		hash,
	)
	tt.Require.NoError(err)

	ret := rp.ResultByHash(tt.Ctx, hash)

	tt.Require.Error(ret.Err)
	tt.Assert.Equal("AAAAAAAAAGT/////AAAAAQAAAAAAAAAB/////gAAAAA=", ret.Err.(*txsub.FailedTransactionError).ResultXDR)
}

func TestResultFailedNotInHorizonDB(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	rp := &DB{
		Core:           &core.Q{Session: tt.CoreSession()},
		History:        &history.Q{Session: tt.HorizonSession()},
		SkipCoreChecks: false,
	}

	hash := "aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf"

	// remove tx from horizon db
	_, err := tt.HorizonSession().ExecRaw(
		`DELETE FROM history_transactions WHERE transaction_hash = ?`,
		hash,
	)
	tt.Require.NoError(err)

	ret := rp.ResultByHash(tt.Ctx, hash)

	tt.Require.Error(ret.Err)
	tt.Assert.Equal("AAAAAAAAAGT/////AAAAAQAAAAAAAAAB/////gAAAAA=", ret.Err.(*txsub.FailedTransactionError).ResultXDR)
}
