package horizon

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestOperationActions_Index(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	// no filter
	w := ht.Get("/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(4, w.Body)
	}

	// filtered by ledger sequence
	w = ht.Get("/ledgers/1/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(0, w.Body)
	}

	w = ht.Get("/ledgers/2/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(3, w.Body)
	}

	w = ht.Get("/ledgers/3/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)
	}

	// filtered by account
	w = ht.Get("/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(3, w.Body)
	}

	w = ht.Get("/accounts/GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)
	}

	w = ht.Get("/accounts/GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(2, w.Body)
	}

	// filtered by transaction
	w = ht.Get("/transactions/2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)
	}

	w = ht.Get("/transactions/164a5064eba64f2cdbadb856bf3448485fc626247ada3ed39cddf0f6902133b6/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)
	}

	// 400 for invalid tx hash
	w = ht.Get("/transactions/ /operations")
	ht.Assert.Equal(400, w.Code)

	w = ht.Get("/transactions/invalid/operations")
	ht.Assert.Equal(400, w.Code)

	w = ht.Get("/transactions/1d2a4be72470658f68db50eef29ea0af3f985ce18b5c218f03461d40c47dc29/operations")
	ht.Assert.Equal(400, w.Code)

	w = ht.Get("/transactions/1d2a4be72470658f68db50eef29ea0af3f985ce18b5c218f03461d40c47dc29222/operations")
	ht.Assert.Equal(400, w.Code)

	// filtered by ledger
	w = ht.Get("/ledgers/3/operations")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(1, w.Body)
	}

	// missing ledger
	w = ht.Get("/ledgers/100/operations")
	ht.Assert.Equal(404, w.Code)
}

func TestOperationActions_Show_Failed(t *testing.T) {
	ht := StartHTTPTest(t, "failed_transactions")
	defer ht.Finish()

	// Should show successful transactions only
	w := ht.Get("/operations?limit=200")

	if ht.Assert.Equal(200, w.Code) {
		records := []operations.Base{}
		ht.UnmarshalPage(w.Body, &records)

		successful := 0
		failed := 0

		for _, op := range records {
			if op.TransactionSuccessful {
				successful++
			} else {
				failed++
			}
		}

		ht.Assert.Equal(8, successful)
		ht.Assert.Equal(0, failed)
	}

	// Should show all transactions: both successful and failed
	w = ht.Get("/operations?limit=200&include_failed=true")

	if ht.Assert.Equal(200, w.Code) {
		records := []operations.Base{}
		ht.UnmarshalPage(w.Body, &records)

		successful := 0
		failed := 0

		for _, op := range records {
			if op.TransactionSuccessful {
				successful++
			} else {
				failed++
			}
		}

		ht.Assert.Equal(8, successful)
		ht.Assert.Equal(1, failed)
	}

	w = ht.Get("/transactions/aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf/operations")

	if ht.Assert.Equal(200, w.Code) {
		records := []operations.Base{}
		ht.UnmarshalPage(w.Body, &records)

		ht.Assert.Equal(1, len(records))
		for _, op := range records {
			ht.Assert.False(op.TransactionSuccessful)
			ht.Assert.Equal("aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf", op.TransactionHash)
		}
	}

	w = ht.Get("/transactions/56e3216045d579bea40f2d35a09406de3a894ecb5be70dbda5ec9c0427a0d5a1/operations")

	if ht.Assert.Equal(200, w.Code) {
		records := []operations.Base{}
		ht.UnmarshalPage(w.Body, &records)

		ht.Assert.Equal(1, len(records))
		for _, op := range records {
			ht.Assert.True(op.TransactionSuccessful)
		}
	}

	// NULL value
	_, err := ht.HorizonSession().ExecRaw(
		`UPDATE history_transactions SET successful = NULL WHERE transaction_hash = ?`,
		"56e3216045d579bea40f2d35a09406de3a894ecb5be70dbda5ec9c0427a0d5a1",
	)
	ht.Require.NoError(err)

	w = ht.Get("/transactions/56e3216045d579bea40f2d35a09406de3a894ecb5be70dbda5ec9c0427a0d5a1/operations")

	if ht.Assert.Equal(200, w.Code) {
		records := []operations.Base{}
		ht.UnmarshalPage(w.Body, &records)

		ht.Assert.Equal(1, len(records))
		for _, op := range records {
			ht.Assert.True(op.TransactionSuccessful)
		}
	}
}

func TestOperationActions_IncludeTransactions(t *testing.T) {
	ht := StartHTTPTest(t, "failed_transactions")
	defer ht.Finish()

	w := ht.Get("/operations?account_id=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")
	ht.Assert.Equal(200, w.Code)
	withoutTransactions := []operations.Base{}
	ht.UnmarshalPage(w.Body, &withoutTransactions)

	w = ht.Get("/operations?account_id=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2&join=transactions")
	ht.Assert.Equal(200, w.Code)
	withTransactions := []operations.Base{}
	ht.UnmarshalPage(w.Body, &withTransactions)

	for i, operation := range withTransactions {
		getTransaction := ht.Get("/transactions/" + operation.Transaction.ID)
		ht.Assert.Equal(200, getTransaction.Code)
		var getTransactionResponse horizon.Transaction
		err := json.Unmarshal(getTransaction.Body.Bytes(), &getTransactionResponse)

		ht.Require.NoError(err, "failed to parse body")
		tx := operation.Transaction
		ht.Assert.Equal(*tx, getTransactionResponse)

		withTransactions[i].Transaction = nil
	}

	ht.Assert.Equal(withoutTransactions, withTransactions)
}

func TestOperationActions_SSE(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	ctx := context.Background()
	stream := sse.NewStream(ctx, httptest.NewRecorder())
	oa := OperationIndexAction{
		Action: *NewTestAction(ctx, "/operations?account_id=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2"),
	}

	oa.SSE(stream)
	tt.Require.NoError(oa.Err)

	streamWithTransactions := sse.NewStream(ctx, httptest.NewRecorder())
	oaWithTransactions := OperationIndexAction{
		Action: *NewTestAction(ctx, "/operations?account_id=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2&join=transactions"),
	}
	oaWithTransactions.SSE(streamWithTransactions)
	tt.Require.NoError(oaWithTransactions.Err)
}

func TestOperationActions_Show(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	// exists
	w := ht.Get("/operations/8589938689")
	if ht.Assert.Equal(200, w.Code) {
		var result operations.Base
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Require.NoError(err, "failed to parse body")
		ht.Assert.Equal("8589938689", result.PT)
		ht.Assert.Equal("2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d", result.TransactionHash)
	}

	// doesn't exist
	w = ht.Get("/operations/9589938689")
	ht.Assert.Equal(404, w.Code)

	// before history
	ht.ReapHistory(1)
	w = ht.Get("/operations/8589938689")
	ht.Assert.Equal(410, w.Code)
}

func TestOperationActions_Regressions(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	// ensure that trying to stream ops from an account that doesn't exist
	// fails before streaming the hello message.  Regression test for #285
	w := ht.Get("/accounts/GAS2FZOQRFVHIDY35TUSBWFGCROPLWPZVFRN5JZEOUUVRGDRZGHPBLYZ/operations?limit=1", test.RequestHelperStreaming)
	if ht.Assert.Equal(404, w.Code) {
		ht.Assert.ProblemType(w.Body, "not_found")
	}

	// #202 - price is not shown on manage_offer operations
	test.LoadScenario("trades")
	w = ht.Get("/operations/25769807873")
	if ht.Assert.Equal(200, w.Code) {
		var result operations.ManageSellOffer
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Require.NoError(err, "failed to parse body")
		ht.Assert.Equal("1.0000000", result.Price)
		ht.Assert.Equal(int32(1), result.PriceR.N)
		ht.Assert.Equal(int32(1), result.PriceR.D)
	}
}

func TestOperation_CreatedAt(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	w := ht.Get("/ledgers/3/operations")
	records := []operations.Base{}
	ht.UnmarshalPage(w.Body, &records)

	l := history.Ledger{}
	hq := history.Q{Session: ht.HorizonSession()}
	ht.Require.NoError(hq.LedgerBySequence(&l, 3))

	ht.Assert.WithinDuration(l.ClosedAt, records[0].LedgerCloseTime, 1*time.Second)
}

func TestOperation_BumpSequence(t *testing.T) {
	ht := StartHTTPTest(t, "kahuna")
	defer ht.Finish()

	w := ht.Get("/operations/261993009153")
	if ht.Assert.Equal(200, w.Code) {
		var result operations.BumpSequence
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Require.NoError(err, "failed to parse body")
		ht.Assert.Equal("bump_sequence", result.Type)
		ht.Assert.Equal("300000000003", result.BumpTo)
	}
}

func TestOperationEffect_BumpSequence(t *testing.T) {
	ht := StartHTTPTest(t, "kahuna")
	defer ht.Finish()

	w := ht.Get("/operations/249108107265/effects")
	if ht.Assert.Equal(200, w.Code) {
		var result []effects.SequenceBumped
		ht.UnmarshalPage(w.Body, &result)
		ht.Assert.Equal(int64(300000000000), result[0].NewSeq)
	}
}

func TestOperation_IncludeTransaction(t *testing.T) {
	ht := StartHTTPTest(t, "kahuna")
	defer ht.Finish()

	withoutTransaction := ht.Get("/operations/261993009153")
	ht.Assert.Equal(200, withoutTransaction.Code)
	var responseWithoutTransaction operations.BumpSequence
	err := json.Unmarshal(withoutTransaction.Body.Bytes(), &responseWithoutTransaction)
	ht.Require.NoError(err, "failed to parse body")
	ht.Assert.Nil(responseWithoutTransaction.Transaction)

	withTransaction := ht.Get("/operations/261993009153?join=transactions")
	ht.Assert.Equal(200, withTransaction.Code)
	var responseWithTransaction operations.BumpSequence
	err = json.Unmarshal(withTransaction.Body.Bytes(), &responseWithTransaction)
	ht.Require.NoError(err, "failed to parse body")

	transactionInOperationsResponse := *responseWithTransaction.Transaction
	responseWithTransaction.Transaction = nil
	ht.Assert.Equal(responseWithoutTransaction, responseWithTransaction)

	getTransaction := ht.Get("/transactions/" + transactionInOperationsResponse.ID)
	ht.Assert.Equal(200, getTransaction.Code)
	var getTransactionResponse horizon.Transaction
	err = json.Unmarshal(getTransaction.Body.Bytes(), &getTransactionResponse)
	ht.Require.NoError(err, "failed to parse body")
	ht.Assert.Equal(transactionInOperationsResponse, getTransactionResponse)
}

// TestOperationActions_Show_Extra_TxID tests if failed transactions are not returned
// when `tx_id` GET param is present. This was happening because `base.GetString()`
// method retuns values from the query when URL param is not present.
func TestOperationActions_Show_Extra_TxID(t *testing.T) {
	ht := StartHTTPTest(t, "failed_transactions")
	defer ht.Finish()

	w := ht.Get("/accounts/GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON/operations?limit=200&tx_id=abc")

	if ht.Assert.Equal(200, w.Code) {
		records := []operations.Base{}
		ht.UnmarshalPage(w.Body, &records)

		successful := 0
		failed := 0

		for _, op := range records {
			if op.TransactionSuccessful {
				successful++
			} else {
				failed++
			}
		}

		ht.Assert.Equal(3, successful)
		ht.Assert.Equal(0, failed)
	}
}
