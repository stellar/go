package serve

import (
	"context"
	"net/http"
	"reflect"

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
	Sep8StatusRevised  sep8Status = "revised"
)

const (
	missingParamErr     = "Missing parameter \"tx\"."
	invalidParamErr     = "Invalid parameter \"tx\"."
	internalErrErr      = "Internal Error."
	invalidSrcAccErr    = "The source account is invalid."
	unauthorizedOpErr   = "There is one or more unauthorized operations in the provided transaction."
	exceededOpsLenErr   = "There are too many operations in the provided transaction."
	notImplementedErr   = "Not implemented."
	revisedHappyPathMsg = "Authorization and deauthorization operations were added."
)

type txApproveHandler struct {
	issuerKP          *keypair.Full
	assetCode         string
	networkPassphrase string
}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

type txApproveResponse struct {
	Status      sep8Status `json:"status"`
	Message     string     `json:"message,omitempty"`
	Transaction string     `json:"tx,omitempty"`
	Error       string     `json:"error,omitempty"`
}

func NewRejectedTXApproveResponse(errorMessage string) *txApproveResponse {
	return &txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  errorMessage,
	}

}

func NewRevisedTXApproveResponse(message string, tx string) *txApproveResponse {
	return &txApproveResponse{
		Status:      Sep8StatusRevised,
		Message:     message,
		Transaction: tx,
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
	resp, err := h.isRejected(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httpError)
		if !ok {
			httpErr = serverError
		}
		httpErr.Render(w)
		return
	}
	if resp != nil {
		httpjson.RenderStatus(w, http.StatusBadRequest, resp, httpjson.JSON)
		return
	}
	resp, err = h.Approve(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httpError)
		if !ok {
			httpErr = serverError
		}
		httpErr.Render(w)
		return
	}
	httpjson.RenderStatus(w, http.StatusOK, resp, httpjson.JSON)
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

	// Check if transaction has only one operation.
	if len(tx.Operations()) > 1 {
		log.Ctx(ctx).Errorf("Transaction has %d operations.", len(tx.Operations()))
		return NewRejectedTXApproveResponse(exceededOpsLenErr), nil
	}

	// Check if operation is a payment.
	op, ok := tx.Operations()[0].(*txnbuild.Payment)
	if !ok {
		log.Ctx(ctx).Errorf("Transaction contains a %q operation.", reflect.TypeOf(op))
		return NewRejectedTXApproveResponse(unauthorizedOpErr), nil
	}

	// Check if transaction's operation sourceaccount is the same as the server issuer account.
	if op.GetSourceAccount() == h.issuerKP.Address() {
		log.Ctx(ctx).Errorf("Transaction's operation sourceaccount %q is the same as issuer account.", op.GetSourceAccount())
		return NewRejectedTXApproveResponse(unauthorizedOpErr), nil
	}

	return nil, nil
}

func (h txApproveHandler) Approve(ctx context.Context, in txApproveRequest) (*txApproveResponse, error) {
	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(in.Transaction)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Parsing transaction %s failed", in.Transaction))
		return nil, NewHTTPError(http.StatusBadRequest, `Parsing transaction failed.`)
	}

	tx, ok := parsed.Transaction()
	if !ok {
		log.Ctx(ctx).Errorf("Transaction %s is not a simple transaction.", in.Transaction)
		return nil, NewHTTPError(http.StatusBadRequest, `Transaction submitted is not a simple transaction.`)
	}

	op, ok := tx.Operations()[0].(*txnbuild.Payment)
	if !ok {
		log.Ctx(ctx).Errorf("Transaction contains a %q operation.", reflect.TypeOf(op))
		return nil, NewHTTPError(http.StatusBadRequest, `Transaction contains is not a Payment operation.`)
	}

	asset := txnbuild.CreditAsset{
		Code:   h.assetCode,
		Issuer: h.issuerKP.Address(),
	}

	tx, err = txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &txnbuild.SimpleAccount{AccountID: tx.SourceAccount().AccountID},
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{
			&txnbuild.AllowTrust{
				Trustor:   tx.SourceAccount().AccountID,
				Type:      asset,
				Authorize: true,
			},
			&txnbuild.AllowTrust{
				Trustor:   op.Destination,
				Type:      asset,
				Authorize: true,
			},
			op,
			&txnbuild.AllowTrust{
				Trustor:   tx.SourceAccount().AccountID,
				Type:      asset,
				Authorize: false,
			},
			&txnbuild.AllowTrust{
				Trustor:   op.Destination,
				Type:      asset,
				Authorize: false,
			},
		},
		BaseFee:    tx.BaseFee(),
		Timebounds: tx.Timebounds(),
	})
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "building transaction"))
		return nil, NewHTTPError(http.StatusBadRequest, `Failed to build and sandwich transaction.`)
	}

	tx, err = tx.Sign(h.networkPassphrase, h.issuerKP)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "signing transaction"))
		return nil, NewHTTPError(http.StatusBadRequest, `Failed to sign transaction.`)
	}

	txEnc, err := tx.Base64()
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "unable to serialize tx"))
		return nil, NewHTTPError(http.StatusBadRequest, `unable to serialize tx.`)
	}

	return NewRevisedTXApproveResponse(revisedHappyPathMsg, txEnc), nil
}
