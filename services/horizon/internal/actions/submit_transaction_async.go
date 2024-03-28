package actions

import (
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/protocols/horizon"
	proto "github.com/stellar/go/protocols/stellarcore"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/problem"
	"net/http"
)

var (
	logger = log.New().WithField("service", "async-txsub")
)

type AsyncSubmitTransactionHandler struct {
	NetworkPassphrase string
	DisableTxSub      bool
	ClientWithMetrics stellarcore.ClientWithMetrics
	CoreStateGetter
}

func (handler AsyncSubmitTransactionHandler) GetResource(_ HeaderWriter, r *http.Request) (interface{}, error) {
	logger.SetLevel(log.DebugLevel)

	if err := validateBodyType(r); err != nil {
		logger.WithError(err).Error("Could not validate request body type")
		return nil, err
	}

	raw, err := getString(r, "tx")
	if err != nil {
		logger.WithError(err).Error("Could not read transaction string from request URL")
		return nil, err
	}

	if handler.DisableTxSub {
		logger.WithField("envelope_xdr", raw).Error("Could not submit transaction: transaction submission is disabled")
		return nil, &problem.P{
			Type:   "transaction_submission_disabled",
			Title:  "Transaction Submission Disabled",
			Status: http.StatusMethodNotAllowed,
			Detail: "Transaction submission has been disabled for Horizon. " +
				"To enable it again, remove env variable DISABLE_TX_SUB.",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
			},
		}
	}

	info, err := extractEnvelopeInfo(raw, handler.NetworkPassphrase)
	if err != nil {
		logger.WithField("envelope_xdr", raw).Error("Could not parse transaction envelope")
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
		logger.WithField("envelope_xdr", raw).Error("Stellar-core is not synced")
		return nil, hProblem.StaleHistory
	}

	resp, err := handler.ClientWithMetrics.SubmitTransaction(r.Context(), info.raw, info.parsed)
	if err != nil {
		logger.WithField("envelope_xdr", raw).WithError(err).Error("Transaction submission to stellar-core failed")
		return nil, &problem.P{
			Type:   "transaction_submission_failed",
			Title:  "Transaction Submission Failed",
			Status: http.StatusInternalServerError,
			Detail: "Could not submit transaction to stellar-core. " +
				"The `extras.error` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-async/transaction_submission_failed",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
				"error":        err,
			},
		}
	}

	if resp.IsException() {
		logger.WithField("envelope_xdr", raw).WithError(errors.Errorf(resp.Exception)).Error("Transaction submission exception from stellar-core")
		return nil, &problem.P{
			Type:   "transaction_submission_exception",
			Title:  "Transaction Submission Exception",
			Status: http.StatusInternalServerError,
			Detail: "Received exception from stellar-core." +
				"The `extras.error` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-async/transaction_submission_exception",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
				"error":        resp.Exception,
			},
		}
	}

	switch resp.Status {
	case proto.TXStatusError, proto.TXStatusPending, proto.TXStatusDuplicate, proto.TXStatusTryAgainLater:
		response := horizon.AsyncTransactionSubmissionResponse{
			TxStatus: resp.Status,
			Hash:     info.hash,
		}

		if resp.Status == proto.TXStatusError {
			logger.WithFields(log.F{
				"envelope_xdr": raw,
				"error_xdr":    resp.Error,
				"status":       resp.Status,
				"hash":         info.hash,
			}).Error("Transaction submission to stellar-core resulted in ERROR status")
			response.ErrorResultXDR = resp.Error
		}

		return response, nil
	default:
		logger.WithField("envelope_xdr", raw).WithError(errors.Errorf(resp.Error)).Error("Received invalid submission status from stellar-core")
		return nil, &problem.P{
			Type:   "transaction_submission_invalid_status",
			Title:  "Transaction Submission Invalid Status",
			Status: http.StatusInternalServerError,
			Detail: "Received invalid status from stellar-core." +
				"The `extras.error` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://developers.stellar.org/api/errors/http-status-codes/horizon-specific/transaction-submission-async/transaction_submission_invalid_status",
			Extras: map[string]interface{}{
				"envelope_xdr": raw,
				"error":        resp.Error,
			},
		}
	}

}
