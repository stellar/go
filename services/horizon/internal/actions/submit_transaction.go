package actions

import (
	"context"
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/stellarcore"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

type NetworkSubmitter interface {
	Submit(ctx context.Context, rawTx string, envelope xdr.TransactionEnvelope, hash string) <-chan txsub.Result
}

type SubmitTransactionHandler struct {
	Submitter         NetworkSubmitter
	NetworkPassphrase string
	DisableTxSub      bool
	CoreStateGetter
}

func (handler SubmitTransactionHandler) response(r *http.Request, info envelopeInfo, result txsub.Result) (hal.Pageable, error) {
	if result.Err == nil {
		var resource horizon.Transaction
		err := resourceadapter.PopulateTransaction(
			r.Context(),
			info.hash,
			&resource,
			result.Transaction,
		)
		return resource, err
	}

	if result.Err == txsub.ErrTimeout {
		return nil, &hProblem.Timeout
	}

	if result.Err == txsub.ErrCanceled {
		return nil, &hProblem.ClientDisconnected
	}

	if failedErr, ok := result.Err.(*txsub.FailedTransactionError); ok {
		rcr := horizon.TransactionResultCodes{}
		err := resourceadapter.PopulateTransactionResultCodes(
			r.Context(),
			info.hash,
			&rcr,
			failedErr,
		)
		if err != nil {
			return nil, failedErr
		}

		extras := map[string]interface{}{
			"envelope_xdr": info.raw,
			"result_xdr":   failedErr.ResultXDR,
			"result_codes": rcr,
		}
		if failedErr.DiagnosticEventsXDR != "" {
			events, err := stellarcore.DiagnosticEventsToSlice(failedErr.DiagnosticEventsXDR)
			if err != nil {
				return nil, err
			}
			extras["diagnostic_events"] = events
		}

		return nil, &problem.P{
			Type:   "transaction_failed",
			Title:  "Transaction Failed",
			Status: http.StatusBadRequest,
			Detail: "The transaction failed when submitted to the stellar network. " +
				"The `extras.result_codes` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-failed/",
			Extras: extras,
		}
	}

	return nil, result.Err
}

func (handler SubmitTransactionHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	if err := validateBodyType(r); err != nil {
		return nil, err
	}

	if handler.DisableTxSub {
		return nil, &problem.P{
			Type:   "transaction_submission_disabled",
			Title:  "Transaction Submission Disabled",
			Status: http.StatusMethodNotAllowed,
			Detail: "Transaction submission has been disabled for Horizon. " +
				"To enable it again, remove env variable DISABLE_TX_SUB.",
			Extras: map[string]interface{}{},
		}
	}

	raw, err := getString(r, "tx")
	if err != nil {
		return nil, err
	}

	info, err := extractEnvelopeInfo(raw, handler.NetworkPassphrase)
	if err != nil {
		return nil, &problem.P{
			Type:   "transaction_malformed",
			Title:  "Transaction Malformed",
			Status: http.StatusBadRequest,
			Detail: "Horizon could not decode the transaction envelope in this " +
				"request. A transaction should be an XDR TransactionEnvelope struct " +
				"encoded using base64.  The envelope read from this request is " +
				"echoed in the `extras.envelope_xdr` field of this response for your " +
				"convenience.",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
			},
		}
	}

	coreState := handler.GetCoreState()
	if !coreState.Synced {
		return nil, hProblem.StaleHistory
	}

	submission := handler.Submitter.Submit(r.Context(), info.raw, info.parsed, info.hash)

	select {
	case result := <-submission:
		return handler.response(r, info, result)
	case <-r.Context().Done():
		if r.Context().Err() == context.Canceled {
			return nil, hProblem.ClientDisconnected
		}
		return nil, hProblem.Timeout
	}
}
