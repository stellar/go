package txsub

import (
	"context"
	"net/http"
	"net/url"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/txsub"
)

func populateTransactionResultCodes(ctx context.Context,
	dest *horizon.TransactionResultCodes,
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
func populateTransactionSuccess(ctx context.Context, dest *horizon.TransactionSuccess, result txsub.Result) {
	dest.Hash = result.Hash
	dest.Ledger = result.LedgerSequence
	dest.Env = result.EnvelopeXDR
	dest.Result = result.ResultXDR
	dest.Meta = result.ResultMetaXDR

	lb := hal.LinkBuilder{baseURL(ctx)}
	dest.Links.Transaction = lb.Link("/transactions", result.Hash)
	return
}

// BaseURL returns the "base" url for this request, defined as a url containing
// the Host and Scheme portions of the request uri.
func baseURL(ctx context.Context) *url.URL {
	r := requestFromContext(ctx)

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

var requestContextKey = 0

func requestFromContext(ctx context.Context) *http.Request {
	found := ctx.Value(&requestContextKey)

	if found == nil {
		return nil
	}

	return found.(*http.Request)
}
