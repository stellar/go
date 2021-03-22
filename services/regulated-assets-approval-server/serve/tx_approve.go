package serve

import (
	"context"
	"net/http"

	"github.com/stellar/go/keypair"

	"github.com/stellar/go/txnbuild"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

const (
	RejectedStatus    = "rejected"
	MissingParamMsg   = "Missing parameter \"tx\"."
	InvalidParamMsg   = "Invalid parameter \"tx\"."
	InternalErrMsg    = "Internal Error."
	InvalidSrcAccMsg  = "The source account is invalid."
	UnauthorizedOpMsg = "There is one or more unauthorized operations in the provided transaction."
	NotImplementedMsg = "Not implemented."
)

type txApproveHandler struct {
	issuerAccountSecret string
	assetCode           string
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
	if in.Transaction == "" {
		return &txApproveResponse{
			Status:  RejectedStatus,
			Message: MissingParamMsg,
		}, nil
	}

	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(in.Transaction)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Parsing transaction %s failed", in.Transaction))
		return &txApproveResponse{
			Status:  RejectedStatus,
			Message: InvalidParamMsg,
		}, NewHTTPError(http.StatusBadRequest, `Parsing transaction failed.`)
	}
	tx, ok := parsed.Transaction()
	if !ok {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Transaction %s is not a simple transaction.", in.Transaction))
		return &txApproveResponse{
			Status:  RejectedStatus,
			Message: InvalidParamMsg,
		}, NewHTTPError(http.StatusBadRequest, `Transaction is not a simple transaction.`)
	}

	// Check if transaction's sourceaccount is the same as the server issuer account.
	issuerKP, err := keypair.Parse(h.issuerAccountSecret)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "Parsing issuer secret failed."))
		return &txApproveResponse{
			Status:  RejectedStatus,
			Message: InternalErrMsg,
		}, NewHTTPError(http.StatusBadRequest, `Parsing issuer secret failed.`)
	}
	if tx.SourceAccount().AccountID == issuerKP.Address() {
		log.Ctx(ctx).Error(errors.Wrapf(err,
			"Transaction %s sourceAccount %s the same as the server issuer account %s",
			in.Transaction,
			tx.SourceAccount().AccountID,
			issuerKP.Address()))
		return &txApproveResponse{
			Status:  RejectedStatus,
			Message: InvalidSrcAccMsg,
		}, NewHTTPError(http.StatusBadRequest, `Transaction sourceAccount the same as the server issuer account.`)

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
				Status:  RejectedStatus,
				Message: UnauthorizedOpMsg,
			}, NewHTTPError(http.StatusBadRequest, UnauthorizedOpMsg)
		}
	}

	return &txApproveResponse{
		Status:  RejectedStatus,
		Message: NotImplementedMsg,
	}, NewHTTPError(http.StatusBadRequest, NotImplementedMsg)
}
