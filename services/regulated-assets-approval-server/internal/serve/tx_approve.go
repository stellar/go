package serve

import (
	"context"
	"net/http"

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
	horizonClient     horizonclient.ClientInterface
	networkPassphrase string
}

type txApproveRequest struct {
	Tx string `json:"tx" form:"tx"`
}

// validate performs some validations on the provided handler data.
func (h txApproveHandler) validate() error {
	if h.issuerKP == nil {
		return errors.New("issuer keypair cannot be nil")
	}
	if h.assetCode == "" {
		return errors.New("asset code cannot be empty")
	}
	if h.horizonClient == nil {
		return errors.New("horizon client cannot be nil")
	}
	if h.networkPassphrase == "" {
		return errors.New("network passphrase cannot be empty")
	}
	return nil
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
		log.Ctx(ctx).Error(errors.Wrap(err, "validating txApproveHandler"))
		httperror.InternalServerError.Render(w)
		return
	}

	txApproveResp, err := h.txApprove(ctx, in)
	if err != nil {
		httperror.InternalServerError.Render(w)
		return
	}
	txApproveResp.Render(w)
}

// validateInput performs some validations on the provided transaction. It can
// reject the transaction based on general criteria that would be applied in any
// approval server.
func (h txApproveHandler) validateInput(ctx context.Context, in txApproveRequest) (*txApprovalResponse, *txnbuild.Transaction) {
	if in.Tx == "" {
		log.Ctx(ctx).Error(`request is missing parameter "tx".`)
		return NewRejectedTxApprovalResponse(`Missing parameter "tx".`), nil
	}

	genericTx, err := txnbuild.TransactionFromXDR(in.Tx)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "parsing transaction xdr"))
		return NewRejectedTxApprovalResponse(`Invalid parameter "tx".`), nil
	}

	tx, ok := genericTx.Transaction()
	if !ok {
		log.Ctx(ctx).Error(`invalid parameter "tx", generic transaction not given.`)
		return NewRejectedTxApprovalResponse(`Invalid parameter "tx".`), nil
	}

	if tx.SourceAccount().AccountID == h.issuerKP.Address() {
		log.Ctx(ctx).Errorf("transaction %s sourceAccount is the same as the server issuer account %s",
			in.Tx,
			h.issuerKP.Address())
		return NewRejectedTxApprovalResponse("The source account is invalid."), nil
	}

	if len(tx.Operations()) != 1 {
		return NewRejectedTxApprovalResponse("Please submit a transaction with exactly one operation of type payment."), nil
	}

	if tx.Operations()[0].GetSourceAccount() == h.issuerKP.Address() {
		log.Ctx(ctx).Error(`transaction contains one or more operations where sourceAccount is issuer account.`)
		return NewRejectedTxApprovalResponse("There is one or more unauthorized operations in the provided transaction."), nil
	}

	return nil, tx
}

// txApprove is called to validate the input transaction.
// At the moment valid transactions will be rejected with "Not implemented." until subsequent updates.
func (h txApproveHandler) txApprove(ctx context.Context, in txApproveRequest) (resp *txApprovalResponse, err error) {
	defer func() {
		log.Ctx(ctx).Debug("==== will log responses ====")
		log.Ctx(ctx).Debugf("req: %+v", in)
		log.Ctx(ctx).Debugf("resp: %+v", resp)
		log.Ctx(ctx).Debug("====  did log responses ====")
	}()

	txRejectedResp, tx := h.validateInput(ctx, in)
	if txRejectedResp != nil {
		return txRejectedResp, nil
	}
	paymentOp, ok := tx.Operations()[0].(*txnbuild.Payment)
	if !ok {
		log.Ctx(ctx).Error(`transaction contains one or more operations is not of type payment`)
		return NewRejectedTxApprovalResponse("There is one or more unauthorized operations in the provided transaction."), nil
	}
	paymentSource := paymentOp.SourceAccount
	if paymentSource == "" {
		paymentSource = tx.SourceAccount().AccountID
	}

	// TODO: encode the revised transaction currently this is the same transaction given via request
	txe, err := tx.Base64()
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "encoding revised transaction"))
		return nil, err
	}
	return NewRevisedTxApprovalResponse(txe), nil
}
