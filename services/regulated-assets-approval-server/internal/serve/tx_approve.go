package serve

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/amount"
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
	db                *sqlx.DB
	kycThreshold      int64
	baseURL           string
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
	if h.db == nil {
		return errors.New("database cannot be nil")
	}
	if h.kycThreshold <= 0 {
		return errors.New("kyc threshold cannot be less than or equal to zero")
	}
	if h.baseURL == "" {
		return errors.New("base url cannot be empty")
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

// validateInput validates if the input parameters contain a valid transaction
// and if the source account is not set in a way that would harm the issuer.
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
		log.Ctx(ctx).Errorf("transaction %s sourceAccount is the same as the server issuer account %s", in.Tx, h.issuerKP.Address())
		return NewRejectedTxApprovalResponse("Transaction source account is invalid."), nil
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

	rejectedResponse, tx := h.validateInput(ctx, in)
	if rejectedResponse != nil {
		return rejectedResponse, nil
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

	// validate payment asset is the one supported by the issuer
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

	actionRequiredResponse, err := h.handleActionRequiredResponseIfNeeded(ctx, paymentSource, paymentOp)
	if err != nil {
		return nil, errors.Wrap(err, "handling KYC required payment")
	}
	if actionRequiredResponse != nil {
		return actionRequiredResponse, nil
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

// handleActionRequiredResponseIfNeeded validates and returns an action_required response if the payment requires KYC.
func (h txApproveHandler) handleActionRequiredResponseIfNeeded(ctx context.Context, stellarAddress string, paymentOp *txnbuild.Payment) (*txApprovalResponse, error) {
	paymentAmount, err := amount.ParseInt64(paymentOp.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "parsing payment amount from string to Int64")
	}
	if paymentAmount <= h.kycThreshold {
		return nil, nil
	}

	intendedCallbackID := uuid.New().String()
	const q = `
		WITH new_row AS (
			INSERT INTO accounts_kyc_status (stellar_address, callback_id)
			VALUES ($1, $2)
			ON CONFLICT(stellar_address) DO NOTHING
			RETURNING *
		)
		SELECT callback_id, approved_at, rejected_at FROM new_row
		UNION
		SELECT callback_id, approved_at, rejected_at
		FROM accounts_kyc_status
		WHERE stellar_address = $1
	`
	var (
		callbackID             string
		approvedAt, rejectedAt sql.NullTime
	)
	err = h.db.QueryRowContext(ctx, q, stellarAddress, intendedCallbackID).Scan(&callbackID, &approvedAt, &rejectedAt)
	if err != nil {
		return nil, errors.Wrap(err, "inserting new row into accounts_kyc_status table")
	}

	if approvedAt.Valid {
		return nil, nil
	}

	kycThreshold, err := convertAmountToReadableString(h.kycThreshold)
	if err != nil {
		return nil, errors.Wrap(err, "converting kycThreshold to human readable string")
	}

	if rejectedAt.Valid {
		return NewRejectedTxApprovalResponse(fmt.Sprintf("Your KYC was rejected and you're not authorized for operations above %s %s.", kycThreshold, h.assetCode)), nil
	}

	return NewActionRequiredTxApprovalResponse(
		fmt.Sprintf(`Payments exceeding %s %s require KYC approval. Please provide an email address.`, kycThreshold, h.assetCode),
		fmt.Sprintf("%s/kyc-status/%s", h.baseURL, callbackID),
		[]string{"email_address"},
	), nil
}

func convertAmountToReadableString(threshold int64) (string, error) {
	amountStr := amount.StringFromInt64(threshold)
	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return "", errors.Wrap(err, "converting threshold amount from string to float")
	}
	return fmt.Sprintf("%.2f", amountFloat), nil
}
