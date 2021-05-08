package serve

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/httperror"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
)

type txApproveHandler struct {
	issuerKP          *keypair.Full
	assetCode         string
	baseURL           string
	db                *sqlx.DB
	horizonClient     horizonclient.ClientInterface
	kycThreshold      int64
	networkPassphrase string
}

type txApproveRequest struct {
	Tx string `json:"tx" form:"tx"`
}

func (h txApproveHandler) validate() error {
	if h.issuerKP == nil {
		return errors.New("issuer keypair cannot be nil")
	}

	if h.assetCode == "" {
		return errors.New("asset code cannot be empty")
	}

	if h.baseURL == "" {
		return errors.New("base url cannot be empty")
	}

	if h.db == nil {
		return errors.New("db cannot be nil")
	}

	if h.horizonClient == nil {
		return errors.New("horizon client cannot be nil")
	}

	if h.kycThreshold <= 0 {
		return errors.New("kyc threshold cannot be less than or equal to zero")
	}

	if h.networkPassphrase == "" {
		return errors.New("network passphrase cannot be empty")
	}

	return nil
}

// validateInput performs some validations on the provided transaction. It can
// reject the transaction based on general criteria that would be applied in any
// approval server.
func (h txApproveHandler) validateInput(ctx context.Context, in txApproveRequest) (*txnbuild.Transaction, *txApprovalResponse, error) {
	if in.Tx == "" {
		return nil, NewRejectedTxApprovalResponse(`Missing parameter "tx".`), nil
	}

	genericTx, err := txnbuild.TransactionFromXDR(in.Tx)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "parsing transaction xdr"))
		return nil, NewRejectedTxApprovalResponse(`Invalid parameter "tx".`), nil
	}

	tx, ok := genericTx.Transaction()
	if !ok {
		return nil, NewRejectedTxApprovalResponse(`Invalid parameter "tx".`), nil
	}

	if tx.SourceAccount().AccountID == h.issuerKP.Address() {
		log.Ctx(ctx).Errorf("Transaction %s sourceAccount is the same as the server issuer account %s",
			in.Tx,
			h.issuerKP.Address())
		return tx, NewRejectedTxApprovalResponse("The source account is invalid."), nil
	}

	for _, op := range tx.Operations() {
		if op.GetSourceAccount() == h.issuerKP.Address() {
			return tx, NewRejectedTxApprovalResponse("There is one or more unauthorized operations in the provided transaction."), nil
		}
	}

	return tx, nil, nil
}

func (h txApproveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := h.validate()
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "validating txApproveHandler"))
		httperror.InternalServerError.Render(w)
		return
	}

	in := txApproveRequest{}
	err = httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding input parameters"))
		httpErr := httperror.NewHTTPError(http.StatusBadRequest, "Invalid input parameters")
		httpErr.Render(w)
		return
	}

	txApproveResp, err := h.txApprove(ctx, in)
	if err != nil {
		httperror.InternalServerError.Render(w)
		return
	}
	txApproveResp.Render(w)
}

// txApprove is called to validate the input transaction.
// At the moment valid transactions will be rejected with "Not implemented." until subsequent updates.
func (h txApproveHandler) txApprove(ctx context.Context, in txApproveRequest) (resp *txApprovalResponse, err error) {
	defer func() {
		log.Ctx(ctx).Debug("==== will log responses ====")
		log.Ctx(ctx).Debugf("req: %+v", in)
		log.Ctx(ctx).Debugf("resp: %+v", resp)
		log.Ctx(ctx).Debugf("err: %+v", err)
		log.Ctx(ctx).Debug("====  did log responses ====")
	}()

	_, txRejectedResp, err := h.validateInput(ctx, in)
	if txRejectedResp != nil || err != nil {
		return txRejectedResp, err
	}
	// Temporarily reject all approval attempts(even those that meet the validateInput standards)
	return NewRejectedTxApprovalResponse("Not implemented."), nil
}
