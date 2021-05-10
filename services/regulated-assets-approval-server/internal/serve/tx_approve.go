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
	issuerKP  *keypair.Full
	assetCode string
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
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding input parameters"))
		httpErr := httperror.NewHTTPError(http.StatusBadRequest, "Invalid input parameters")
		httpErr.Render(w)
		return
	}

	txApproveResp := h.txApprove(ctx, in)

	txApproveResp.Render(w)
}

// validateInput performs some validations on the provided transaction. It can
// reject the transaction based on general criteria that would be applied in any
// approval server.
func (h txApproveHandler) validateInput(ctx context.Context, in txApproveRequest) *txApprovalResponse {
	if in.Tx == "" {
		log.Ctx(ctx).Error(`request is missing parameter "tx".`)
		return NewRejectedTxApprovalResponse(`Missing parameter "tx".`)
	}

	genericTx, err := txnbuild.TransactionFromXDR(in.Tx)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "parsing transaction xdr"))
		return NewRejectedTxApprovalResponse(`Invalid parameter "tx".`)
	}

	tx, ok := genericTx.Transaction()
	if !ok {
		log.Ctx(ctx).Error(`invalid parameter "tx", generic transaction not given.`)
		return NewRejectedTxApprovalResponse(`Invalid parameter "tx".`)
	}

	if tx.SourceAccount().AccountID == h.issuerKP.Address() {
		log.Ctx(ctx).Errorf("transaction %s sourceAccount is the same as the server issuer account %s",
			in.Tx,
			h.issuerKP.Address())
		return NewRejectedTxApprovalResponse("The source account is invalid.")
	}

	for _, op := range tx.Operations() {
		if op.GetSourceAccount() == h.issuerKP.Address() {
			log.Ctx(ctx).Error(`transaction contains one or more operations where sourceAccount is issuer account.`)
			return NewRejectedTxApprovalResponse("There is one or more unauthorized operations in the provided transaction.")
		}

		_, ok := op.(*txnbuild.Payment)
		if !ok {
			log.Ctx(ctx).Error(`transaction contains one or more operations is not of type payment`)
			return NewRejectedTxApprovalResponse("There is one or more unauthorized operations in the provided transaction.")
		}
	}

	// Temporarily reject all approval attempts(even those that meet the validateInput standards)
	return NewRejectedTxApprovalResponse("Not implemented.")
}

// txApprove is called to validate the input transaction.
// At the moment valid transactions will be rejected with "Not implemented." until subsequent updates.
func (h txApproveHandler) txApprove(ctx context.Context, in txApproveRequest) (resp *txApprovalResponse) {
	defer func() {
		log.Ctx(ctx).Debug("==== will log responses ====")
		log.Ctx(ctx).Debugf("req: %+v", in)
		log.Ctx(ctx).Debugf("resp: %+v", resp)
		log.Ctx(ctx).Debug("====  did log responses ====")
	}()

	txRejectedResp := h.validateInput(ctx, in)

	return txRejectedResp
}
