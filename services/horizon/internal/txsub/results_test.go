package txsub

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestGetMissingTx(t *testing.T) {
	tt := test.Start(t)
	tt.Scenario("base")
	defer tt.Finish()
	q := &history.Q{SessionInterface: tt.HorizonSession()}
	hash := "adf1efb9fd253f53cbbe6230c131d2af19830328e52b610464652d67d2fb7195"

	_, err := txResultByHash(tt.Ctx, q, hash)
	tt.Assert.Equal(ErrNoResults, err)
}

func TestGetFailedTx(t *testing.T) {
	tt := test.Start(t)
	tt.Scenario("failed_transactions")
	defer tt.Finish()
	q := &history.Q{SessionInterface: tt.HorizonSession()}
	hash := "aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf"

	_, err := txResultByHash(tt.Ctx, q, hash)
	tt.Assert.Equal("AAAAAAAAAGT/////AAAAAQAAAAAAAAAB/////gAAAAA=", err.(*FailedTransactionError).ResultXDR)
}
