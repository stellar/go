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

const (
	rejectedStatus      = "rejected"
	revisedStatus       = "revised"
	revisedHappyPathMsg = "Authorization and deauthorization operations were added."
	missingParamErr     = "Missing parameter \"tx\"."
	invalidParamErr     = "Invalid parameter \"tx\"."
	internalErrErr      = "Internal Error."
	invalidSrcAccErr    = "The source account is invalid."
	unauthorizedOpErr   = "There is one or more unauthorized operations in the provided transaction."
	notImplementedErr   = "Not implemented."
)

var (
	txnBuildPaymentType = reflect.TypeOf((*txnbuild.Payment)(nil))
)

type txApproveHandler struct {
	issuerAccountSecret string
	assetCode           string
	networkPassphrase   string
}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

type txApproveResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Transaction string `json:"tx"`
	Error       string `json:"error"`
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
		log.Ctx(ctx).Error(errors.Wrapf(err, "Transaction %s is not a simple transaction.", in.Transaction))
		return nil, NewHTTPError(http.StatusBadRequest, `Transaction submitted is not a simple transaction.`)
	}
	log.Ctx(ctx).Debug(tx)

	issuerKP, err := keypair.ParseFull(h.issuerAccountSecret)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "Parsing issuer secret failed."))
		return nil, NewHTTPError(http.StatusBadRequest, `Parsing issuer secret failed.`)
	}

	log.Ctx(ctx).Debug(issuerKP)

	// Check if transaction has only one operation. The happy path requirement for now
	if len(tx.Operations()) > 1 {
		log.Ctx(ctx).Error(errors.Wrapf(nil, "Transaction has %d operations.", len(tx.Operations())))
		return nil, NewHTTPError(http.StatusBadRequest, `Too many operations in transaction.`)
	}
	// Check if operation is a payment. The happy path requirement for now
	op, ok := tx.Operations()[0].(*txnbuild.Payment)
	if !ok {
		log.Ctx(ctx).Error(errors.Wrapf(nil, "Transaction contains a %q operation.", reflect.TypeOf(op)))
		return nil, NewHTTPError(http.StatusBadRequest, `Not a payment operation.`)
	}
	asset := txnbuild.CreditAsset{
		Code:   h.assetCode,
		Issuer: issuerKP.Address(),
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
		Timebounds: txnbuild.NewTimeout(300),
	})
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "building transaction"))
		return nil, NewHTTPError(http.StatusBadRequest, `Failed to build and sandwich transaction.`)
	}

	tx, err = tx.Sign(h.networkPassphrase, issuerKP)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "signing transaction"))
		return nil, NewHTTPError(http.StatusBadRequest, `Failed to sign transaction.`)
	}

	txEnc, err := tx.Base64()
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "unable to serialize tx"))
		return nil, NewHTTPError(http.StatusBadRequest, `unable to serialize tx.`)
	}
	return &txApproveResponse{
		Status:      revisedStatus,
		Message:     revisedHappyPathMsg,
		Transaction: txEnc,
	}, nil
}
