package txsub

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestGetIngestedTx(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &history.Q{Session: tt.HorizonSession()}
	hash := "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"
	tx, err := txResultByHash(q, hash)
	tt.Assert.NoError(err)
	tt.Assert.Equal(hash, tx.TransactionHash)
}

func TestGetMissingTx(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &history.Q{Session: tt.HorizonSession()}
	hash := "adf1efb9fd253f53cbbe6230c131d2af19830328e52b610464652d67d2fb7195"

	_, err := txResultByHash(q, hash)
	tt.Assert.Equal(ErrNoResults, err)
}

func TestGetFailedTx(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()
	q := &history.Q{Session: tt.HorizonSession()}
	hash := "aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf"

	_, err := txResultByHash(q, hash)
	tt.Assert.Equal("AAAAAAAAAGT/////AAAAAQAAAAAAAAAB/////gAAAAA=", err.(*FailedTransactionError).ResultXDR)
}
