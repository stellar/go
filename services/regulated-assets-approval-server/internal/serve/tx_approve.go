package serve

import (
	"context"
	"net/http"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/httperror"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
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

func (h txApproveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	in := txApproveRequest{}
	err := httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding input parameters"))
		httpErr := httperror.NewHTTPError(http.StatusBadRequest, "Invalid input parameters")
		httpErr.Render(w)
		return
	}
	rejectedResponse, err := h.isRejected(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httperror.Error)
		if !ok {
			httpErr = httperror.InternalServerError
		}
		httpErr.Render(w)
		return
	}
	httpjson.RenderStatus(w, http.StatusBadRequest, rejectedResponse, httpjson.JSON)
}

func (h txApproveHandler) isRejected(ctx context.Context, in txApproveRequest) (*txApprovalResponse, error) {
	if in.Transaction == "" {
		return NewRejectedTxApprovalResponse("Missing parameter \"tx\"."), nil
	}

	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(in.Transaction)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Parsing transaction %s failed", in.Transaction))
		return NewRejectedTxApprovalResponse(invalidParamErr), nil
	}

	tx, ok := parsed.Transaction()
	if !ok {
		log.Ctx(ctx).Errorf("Transaction %s is not a simple transaction.", in.Transaction)
		return NewRejectedTxApprovalResponse(invalidParamErr), nil
	}

	// Check if transaction's source account is the same as the server issuer account.
	if tx.SourceAccount().AccountID == h.issuerKP.Address() {
		log.Ctx(ctx).Errorf("Transaction %s sourceAccount is the same as the server issuer account %s",
			in.Transaction,
			h.issuerKP.Address())
		return NewRejectedTxApprovalResponse(invalidSrcAccErr), nil
	}

	// Check if transaction's operation(s)' sourceaccount is the same as the server issuer account.
	for _, op := range tx.Operations() {
		if op.GetSourceAccount() == h.issuerKP.Address() {
			return NewRejectedTxApprovalResponse(unauthorizedOpErr), nil
		}
	}

	return NewRejectedTxApprovalResponse(notImplementedErr), nil
}
