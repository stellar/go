package horizon

import (
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

// This file contains the action:
//
// TransactionShowAction: single transaction by sequence, by hash or id

// Interface verification
var _ actions.JSONer = (*TransactionShowAction)(nil)

// TransactionShowAction renders a ledger found by its sequence number.
type TransactionShowAction struct {
	Action
	Hash     string
	Record   history.Transaction
	Resource horizon.Transaction
}

func (action *TransactionShowAction) loadParams() {
	action.Hash = action.GetString("tx_id")
}

func (action *TransactionShowAction) loadRecord() {
	action.Err = action.HistoryQ().TransactionByHash(&action.Record, action.Hash)
}

func (action *TransactionShowAction) loadResource() {
	resourceadapter.PopulateTransaction(action.R.Context(), &action.Resource, action.Record)
}

// JSON is a method for actions.JSON
func (action *TransactionShowAction) JSON() error {
	action.Do(
		action.EnsureHistoryFreshness,
		action.loadParams,
		action.loadRecord,
		action.loadResource,
		func() { hal.Render(action.W, action.Resource) },
	)
	return action.Err
}

// Interface verification
var _ actions.JSONer = (*TransactionCreateAction)(nil)

// TransactionCreateAction submits a transaction to the stellar-core network
// on behalf of the requesting client.
type TransactionCreateAction struct {
	Action
	TX       string
	Result   txsub.Result
	Resource horizon.TransactionSuccess
}

// JSON format action handler
func (action *TransactionCreateAction) JSON() error {
	action.Do(
		action.loadTX,
		action.loadResult,
		action.loadResource,
		func() { hal.Render(action.W, action.Resource) },
	)
	return action.Err
}

func (action *TransactionCreateAction) loadTX() {
	action.ValidateBodyType()
	action.TX = action.GetString("tx")
}

func (action *TransactionCreateAction) loadResult() {
	submission := action.App.submitter.Submit(action.R.Context(), action.TX)

	select {
	case result := <-submission:
		action.Result = result
	case <-action.R.Context().Done():
		action.Err = &hProblem.Timeout
	}
}

func (action *TransactionCreateAction) loadResource() {
	if action.Result.Err == nil {
		resourceadapter.PopulateTransactionSuccess(action.R.Context(), &action.Resource, action.Result)
		return
	}

	if action.Result.Err == txsub.ErrTimeout {
		action.Err = &hProblem.Timeout
		return
	}

	if action.Result.Err == txsub.ErrCanceled {
		action.Err = &hProblem.Timeout
		return
	}

	switch err := action.Result.Err.(type) {
	case *txsub.FailedTransactionError:
		rcr := horizon.TransactionResultCodes{}
		resourceadapter.PopulateTransactionResultCodes(action.R.Context(), &rcr, err)

		action.Err = &problem.P{
			Type:   "transaction_failed",
			Title:  "Transaction Failed",
			Status: http.StatusBadRequest,
			Detail: "The transaction failed when submitted to the stellar network. " +
				"The `extras.result_codes` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://www.stellar.org/developers/learn/concepts/list-of-operations.html",
			Extras: map[string]interface{}{
				"envelope_xdr": action.Result.EnvelopeXDR,
				"result_xdr":   err.ResultXDR,
				"result_codes": rcr,
			},
		}
	case *txsub.MalformedTransactionError:
		action.Err = &problem.P{
			Type:   "transaction_malformed",
			Title:  "Transaction Malformed",
			Status: http.StatusBadRequest,
			Detail: "Horizon could not decode the transaction envelope in this " +
				"request. A transaction should be an XDR TransactionEnvelope struct " +
				"encoded using base64.  The envelope read from this request is " +
				"echoed in the `extras.envelope_xdr` field of this response for your " +
				"convenience.",
			Extras: map[string]interface{}{
				"envelope_xdr": err.EnvelopeXDR,
			},
		}
	default:
		action.Err = err
	}
}
