// Package txsub provides a pluggable handler that satisfies the Stellar
// transaction submission implementation defined in support/txsub. Add an
// instance of `Handler` onto your router to allow a server to satisfy the protocol.
//
// The central type in this package is the "Driver" interface.  Implementing
// this interface allows a developer to plug in their own backend for the
// submission service.
package txsub

import (
	"context"
	"net/http"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/support/txsub"
)

// Handler represents an http handler that can service http requests that
// conform to the Stellar txsub service.  This handler should be added to
// your chosen mux at the path `/tx` (and for good measure `/tx/` if your
// middleware doesn't normalize trailing slashes).
type Handler struct {
	Driver  Driver
	Context context.Context
	Ticks   *time.Ticker
}

// Driver is a wrapper around the configurable parts of txsub.System. By requiring
// that the following methods are implemented, we have essentially created an
// interface for the txsub.System struct. You may notice we have not included the
// SubmitOnce() method here, that is because that method does not need to be
// exposed to the Handler struct.
type Driver interface {
	// SubmitTransaction submits the provided base64 encoded transaction envelope
	// to the network using this submission system.
	SubmitTransaction(context.Context, string) <-chan txsub.Result
	// Tick triggers the system to update itself with any new data available.
	Tick(ctx context.Context)
}

// HorizonProxyDriver is a pre-baked implementation of `Driver` that provides the
// entirety of the submission system necessary to satisfy the protocol.
type HorizonProxyDriver struct {
	submissionSystem *txsub.System
}

// HorizonProxyResultProvider represents a Horizon Proxy that can lookup Result objects
// by transaction hash or by [address,sequence] pairs.
type HorizonProxyResultProvider struct {
	client *horizon.Client
}

// HorizonProxySequenceProvider represents a Horizon proxy that can lookup the current
// sequence number of an account.
type HorizonProxySequenceProvider struct {
	client *horizon.Client
}

// HorizonProxySubmitterProvider represents the high-level "submit a transaction to
// an upstream horizon" provider.
type HorizonProxySubmitterProvider struct {
	client *horizon.Client
}

// TransactionSuccess is ported from protocols/horizon without the links. Here links are
// not needed since a) it is not necessarilly clear which domain to fill in (in the
// horizon implementation of txsub, context was used to grab the local horizon domain)
// and b) the tx hash is returned and that is sufficient to query the transaction in
// question.
type TransactionSuccess struct {
	Hash   string `json:"hash"`
	Ledger int32  `json:"ledger"`
	Env    string `json:"envelope_xdr"`
	Result string `json:"result_xdr"`
	Meta   string `json:"result_meta_xdr"`
}

// HandlerResponse is used by each request to capture important data to return to the user.
type HandlerResponse struct {
	Result   txsub.Result
	Err      error
	Resource TransactionSuccess
}

// LoadResource translates its result into either an Err in case of error or a Resource in
// case of success.
func (hr *HandlerResponse) LoadResource() {
	if hr.Result.Err == nil {
		hr.Resource.Hash = hr.Result.Hash
		hr.Resource.Ledger = hr.Result.LedgerSequence
		hr.Resource.Env = hr.Result.EnvelopeXDR
		hr.Resource.Result = hr.Result.ResultXDR
		hr.Resource.Meta = hr.Result.ResultMetaXDR
		return
	}

	if hr.Result.Err == txsub.ErrTimeout || hr.Result.Err == txsub.ErrCanceled {
		hr.Err = &problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
			Detail: "Your request timed out before completing.  Please try your " +
				"request again.",
		}
		return
	}

	switch err := hr.Result.Err.(type) {
	case *txsub.FailedTransactionError:
		rcr := horizon.TransactionResultCodes{}

		// Populate transaction result codes
		var e error
		rcr.TransactionCode, e = err.TransactionResultCode()
		if e == nil {
			rcr.OperationCodes, e = err.OperationResultCodes()
		}

		hr.Err = &problem.P{
			Type:   "transaction_failed",
			Title:  "Transaction Failed",
			Status: http.StatusBadRequest,
			Detail: "The transaction failed when submitted to the stellar network. " +
				"The `extras.result_codes` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://www.stellar.org/developers/learn/concepts/list-of-operations.html",
			Extras: map[string]interface{}{
				"envelope_xdr": hr.Result.EnvelopeXDR,
				"result_xdr":   err.ResultXDR,
				"result_codes": rcr,
			},
		}
	case *txsub.MalformedTransactionError:
		hr.Err = &problem.P{
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
		hr.Err = err
	}
}
