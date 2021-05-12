package serve

import (
	"context"
	"net/http"
	"strconv"

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
		httperror.InternalServer.Render(w)
		return
	}

	in := txApproveRequest{}
	err = httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding txApproveRequest"))
		httperror.BadRequest.Render(w)
		return
	}

	txApproveResp, err := h.txApprove(ctx, in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "validating the input transaction for approval"))
		httperror.InternalServer.Render(w)
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
func (h txApproveHandler) txApprove(ctx context.Context, in txApproveRequest) (resp *txApprovalResponse, err error) {
	defer func() {
		log.Ctx(ctx).Debug("==== will log responses ====")
		log.Ctx(ctx).Debugf("req: %+v", in)
		log.Ctx(ctx).Debugf("resp: %+v", resp)
		log.Ctx(ctx).Debugf("err: %+v", err)
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

	issuerAddress := h.issuerKP.Address()
	if paymentOp.Asset.GetCode() != h.assetCode || paymentOp.Asset.GetIssuer() != issuerAddress {
		log.Ctx(ctx).Error(`the payment asset is not supported by this issuer`)
		return NewRejectedTxApprovalResponse("The payment asset is not supported by this issuer."), nil
	}

	acc, err := h.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: paymentSource})
	if err != nil {
		return nil, errors.Wrapf(err, "getting detail for payment source account %s", issuerAddress)
	}
	// validate the sequence number
	accountSequence, err := strconv.ParseInt(acc.Sequence, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing account sequence number %q from string to int64", acc.Sequence)
	}
	if tx.SourceAccount().Sequence != accountSequence+1 {
		log.Ctx(ctx).Errorf(`invalid transaction sequence number tx.SourceAccount().Sequence: %d, accountSequence+1:%d`, tx.SourceAccount().Sequence, accountSequence+1)
		return NewRejectedTxApprovalResponse("Invalid transaction sequence number."), nil
	}

	// build the transaction
	revisedOperations := []txnbuild.Operation{
		&txnbuild.AllowTrust{
			Trustor:       paymentSource,
			Type:          paymentOp.Asset,
			Authorize:     true,
			SourceAccount: issuerAddress,
		},
		&txnbuild.AllowTrust{
			Trustor:       paymentOp.Destination,
			Type:          paymentOp.Asset,
			Authorize:     true,
			SourceAccount: issuerAddress,
		},
		paymentOp,
		&txnbuild.AllowTrust{
			Trustor:       paymentOp.Destination,
			Type:          paymentOp.Asset,
			Authorize:     false,
			SourceAccount: issuerAddress,
		},
		&txnbuild.AllowTrust{
			Trustor:       paymentSource,
			Type:          paymentOp.Asset,
			Authorize:     false,
			SourceAccount: issuerAddress,
		},
	}
	revisedTx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &acc,
		IncrementSequenceNum: true,
		Operations:           revisedOperations,
		BaseFee:              300,
		Timebounds:           txnbuild.NewTimeout(300),
	})
	if err != nil {
		return nil, errors.Wrap(err, "building transaction")
	}

	revisedTx, err = revisedTx.Sign(h.networkPassphrase, h.issuerKP)
	if err != nil {
		return nil, errors.Wrap(err, "signing transaction")
	}

	txe, err := revisedTx.Base64()
	if err != nil {
		return nil, errors.Wrap(err, "encoding revised transaction")
	}
	return NewRevisedTxApprovalResponse(txe), nil
}
