package actions

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/render/problem"
)

func TestGetOperationsFilterByAccountID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	tt.Scenario("base")

	q := &history.Q{tt.HorizonSession()}
	handler := GetOperationsHandler{}

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
		{
			accountID: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			expected:  2,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("account %s operations", tc.accountID), func(t *testing.T) {
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
}

func TestGetOperationsFilterByTxID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	tt.Scenario("base")

	q := &history.Q{tt.HorizonSession()}
	handler := GetOperationsHandler{}

	testCases := []struct {
		desc          string
		transactionID string
		expected      int
		expectedErr   string
		notFound      bool
	}{
		{
			desc:          "operations for 2374...6d4d",
			transactionID: "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d",
			expected:      1,
		},
		{
			desc:          "operations for 164a...33b6",
			transactionID: "164a5064eba64f2cdbadb856bf3448485fc626247ada3ed39cddf0f6902133b6",
			expected:      1,
		},
		{
			desc:          "missing transaction",
			transactionID: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedErr:   "sql: no rows in result set",
			notFound:      true,
		},
		{
			desc:          "uppercase tx hash not accepted",
			transactionID: "2374E99349B9EF7DBA9A5DB3339B78FDA8F34777B1AF33BA468AD5C0DF946D4D",
			expectedErr:   "Transaction hash must be a hex-encoded, lowercase SHA-256 hash",
		},
		{
			desc:          "badly formated tx hash not accepted",
			transactionID: "%00%1E4%5E%EF%BF%BD%EF%BF%BD%EF%BF%BDpVP%EF%BF%BDI&R%0BK%EF%BF%BD%1D%EF%BF%BD%EF%BF%BD=%EF%BF%BD%3F%23%EF%BF%BD%EF%BF%BDl%EF%BF%BD%1El%EF%BF%BD%EF%BF%BD",
			expectedErr:   "Transaction hash must be a hex-encoded, lowercase SHA-256 hash",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.transactionID), func(t *testing.T) {
			records, err := handler.GetResourcePage(
				httptest.NewRecorder(),
				makeRequest(
					t, map[string]string{
						"tx_id": tc.transactionID,
					}, map[string]string{}, q.Session,
				),
			)

			if tc.expectedErr == "" {
				tt.Assert.NoError(err)
				tt.Assert.Len(records, tc.expected)
			} else {
				if tc.notFound {
					tt.Assert.EqualError(err, tc.expectedErr)
				} else {
					tt.Assert.IsType(&problem.P{}, err)
					p := err.(*problem.P)
					tt.Assert.Equal("bad_request", p.Type)
					tt.Assert.Equal("tx_id", p.Extras["invalid_field"])
					tt.Assert.Equal(
						tc.expectedErr,
						p.Extras["reason"],
					)
				}
			}
		})
	}
}

func TestGetOperations(t *testing.T) {
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
	t.Run("With includes(join)", func(t *testing.T) {})
	t.Run("Filter by payments only", func(t *testing.T) {})
}
