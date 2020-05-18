package actions

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestGetOperations(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	tt.Scenario("base")

	q := &history.Q{tt.HorizonSession()}

	handler := GetOperationsHandler{}

	t.Run("Filter by account_id", func(t *testing.T) {
		testCases := []struct {
			accountID string
			expected  int
		}{
			{
				accountID: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
				expected:  3,
			},
			{
				accountID: "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
				expected:  1,
			},
		}
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%s operations", tc.accountID), func(t *testing.T) {
				records, err := handler.GetResourcePage(
					httptest.NewRecorder(),
					makeRequest(
						t, map[string]string{
							"account_id": tc.accountID,
						}, map[string]string{}, q.Session,
					),
				)
				tt.Assert.NoError(err)
				tt.Assert.Len(records, tc.expected)
			})
		}

	})

	t.Run("Validates cursor as default", func(t *testing.T) {})
	t.Run("Validates cursor within history", func(t *testing.T) {})
	// should this be a middleware?
	t.Run("EnsureHistoryFreshness", func(t *testing.T) {})
	t.Run("No filter", func(t *testing.T) {})
	t.Run("Pagination", func(t *testing.T) {})
	t.Run("Included failed", func(t *testing.T) {
		// should failed if failed txs are not ingested
		// if action.IncludeFailed && !action.App.config.IngestFailedTransactions {
		// 	err := errors.New("`include_failed` parameter is unavailable when Horizon is not ingesting failed " +
		// 		"transactions. Set `INGEST_FAILED_TRANSACTIONS=true` to start ingesting them.")
		// 	action.Err = supportProblem.MakeInvalidFieldProblem("include_failed", err)
		// 	return
		// }

	})
	t.Run("Filter by ledger_id", func(t *testing.T) {})
	t.Run("Filter by tx_id", func(t *testing.T) {
		// validates transaction hash
		// if action.TransactionFilter != "" && !isValidTransactionHash(action.TransactionFilter) {
		// 	action.Err = supportProblem.MakeInvalidFieldProblem("tx_id", errors.New("Invalid transaction hash"))
		// 	return

	})
	t.Run("With includes(join)", func(t *testing.T) {})
	t.Run("Filter by payments only", func(t *testing.T) {})
}
