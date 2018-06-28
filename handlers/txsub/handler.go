package txsub

import (
	"context"
	"mime"
	"net/http"
	"net/url"
	"sync"

	"github.com/stellar/go/clients/horizon"
	phorizon "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/support/txsub"
)

// Run is the function that runs in the background that triggers Tick each
// second.
func (h *Handler) Run() {
	for {
		select {
		case <-h.Ticks.C:
			h.Tick()
		case <-h.Context.Done():
			log.Info("finished background ticker")
			return
		}
	}
}

// Tick triggers txsub to update all of it's background processes.
func (h *Handler) Tick() {
	var wg sync.WaitGroup
	log.Debug("ticking app")

	wg.Add(1)
	go func() { h.Driver.Tick(h.Context); wg.Done() }()
	wg.Wait()

	// finally, update metrics
	log.Debug("finished ticking app")
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Validate body type is `application/x-www-form-urlencoded`
	c := r.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(c)
	if err != nil {
		h.Err = err
	}
	switch {
	case mt == "application/x-www-form-urlencoded":
		break
	case mt == "multipart/form-data":
		break
	default:
		h.Err = problem.P{
			Type:   "unsupported_media_type",
			Title:  "Unsupported Media Type",
			Status: http.StatusUnsupportedMediaType,
			Detail: "The request has an unsupported content type. Presently, the " +
				"only supported content type is application/x-www-form-urlencoded.",
		}
	}
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
	// 5. load resource
	h.loadResource()
	// 6. write hal
	hal.Render(w, h.Resource)
}

func (action *Handler) loadResource() {
	if action.Result.Err == nil {
		PopulateTransactionSuccess(action.Context, &action.Resource, action.Result)
		return
	}

	if action.Result.Err == txsub.ErrTimeout {
		action.Err = &problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
			Detail: "Your request timed out before completing.  Please try your " +
				"request again.",
		}
		return
	}

	if action.Result.Err == txsub.ErrCanceled {
		action.Err = &problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
			Detail: "Your request timed out before completing.  Please try your " +
				"request again.",
		}
		return
	}

	switch err := action.Result.Err.(type) {
	case *txsub.FailedTransactionError:
		rcr := horizon.TransactionResultCodes{}
		PopulateTransactionResultCodes(action.Context, &rcr, err)

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

func PopulateTransactionResultCodes(ctx context.Context,
	dest *phorizon.TransactionResultCodes,
	fail *txsub.FailedTransactionError,
) (err error) {

	dest.TransactionCode, err = fail.TransactionResultCode()
	if err != nil {
		return
	}

	dest.OperationCodes, err = fail.OperationResultCodes()
	if err != nil {
		return
	}

	return
}

// Populate fills out the details
func PopulateTransactionSuccess(ctx context.Context, dest *phorizon.TransactionSuccess, result txsub.Result) {
	dest.Hash = result.Hash
	dest.Ledger = result.LedgerSequence
	dest.Env = result.EnvelopeXDR
	dest.Result = result.ResultXDR
	dest.Meta = result.ResultMetaXDR

	lb := hal.LinkBuilder{BaseURL(ctx)}
	dest.Links.Transaction = lb.Link("/transactions", result.Hash)
	return
}

// BaseURL returns the "base" url for this request, defined as a url containing
// the Host and Scheme portions of the request uri.
func BaseURL(ctx context.Context) *url.URL {
	r := RequestFromContext(ctx)

	if r == nil {
		return nil
	}

	var scheme string
	switch {
	case r.Header.Get("X-Forwarded-Proto") != "":
		scheme = r.Header.Get("X-Forwarded-Proto")
	case r.TLS != nil:
		scheme = "https"
	default:
		scheme = "http"
	}

	return &url.URL{
		Scheme: scheme,
		Host:   r.Host,
	}
}

var RequestContextKey = 0

func RequestFromContext(ctx context.Context) *http.Request {
	found := ctx.Value(&RequestContextKey)

	if found == nil {
		return nil
	}

	return found.(*http.Request)
}
