package serve

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
		return errors.New("db cannot be nil")
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
	// Validate if payment operation requires KYC.
	var kycRequiredResponse *txApprovalResponse
	kycRequiredResponse, err = h.handleKYCRequiredOperationIfNeeded(ctx, paymentSource, paymentOp)
	if err != nil {
		return nil, errors.Wrap(err, "handling KYC required payment")
	}
	if kycRequiredResponse != nil {
		return kycRequiredResponse, nil
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

// handleKYCRequiredOperationIfNeeded validates and returns an "action required"(or rejected) response if the payment requires KYC.
func (h txApproveHandler) handleKYCRequiredOperationIfNeeded(ctx context.Context, stellarAddress string, paymentOp *txnbuild.Payment) (*txApprovalResponse, error) {
	// validate payment operation against KYC condition(s).
	// partialKYCRequiredMessage is used to build "action required" or rejected messages.
	partialKYCRequiredMessage, err := h.validateKYC(paymentOp)
	if err != nil {
		return nil, errors.Wrap(err, "validating KYC")
	}
	if partialKYCRequiredMessage == "" {
		return nil, nil
	}
	//Build "Action Required" message.
	actionFields := []string{"email_address"}
	actionFieldsMessage := strings.Join(actionFields, ", ")
	FullKYCRequiredMessage := fmt.Sprintf(`%s requires your %s for KYC approval.`, partialKYCRequiredMessage, actionFieldsMessage)
	// Create new callBackID used to insert new accounts_kyc_status table entrees.
	intendedCallbackID := uuid.New().String()
	// This query inserts a new row with the intended callbackID or selects the
	// existing callbackID for that stellar account, if it already exists.
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
		err = errors.Wrap(err, "getting or creating callback id")
		log.Ctx(ctx).Error(err)
		return nil, err
	}
	if approvedAt.Valid {
		return nil, nil
	}
	if rejectedAt.Valid {
		return NewRejectedTxApprovalResponse(fmt.Sprintf(`Your KYC was rejected and you're not authorized for "%s".`, partialKYCRequiredMessage)), nil
	}
	return NewActionRequiredTxApprovalResponse(
		FullKYCRequiredMessage,
		fmt.Sprintf("%s/kyc-status/%s", h.baseURL, callbackID),
		actionFields,
	), nil
}

// validateKYC returns a partial "action required" message (used in NewActionRequiredTxApprovalResponse) if the payment operation meets KYC conditions
// Currently rule(s) are, checking if payment amount is > KYCThreshold amount.
func (h txApproveHandler) validateKYC(paymentOp *txnbuild.Payment) (string, error) {
	paymentAmount, err := amount.ParseInt64(paymentOp.Amount)
	if err != nil {
		return "", errors.Wrapf(err, "parsing account payment amount %d from string to Int64", paymentAmount)
	}
	if paymentAmount > h.kycThreshold {
		return fmt.Sprintf(`Payments exceeding %s %s`, amount.StringFromInt64(h.kycThreshold), h.assetCode), nil
	}
	return "", nil
}
