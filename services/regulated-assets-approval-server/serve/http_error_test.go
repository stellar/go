package serve

import (
	"net/http"
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stretchr/testify/require"
)

func TestParseHorizonError(t *testing.T) {
	err := parseHorizonError(nil)
	require.Nil(t, err)

	err = parseHorizonError(errors.New("some error"))
	require.EqualError(t, err, "error submitting transaction: some error")

	horizonError := horizonclient.Error{
		Problem: problem.P{
			Type:   "bad_request",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Extras: map[string]interface{}{
				"result_codes": hProtocol.TransactionResultCodes{
					TransactionCode: "tx_code_here",
					OperationCodes: []string{
						"op_success",
						"op_bad_auth",
					},
				},
			},
		},
	}
	err = parseHorizonError(horizonError)
	require.EqualError(t, err, "error submitting transaction: problem: bad_request, &{TransactionCode:tx_code_here OperationCodes:[op_success op_bad_auth]}\n: horizon error: \"Bad Request\" - check horizon.Error.Problem for more information")
}
