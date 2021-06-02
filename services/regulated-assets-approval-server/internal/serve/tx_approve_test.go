package serve

import (
	"context"
	"net/http"
	"testing"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxApproveHandlerValidate(t *testing.T) {
	// empty issuer KP.
	h := txApproveHandler{}
	err := h.validate()
	require.EqualError(t, err, "issuer keypair cannot be nil")

	// empty asset code.
	issuerAccKeyPair := keypair.MustRandom()
	h = txApproveHandler{
		issuerKP: issuerAccKeyPair,
	}
	err = h.validate()
	require.EqualError(t, err, "asset code cannot be empty")

	// No Horizon client.
	h = txApproveHandler{
		issuerKP:  issuerAccKeyPair,
		assetCode: "FOOBAR",
	}
	err = h.validate()
	require.EqualError(t, err, "horizon client cannot be nil")

	// No network passphrase.
	horizonMock := horizonclient.MockClient{}
	h = txApproveHandler{
		issuerKP:      issuerAccKeyPair,
		assetCode:     "FOOBAR",
		horizonClient: &horizonMock,
	}
	err = h.validate()
	require.EqualError(t, err, "network passphrase cannot be empty")

	// No db.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
	}
	err = h.validate()
	require.EqualError(t, err, "database cannot be nil")

	// Empty kycThreshold.
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
	}
	err = h.validate()
	require.EqualError(t, err, "kyc threshold cannot be less than or equal to zero")

	// Negative kycThreshold.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      -1,
	}
	err = h.validate()
	require.EqualError(t, err, "kyc threshold cannot be less than or equal to zero")

	// no baseURL.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      1,
	}
	err = h.validate()
	require.EqualError(t, err, "base url cannot be empty")

	// Success.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      1,
		baseURL:           "https://example.com",
	}
	err = h.validate()
	require.NoError(t, err)
}

func TestTxApproveHandler_validateInput(t *testing.T) {
	h := txApproveHandler{}
	ctx := context.Background()

	// rejects if incoming tx is empty
	in := txApproveRequest{}
	txApprovalResp, gotTx := h.validateInput(ctx, in)
	require.Equal(t, NewRejectedTxApprovalResponse("Missing parameter \"tx\"."), txApprovalResp)
	require.Nil(t, gotTx)

	// rejects if incoming tx is invalid
	in = txApproveRequest{Tx: "foobar"}
	txApprovalResp, gotTx = h.validateInput(ctx, in)
	require.Equal(t, NewRejectedTxApprovalResponse("Invalid parameter \"tx\"."), txApprovalResp)
	require.Nil(t, gotTx)

	// rejects if incoming tx is a fee bump transaction
	in = txApproveRequest{Tx: "AAAABQAAAAAo/cVyQxyGh7F/Vsj0BzfDYuOJvrwgfHGyqYFpHB5RCAAAAAAAAADIAAAAAgAAAAAo/cVyQxyGh7F/Vsj0BzfDYuOJvrwgfHGyqYFpHB5RCAAAAGQAEfDJAAAAAQAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAAAo/cVyQxyGh7F/Vsj0BzfDYuOJvrwgfHGyqYFpHB5RCAAAAAAAAAAAAJiWgAAAAAAAAAAAAAAAAAAAAAA="}
	txApprovalResp, gotTx = h.validateInput(ctx, in)
	require.Equal(t, NewRejectedTxApprovalResponse("Invalid parameter \"tx\"."), txApprovalResp)
	require.Nil(t, gotTx)

	// rejects if tx source account is the issuer
	clientKP := keypair.MustRandom()
	h.issuerKP = keypair.MustRandom()

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &horizon.Account{
			AccountID: h.issuerKP.Address(),
			Sequence:  "1",
		},
		IncrementSequenceNum: true,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		BaseFee:              300,
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: clientKP.Address(),
				Amount:      "1",
				Asset:       txnbuild.NativeAsset{},
			},
		},
	})
	require.NoError(t, err)
	txe, err := tx.Base64()
	require.NoError(t, err)

	in.Tx = txe
	txApprovalResp, gotTx = h.validateInput(ctx, in)
	require.Equal(t, NewRejectedTxApprovalResponse("Transaction source account is invalid."), txApprovalResp)
	require.Nil(t, gotTx)

	// rejects if tx contains more than one operation
	tx, err = txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &horizon.Account{
			AccountID: clientKP.Address(),
			Sequence:  "1",
		},
		IncrementSequenceNum: true,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		BaseFee:              300,
		Operations: []txnbuild.Operation{
			&txnbuild.BumpSequence{},
			&txnbuild.Payment{
				Destination: clientKP.Address(),
				Amount:      "1.0000000",
				Asset:       txnbuild.NativeAsset{},
			},
		},
	})
	require.NoError(t, err)
	txe, err = tx.Base64()
	require.NoError(t, err)

	in.Tx = txe
	txApprovalResp, gotTx = h.validateInput(ctx, in)
	require.Equal(t, NewRejectedTxApprovalResponse("Please submit a transaction with exactly one operation of type payment."), txApprovalResp)
	require.Nil(t, gotTx)

	// validation success
	tx, err = txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &horizon.Account{
			AccountID: clientKP.Address(),
			Sequence:  "1",
		},
		IncrementSequenceNum: true,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		BaseFee:              300,
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: clientKP.Address(),
				Amount:      "1.0000000",
				Asset:       txnbuild.NativeAsset{},
			},
		},
	})
	require.NoError(t, err)
	txe, err = tx.Base64()
	require.NoError(t, err)

	in.Tx = txe
	txApprovalResp, gotTx = h.validateInput(ctx, in)
	require.Nil(t, txApprovalResp)
	require.Equal(t, gotTx, tx)
}

func TestTxApproveHandler_handleActionRequiredResponseIfNeeded(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	kycThreshold, err := amount.ParseInt64("500")
	require.NoError(t, err)
	h := txApproveHandler{
		assetCode:    "FOO",
		baseURL:      "https://example.com",
		kycThreshold: kycThreshold,
		db:           conn,
	}

	// payments up to the the threshold won't trigger "action_required"
	clientKP := keypair.MustRandom()
	paymentOp := &txnbuild.Payment{
		Amount: amount.StringFromInt64(kycThreshold),
	}
	txApprovalResp, err := h.handleActionRequiredResponseIfNeeded(ctx, clientKP.Address(), paymentOp)
	require.NoError(t, err)
	require.Nil(t, txApprovalResp)

	// payments greater than the threshold will trigger "action_required"
	paymentOp = &txnbuild.Payment{
		Amount: amount.StringFromInt64(kycThreshold + 1),
	}
	txApprovalResp, err = h.handleActionRequiredResponseIfNeeded(ctx, clientKP.Address(), paymentOp)
	require.NoError(t, err)

	var callbackID string
	q := `SELECT callback_id FROM accounts_kyc_status WHERE stellar_address = $1`
	err = conn.QueryRowContext(ctx, q, clientKP.Address()).Scan(&callbackID)
	require.NoError(t, err)

	wantResp := &txApprovalResponse{
		Status:       sep8StatusActionRequired,
		Message:      "Payments exceeding 500.00 FOO require KYC approval. Please provide an email address.",
		ActionMethod: "POST",
		StatusCode:   http.StatusOK,
		ActionURL:    "https://example.com/kyc-status/" + callbackID,
		ActionFields: []string{"email_address"},
	}
	require.Equal(t, wantResp, txApprovalResp)

	// if KYC was previously approved, handleActionRequiredResponseIfNeeded will return nil
	q = `
		UPDATE accounts_kyc_status
		SET 
			approved_at = NOW(),
			rejected_at = NULL
		WHERE stellar_address = $1
	`
	_, err = conn.ExecContext(ctx, q, clientKP.Address())
	require.NoError(t, err)
	txApprovalResp, err = h.handleActionRequiredResponseIfNeeded(ctx, clientKP.Address(), paymentOp)
	require.NoError(t, err)
	require.Nil(t, txApprovalResp)

	// if KYC was previously rejected, handleActionRequiredResponseIfNeeded will return a "rejected" response
	q = `
		UPDATE accounts_kyc_status
		SET 
			approved_at = NULL,
			rejected_at = NOW()
		WHERE stellar_address = $1
	`
	_, err = conn.ExecContext(ctx, q, clientKP.Address())
	require.NoError(t, err)
	txApprovalResp, err = h.handleActionRequiredResponseIfNeeded(ctx, clientKP.Address(), paymentOp)
	require.NoError(t, err)
	require.Equal(t, NewRejectedTxApprovalResponse("Your KYC was rejected and you're not authorized for operations above 500.00 FOO."), txApprovalResp)
}

func TestTxApproveHandler_txApprove_rejected(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	senderKP := keypair.MustRandom()
	receiverKP := keypair.MustRandom()
	issuerKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerKP.Address(),
	}
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)

	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderKP.Address()}).
		Return(horizon.Account{
			AccountID: senderKP.Address(),
			Sequence:  "2",
		}, nil)

	handler := txApproveHandler{
		issuerKP:          issuerKP,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://example.com",
	}

	// "rejected" if tx is empty
	rejectedResponse, err := handler.txApprove(ctx, txApproveRequest{})
	require.NoError(t, err)
	wantRejectedResponse := txApprovalResponse{
		Status:     "rejected",
		Error:      `Missing parameter "tx".`,
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// rejected if the single operation is not a payment
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "2",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.BumpSequence{},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err := tx.Base64()
	require.NoError(t, err)

	txApprovalResp, err := handler.txApprove(ctx, txApproveRequest{Tx: txe})
	require.NoError(t, err)
	wantTxApprovalResp := &txApprovalResponse{
		Status:     "rejected",
		Error:      "There is one or more unauthorized operations in the provided transaction.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, wantTxApprovalResp, txApprovalResp)

	// rejected if payment asset is not supported
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "2",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: receiverKP.Address(),
					Amount:      "1",
					Asset: txnbuild.CreditAsset{
						Code:   "FOO",
						Issuer: keypair.MustRandom().Address(),
					},
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err = tx.Base64()
	require.NoError(t, err)

	txApprovalResp, err = handler.txApprove(ctx, txApproveRequest{Tx: txe})
	require.NoError(t, err)
	wantTxApprovalResp = &txApprovalResponse{
		Status:     "rejected",
		Error:      "The payment asset is not supported by this issuer.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, wantTxApprovalResp, txApprovalResp)

	// rejected if sequence number is not incremental
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "20",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: receiverKP.Address(),
					Amount:      "1",
					Asset:       assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err = tx.Base64()
	require.NoError(t, err)

	txApprovalResp, err = handler.txApprove(ctx, txApproveRequest{Tx: txe})
	require.NoError(t, err)
	wantTxApprovalResp = &txApprovalResponse{
		Status:     "rejected",
		Error:      "Invalid transaction sequence number.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, wantTxApprovalResp, txApprovalResp)
}

func TestTxApproveHandler_txApprove_actionRequired(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	senderKP := keypair.MustRandom()
	receiverKP := keypair.MustRandom()
	issuerKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerKP.Address(),
	}
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)

	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderKP.Address()}).
		Return(horizon.Account{
			AccountID: senderKP.Address(),
			Sequence:  "2",
		}, nil)

	handler := txApproveHandler{
		issuerKP:          issuerKP,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://example.com",
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "2",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: receiverKP.Address(),
					Amount:      "501",
					Asset:       assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err := tx.Base64()
	require.NoError(t, err)

	txApprovalResp, err := handler.txApprove(ctx, txApproveRequest{Tx: txe})
	require.NoError(t, err)

	var callbackID string
	q := `SELECT callback_id FROM accounts_kyc_status WHERE stellar_address = $1`
	err = conn.QueryRowContext(ctx, q, senderKP.Address()).Scan(&callbackID)
	require.NoError(t, err)

	wantResp := &txApprovalResponse{
		Status:       sep8StatusActionRequired,
		Message:      "Payments exceeding 500.00 GOAT require KYC approval. Please provide an email address.",
		ActionMethod: "POST",
		StatusCode:   http.StatusOK,
		ActionURL:    "https://example.com/kyc-status/" + callbackID,
		ActionFields: []string{"email_address"},
	}
	require.Equal(t, wantResp, txApprovalResp)
}

func TestTxApproveHandler_txApprove_revised(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	senderKP := keypair.MustRandom()
	receiverKP := keypair.MustRandom()
	issuerKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerKP.Address(),
	}
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)

	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderKP.Address()}).
		Return(horizon.Account{
			AccountID: senderKP.Address(),
			Sequence:  "2",
		}, nil)

	handler := txApproveHandler{
		issuerKP:          issuerKP,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://example.com",
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "2",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: receiverKP.Address(),
					Amount:      "500",
					Asset:       assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err := tx.Base64()
	require.NoError(t, err)

	txApprovalResp, err := handler.txApprove(ctx, txApproveRequest{Tx: txe})
	require.NoError(t, err)
	require.Equal(t, sep8StatusRevised, txApprovalResp.Status)
	require.Equal(t, http.StatusOK, txApprovalResp.StatusCode)
	require.Equal(t, "Authorization and deauthorization operations were added.", txApprovalResp.Message)

	gotGenericTx, err := txnbuild.TransactionFromXDR(txApprovalResp.Tx)
	require.NoError(t, err)
	gotTx, ok := gotGenericTx.Transaction()
	require.True(t, ok)
	require.Equal(t, senderKP.Address(), gotTx.SourceAccount().AccountID)
	require.Equal(t, int64(3), gotTx.SourceAccount().Sequence)

	require.Len(t, gotTx.Operations(), 5)
	// AllowTrust op where issuer fully authorizes sender, asset GOAT
	op0, ok := gotTx.Operations()[0].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op0.Trustor, senderKP.Address())
	assert.Equal(t, op0.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op0.Authorize)
	// AllowTrust op where issuer fully authorizes receiver, asset GOAT
	op1, ok := gotTx.Operations()[1].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op1.Trustor, receiverKP.Address())
	assert.Equal(t, op1.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op1.Authorize)
	// Payment from sender to receiver
	op2, ok := gotTx.Operations()[2].(*txnbuild.Payment)
	require.True(t, ok)
	assert.Equal(t, op2.Destination, receiverKP.Address())
	assert.Equal(t, op2.Asset, assetGOAT)
	// AllowTrust op where issuer fully deauthorizes receiver, asset GOAT
	op3, ok := gotTx.Operations()[3].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op3.Trustor, receiverKP.Address())
	assert.Equal(t, op3.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op3.Authorize)
	// AllowTrust op where issuer fully deauthorizes sender, asset GOAT
	op4, ok := gotTx.Operations()[4].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op4.Trustor, senderKP.Address())
	assert.Equal(t, op4.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op4.Authorize)
}

func TestConvertAmountToReadableString(t *testing.T) {
	parsedAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	assert.Equal(t, int64(5000000000), parsedAmount)

	readableAmount, err := convertAmountToReadableString(parsedAmount)
	require.NoError(t, err)
	assert.Equal(t, "500.00", readableAmount)
}
