package actions

import (
	"encoding/hex"
	"mime"
	"net/http"

	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

type SubmitTransactionHandler struct {
	Submitter         *txsub.System
	NetworkPassphrase string
}

type envelopeInfo struct {
	hash   string
	raw    string
	parsed xdr.TransactionEnvelope
}

func extractEnvelopeInfo(raw string, passphrase string) (envelopeInfo, error) {
	result := envelopeInfo{raw: raw}
	err := xdr.SafeUnmarshalBase64(raw, &result.parsed)
	if err != nil {
		return result, err
	}

	var hash [32]byte
	hash, err = network.HashTransactionInEnvelope(result.parsed, passphrase)
	if err != nil {
		return result, err
	}
	result.hash = hex.EncodeToString(hash[:])
	return result, nil
}

func (handler SubmitTransactionHandler) validateBodyType(r *http.Request) error {
	c := r.Header.Get("Content-Type")
	if c == "" {
		return nil
	}

	mt, _, err := mime.ParseMediaType(c)
	if err != nil {
		return errors.Wrap(err, "Could not determine mime type")
	}

	if mt != "application/x-www-form-urlencoded" && mt != "multipart/form-data" {
		return &hProblem.UnsupportedMediaType
	}
	return nil
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
		return nil, &hProblem.Timeout
	}

	switch err := result.Err.(type) {
	case *txsub.FailedTransactionError:
		rcr := horizon.TransactionResultCodes{}
		resourceadapter.PopulateTransactionResultCodes(
			r.Context(),
			info.hash,
			&rcr,
			err,
		)

		return nil, &problem.P{
			Type:   "transaction_failed",
			Title:  "Transaction Failed",
			Status: http.StatusBadRequest,
			Detail: "The transaction failed when submitted to the stellar network. " +
				"The `extras.result_codes` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://www.stellar.org/developers/guides/concepts/list-of-operations.html",
			Extras: map[string]interface{}{
				"envelope_xdr": info.raw,
				"result_xdr":   err.ResultXDR,
				"result_codes": rcr,
			},
		}
	}

	return nil, result.Err
}

func (handler SubmitTransactionHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	if err := handler.validateBodyType(r); err != nil {
		return nil, err
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

	submission := handler.Submitter.Submit(
		r.Context(),
		info.raw,
		info.parsed,
		info.hash,
	)

	select {
	case result := <-submission:
		return handler.response(r, info, result)
	case <-r.Context().Done():
		return nil, &hProblem.Timeout
	}
}
