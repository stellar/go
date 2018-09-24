package horizon

import (
	"testing"
	"time"

	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/db2/history"
)

func TestPaymentActions(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	w := ht.Get("/payments")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(4, w.Body)
	}

	// filtered by ledger
	w = ht.Get("/ledgers/1/payments")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(0, w.Body)
	}

	w = ht.Get("/ledgers/3/payments")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)
	}

	// filtered by account
	w = ht.Get("/accounts/GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2/payments")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)
	}

	// switch scenarios
	ht.T.Scenario("pathed_payment")

	// filtered by transaction
	w = ht.Get("/transactions/b52f16ffb98c047e33b9c2ec30880330cde71f85b3443dae2c5cb86c7d4d8452/payments")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(0, w.Body)
	}

	w = ht.Get("/transactions/1d2a4be72470658f68db50eef29ea0af3f985ce18b5c218f03461d40c47dc292/payments")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)

		// test for existence of source_amount in path payment details
		var records []map[string]interface{}
		ht.UnmarshalPage(w.Body, &records)
		ht.Assert.Equal("10.0000000", records[0]["source_amount"])
	}

	// Regression: negative cursor
	w = ht.Get("/accounts/GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2/payments?cursor=-23667108046966785&order=asc&limit=100")
	ht.Assert.Equal(400, w.Code)
}

func TestPayment_CreatedAt(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	w := ht.Get("/ledgers/3/payments")
	records := []operations.Base{}
	ht.UnmarshalPage(w.Body, &records)

	l := history.Ledger{}
	hq := history.Q{Session: ht.HorizonSession()}
	ht.Require.NoError(hq.LedgerBySequence(&l, 3))

	ht.Assert.WithinDuration(l.ClosedAt, records[0].LedgerCloseTime, 1*time.Second)
}
