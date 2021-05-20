package kycstatus

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDetailHandlerValidate(t *testing.T) {
	// Test no db.
	h := GetDetailHandler{}
	err := h.validate()
	require.EqualError(t, err, "database cannot be nil")
	// Success.
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	h = GetDetailHandler{DB: conn}
	err = h.validate()
	require.NoError(t, err)
}

func TestTimePointerIfValid(t *testing.T) {
	// Prepare NULL nullTimePtr.
	var nullTimePtr sql.NullTime

	// Send a NullTime Pointer to timePointerIfValid.
	// TEST if timePointer is null; timePointerIfValid will return nil in this case.
	timePointer := timePointerIfValid(nullTimePtr)
	require.Nil(t, timePointer)

	// Prepare a valid nullTimePtr with a time set.
	nullTimePtr.Valid = true
	timeNow := time.Now()
	nullTimePtr.Time = timeNow

	// Send a valid Pointer to timePointerIfValid.
	// TEST if timePointer is valid and if return a time.Time pointer equals the time.Now().
	timePointer = timePointerIfValid(nullTimePtr)
	require.NotNil(t, timePointer)
	assert.Equal(t, &timeNow, timePointer)
}

func TestGetDetailHandlerHandle(t *testing.T) {
	ctx := context.Background()

	// Prepare and validate GetDetailHandler.
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	h := GetDetailHandler{DB: conn}
	err := h.validate()
	require.NoError(t, err)

	// Prepare and send empty getDetailRequest.
	in := getDetailRequest{}
	kycGetResp, err := h.handle(ctx, in)

	// TEST error "missing stellar address or callback_id"
	require.Nil(t, kycGetResp)
	require.EqualError(t, err, "missing stellar address or callback_id")

	// Prepare and send getDetailRequest to an account not in the db.
	accountKP := keypair.MustRandom()
	in = getDetailRequest{StellarAddressOrCallbackID: accountKP.Address()}
	kycGetResp, err = h.handle(ctx, in)

	// TEST error "not found".
	require.Nil(t, kycGetResp)
	require.EqualError(t, err, "not found")

	// Prepare and send getDetailRequest to an callbackID not in the db.
	callbackID := uuid.New().String()
	in = getDetailRequest{StellarAddressOrCallbackID: callbackID}
	kycGetResp, err = h.handle(ctx, in)

	// TEST error "not found".
	require.Nil(t, kycGetResp)
	require.EqualError(t, err, "not found")

	// INSERT new account in db's accounts_kyc_status table; new account was approved after submitting kyc.
	insertNewAccountQuery := `
	INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
	VALUES ($1, $2, $3, NOW(), NOW(), NULL)
	`
	emailAddress := "email@approved.com"
	_, err = h.DB.ExecContext(ctx, insertNewAccountQuery, accountKP.Address(), callbackID, emailAddress)
	require.NoError(t, err)

	// Prepare and execute SELECT query for account was added. Ensures account been added.
	existQuery := `
		SELECT EXISTS(
			SELECT stellar_address
			FROM accounts_kyc_status
			WHERE stellar_address = $1
		)`
	var exists bool
	err = h.DB.QueryRowContext(ctx, existQuery, accountKP.Address()).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)

	// Send getDetailRequest to an account in the db.
	kycGetResp, err = h.handle(ctx, in)

	// TEST if response returns with account that was inserted in db.
	require.NoError(t, err)
	wantKycGetResponse := kycGetResponse{
		StellarAddress: accountKP.Address(),
		CallbackID:     callbackID,
		EmailAddress:   emailAddress,
		CreatedAt:      kycGetResp.CreatedAt,
		KYCSubmittedAt: kycGetResp.KYCSubmittedAt,
		ApprovedAt:     kycGetResp.ApprovedAt,
		RejectedAt:     kycGetResp.RejectedAt,
	}
	assert.Equal(t, &wantKycGetResponse, kycGetResp)

	// TEST if response timestamps are present or null.
	require.NotNil(t, kycGetResp.CreatedAt)
	require.NotNil(t, kycGetResp.KYCSubmittedAt)
	require.NotNil(t, kycGetResp.ApprovedAt)
	require.Nil(t, kycGetResp.RejectedAt)
}
