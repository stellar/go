package serve

import (
	"context"
	"net/http"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
)

const (
	rejectedStatus    = "rejected"
	missingParamErr   = "Missing parameter \"tx\"."
	invalidParamErr   = "Invalid parameter \"tx\"."
	internalErrErr    = "Internal Error."
	invalidSrcAccErr  = "The source account is invalid."
	unauthorizedOpErr = "There is one or more unauthorized operations in the provided transaction."
	notImplementedErr = "Not implemented."
)

type txApproveHandler struct {
	issuerAccountSecret string
	assetCode           string
}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

type txApproveResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
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
	}
	if rejectedResponse != nil {
		httpjson.RenderStatus(w, http.StatusBadRequest, rejectedResponse, httpjson.JSON)
	}
}

func (h txApproveHandler) isRejected(ctx context.Context, in txApproveRequest) (*txApproveResponse, error) {
	if in.Transaction == "" {
		return &txApproveResponse{
			Status: rejectedStatus,
			Error:  missingParamErr,
		}, nil
	}

	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(in.Transaction)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Parsing transaction %s failed", in.Transaction))
		return &txApproveResponse{
			Status: rejectedStatus,
			Error:  invalidParamErr,
		}, nil
	}

	tx, ok := parsed.Transaction()
	if !ok {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Transaction %s is not a simple transaction.", in.Transaction))
		return &txApproveResponse{
			Status: rejectedStatus,
			Error:  invalidParamErr,
		}, nil
	}

	issuerKP, err := keypair.Parse(h.issuerAccountSecret)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "Parsing issuer secret failed."))
		return &txApproveResponse{
			Status: rejectedStatus,
			Error:  internalErrErr,
		}, NewHTTPError(http.StatusBadRequest, `Parsing issuer secret failed.`)
	}

	// Check if transaction's sourceaccount is the same as the server issuer account.
	if tx.SourceAccount().AccountID == issuerKP.Address() {
		log.Ctx(ctx).Error(errors.Wrapf(err,
			"Transaction %s sourceAccount %s the same as the server issuer account %s",
			in.Transaction,
			tx.SourceAccount().AccountID,
			issuerKP.Address()))
		return &txApproveResponse{
			Status: rejectedStatus,
			Error:  invalidSrcAccErr,
		}, nil

	}

	// Check if transaction's operation(s)' sourceaccount is the same as the server issuer account.
	for _, op := range tx.Operations() {
		opSourceAccount := op.GetSourceAccount()
		if opSourceAccount == "" {
			continue
		}
		if op.GetSourceAccount() == issuerKP.Address() {
			log.Ctx(ctx).Error(errors.Wrapf(err,
				"Unauthorized operation from %s in the provided transaction",
				op.GetSourceAccount()))
			return &txApproveResponse{
				Status: rejectedStatus,
				Error:  unauthorizedOpErr,
			}, nil
		}
	}

	return &txApproveResponse{
		Status: rejectedStatus,
		Error:  notImplementedErr,
	}, nil
}
