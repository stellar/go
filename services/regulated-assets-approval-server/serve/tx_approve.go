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

type sep8Status string

const (
	Sep8StatusRejected sep8Status = "rejected"
)

const (
	missingParamErr   = "Missing parameter \"tx\"."
	invalidParamErr   = "Invalid parameter \"tx\"."
	internalErrErr    = "Internal Error."
	invalidSrcAccErr  = "The source account is invalid."
	unauthorizedOpErr = "There is one or more unauthorized operations in the provided transaction."
	notImplementedErr = "Not implemented."
)

type txApproveHandler struct {
	issuerKP  *keypair.Full
	assetCode string
}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

type txApproveResponse struct {
	Status sep8Status `json:"status"`
	Error  string     `json:"error"`
}

func NewRejectedTXApproveResponse(errorMessage string) *txApproveResponse {
	return &txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  errorMessage,
	}
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
	httpjson.RenderStatus(w, http.StatusBadRequest, rejectedResponse, httpjson.JSON)
}

func (h txApproveHandler) isRejected(ctx context.Context, in txApproveRequest) (*txApproveResponse, error) {
	if in.Transaction == "" {
		return NewRejectedTXApproveResponse(missingParamErr), nil
	}

	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(in.Transaction)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Parsing transaction %s failed", in.Transaction))
		return NewRejectedTXApproveResponse(invalidParamErr), nil
	}

	tx, ok := parsed.Transaction()
	if !ok {
		log.Ctx(ctx).Errorf("Transaction %s is not a simple transaction.", in.Transaction)
		return NewRejectedTXApproveResponse(invalidParamErr), nil
	}

	// Check if transaction's source account is the same as the server issuer account.
	if tx.SourceAccount().AccountID == h.issuerKP.Address() {
		log.Ctx(ctx).Errorf("Transaction %s sourceAccount is the same as the server issuer account %s",
			in.Transaction,
			h.issuerKP.Address())
		return NewRejectedTXApproveResponse(invalidSrcAccErr), nil
	}

	// Check if transaction's operation(s)' sourceaccount is the same as the server issuer account.
	for _, op := range tx.Operations() {
		if op.GetSourceAccount() == h.issuerKP.Address() {
			return NewRejectedTXApproveResponse(unauthorizedOpErr), nil
		}
	}

	return NewRejectedTXApproveResponse(notImplementedErr), nil
}
