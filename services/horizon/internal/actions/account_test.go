package actions

import (
	"net/http/httptest"
	"testing"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestAccountInfo(t *testing.T) {
	tt := test.Start(t).Scenario("allow_trust")
	defer tt.Finish()

	account, err := AccountInfo(tt.Ctx, &core.Q{tt.CoreSession()}, "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	tt.Assert.NoError(err)

	tt.Assert.Equal("8589934593", account.Sequence)
	tt.Assert.NotEqual(0, account.LastModifiedLedger)

	for _, balance := range account.Balances {
		if balance.Type == "native" {
			tt.Assert.Equal(uint32(0), balance.LastModifiedLedger)
		} else {
			tt.Assert.NotEqual(uint32(0), balance.LastModifiedLedger)
		}
	}
}
func TestGetAccountsHandlerPageNoResults(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := &GetAccountsHandler{HistoryQ: q}
	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			map[string]string{
				"signer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			},
			map[string]string{},
			q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 0)
}

func TestGetAccountsHandlerPageResults(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := &GetAccountsHandler{HistoryQ: q}

	rows := []history.AccountSigner{
		history.AccountSigner{
			Account: "GABGMPEKKDWR2WFH5AJOZV5PDKLJEHGCR3Q24ALETWR5H3A7GI3YTS7V",
			Signer:  "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			Weight:  1,
		},
		history.AccountSigner{
			Account: "GADTXHUTHIAESMMQ2ZWSTIIGBZRLHUCBLCHPLLUEIAWDEFRDC4SYDKOZ",
			Signer:  "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			Weight:  2,
		},
		history.AccountSigner{
			Account: "GDP347UYM2ZKE6ED6T5OM3BQ5IAS76NKRVEUPNB5PCQ26Z5D7Q7PJOMI",
			Signer:  "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			Weight:  3,
		},
	}

	for _, row := range rows {
		q.CreateAccountSigner(row.Account, row.Signer, row.Weight)
	}

	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			map[string]string{
				"signer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			},
			map[string]string{},
			q.Session,
		),
	)

	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(records))

	for i, row := range rows {
		result := records[i].(protocol.AccountSigner)
		tt.Assert.Equal(row.Account, result.AccountID)
		tt.Assert.Equal(row.Signer, result.Signer.Key)
		tt.Assert.Equal(row.Weight, result.Signer.Weight)
	}
}
