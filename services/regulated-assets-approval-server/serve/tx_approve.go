package serve

import (
	"context"
	"net/http"

	"github.com/stellar/go/txnbuild"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

const (
	Rejected = "rejected"
)

type txApproveHandler struct {
	DistributionAccount string
}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

type txApproveResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (h txApproveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	in := txApproveRequest{}
	err := httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding input parameters"))
		httpErr := NewHTTPError(http.StatusBadRequest, "Invalid input parameters")
		httpErr.Render(w)
		return
	}
	rejectedResponse, err := h.isRejected(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httpError)
		if !ok {
			httpErr = serverError
		}
		httpErr.Render(w)
		return
	}
	if rejectedResponse != nil {
		httpjson.Render(w, rejectedResponse, httpjson.JSON)
	}
}

func (h txApproveHandler) isRejected(ctx context.Context, in txApproveRequest) (*txApproveResponse, error) {
	log.Ctx(ctx).Info(in.Transaction) //!DEBUG REMOVE
	if in.Transaction == "" {
		return &txApproveResponse{
			Status:  Rejected,
			Message: "Missing parameter \"tx\"",
		}, nil
	}
	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(in.Transaction)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Parsing transaction %s failed", in.Transaction))
		return &txApproveResponse{
			Status:  Rejected,
			Message: "Invalid parameter \"tx\"",
		}, NewHTTPError(http.StatusBadRequest, `Parsing transaction failed.`)
	}
	log.Ctx(ctx).Info(parsed) //!DEBUG REMOVE
	//! From services/recoverysigner/internal/serve/account_sign.go 109,17
	tx, ok := parsed.Transaction()
	if !ok {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Transaction %s is not a simple transaction.", in.Transaction))
		return &txApproveResponse{
			Status:  Rejected,
			Message: "Invalid parameter \"tx\"",
		}, NewHTTPError(http.StatusBadRequest, `Transaction is not a simple transaction.`)
	}
	log.Ctx(ctx).Info(tx) //!DEBUG REMOVE

	// Check that the transaction's source account and any operations it
	// contains references only to this account.
	if tx.SourceAccount().AccountID == h.DistributionAccount {
		log.Ctx(ctx).Error(errors.Wrapf(err,
			"Transaction %s sourceAccount %s the same as the server distribution account %s",
			in.Transaction,
			tx.SourceAccount().AccountID,
			h.DistributionAccount))
		return &txApproveResponse{
			Status:  Rejected,
			Message: "The source account is invalid.",
		}, NewHTTPError(http.StatusBadRequest, `Transaction sourceAccount the same as the server distribution account.`)

	}
	return &txApproveResponse{
		Status:  Rejected,
		Message: "Not implemented.",
	}, NewHTTPError(http.StatusBadRequest, `Not implemented.`)
}
