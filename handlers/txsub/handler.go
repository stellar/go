package txsub

import (
	"mime"
	"net/http"
	"sync"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/support/txsub"
)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Validate body type is `application/x-www-form-urlencoded`
	h.validateBodyType(r)

	// 2. Get tx envelope string from form
	tx := r.FormValue("tx")

	// 3. submit tx via submission system
	submission := h.Driver.SubmitTransaction(r.Context(), tx)

	// 4. deal with submission result
	select {
	case result := <-submission:
		h.Result = result
	case <-h.Context.Done():
		h.Err = &problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
			Detail: "Your request timed out before completing.  Please try your " +
				"request again.",
		}
	}

	// 5. load resource struct with appropriate response
	h.loadResource()

	// 6. write hal to return
	if h.Err != nil {
		hal.Render(w, h.Err)
	} else {
		hal.Render(w, h.Resource)
	}

}

func (h *Handler) validateBodyType(r *http.Request) {
	c := r.Header.Get("Content-Type")
	if c == "" {
		return
	}

	mt, _, err := mime.ParseMediaType(c)

	if err != nil {
		h.Err = err
	}

	switch {
	case mt == "application/x-www-form-urlencoded":
		return
	case mt == "multipart/form-data":
		return
	default:
		h.Err = problem.P{
			Type:   "unsupported_media_type",
			Title:  "Unsupported Media Type",
			Status: http.StatusUnsupportedMediaType,
			Detail: "The request has an unsupported content type. Presently, the " +
				"only supported content type is application/x-www-form-urlencoded.",
		}
	}

	return
}

func (h *Handler) loadResource() {
	if h.Result.Err == nil {
		populateTransactionSuccess(h.Context, &h.Resource, h.Result)
		return
	}

	if h.Result.Err == txsub.ErrTimeout || h.Result.Err == txsub.ErrCanceled {
		h.Err = &problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
			Detail: "Your request timed out before completing.  Please try your " +
				"request again.",
		}
		return
	}

	switch err := h.Result.Err.(type) {
	case *txsub.FailedTransactionError:
		rcr := horizon.TransactionResultCodes{}
		populateTransactionResultCodes(h.Context, &rcr, err)

		h.Err = &problem.P{
			Type:   "transaction_failed",
			Title:  "Transaction Failed",
			Status: http.StatusBadRequest,
			Detail: "The transaction failed when submitted to the stellar network. " +
				"The `extras.result_codes` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://www.stellar.org/developers/learn/concepts/list-of-operations.html",
			Extras: map[string]interface{}{
				"envelope_xdr": h.Result.EnvelopeXDR,
				"result_xdr":   err.ResultXDR,
				"result_codes": rcr,
			},
		}
	case *txsub.MalformedTransactionError:
		h.Err = &problem.P{
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
		h.Err = err
	}
}

// Run is the function that runs in the background that triggers Tick each
// second.
func (h *Handler) Run() {
	for {
		select {
		case <-h.Ticks.C:
			h.tick()
		case <-h.Context.Done():
			log.Info("finished background ticker")
			return
		}
	}
}

// Tick triggers txsub to update all of it's background processes.
func (h *Handler) tick() {
	var wg sync.WaitGroup
	log.Debug("ticking app")

	wg.Add(1)
	go func() { h.Driver.Tick(h.Context); wg.Done() }()
	wg.Wait()

	// finally, update metrics
	log.Debug("finished ticking app")
}
