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
	// 	l.Info("Transaction is not a simple transaction.")
	// 	badRequest.Render(w)
	// 	return
	// }
	// hashHex, err := tx.HashHex(h.NetworkPassphrase)
	// if err != nil {
	// 	l.Error("Error hashing transaction:", err)
	// 	serverError.Render(w)
	// 	return
	// }

	// l = l.WithField("transaction_hash", hashHex)

	// l.Info("Signing transaction.")

	// // Check that the transaction's source account and any operations it
	// // contains references only to this account.
	// if tx.SourceAccount().AccountID != req.Address.Address() {
	// 	l.Info("Transaction's source account is not the account in the request.")
	// 	badRequest.Render(w)
	// 	return
	// }
	return nil, nil
}
