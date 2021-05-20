package kycstatus

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteHandlerValidate(t *testing.T) {
	// Test no db.
	h := DeleteHandler{}
	err := h.validate()
	require.EqualError(t, err, "database cannot be nil")
	// Success.
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	h = DeleteHandler{DB: conn}
	err = h.validate()
	require.NoError(t, err)
}

func TestDeleteHandlerHandle(t *testing.T) {
	ctx := context.Background()

	// Prepare and validate DeleteHandler.
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	h := DeleteHandler{DB: conn}
	err := h.validate()
	require.NoError(t, err)

	// Prepare and send empty deleteRequest.
	in := deleteRequest{}
	err = h.handle(ctx, in)

	// TEST error "missing stellar address"
	require.EqualError(t, err, "missing stellar address")

	// Prepare and send deleteRequest to an account not in the db.
	accountKP := keypair.MustRandom()
	in = deleteRequest{StellarAddress: accountKP.Address()}
	err = h.handle(ctx, in)

	// TEST error "not found".
	require.EqualError(t, err, "not found")

	// INSERT new account in db's accounts_kyc_status table; new account was approved after submitting kyc.
	insertNewAccountQuery := `
	INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
	VALUES ($1, $2, $3, NOW(), NOW(), NULL)
	`
	callbackID := uuid.New().String()
	emailAddress := "email@approved.com"
	_, err = h.DB.ExecContext(ctx, insertNewAccountQuery, accountKP.Address(), callbackID, emailAddress)
	require.NoError(t, err)

	// Prepare and execute SELECT query for account was added, ensures its in db.
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

	// Send deleteRequest to an account in the db.
	err = h.handle(ctx, in)

	// TEST if nil error returned (success).
	require.NoError(t, err)

	// Execute SELECT query for account that was deleted; ensure its no longer in db
	err = h.DB.QueryRowContext(ctx, existQuery, accountKP.Address()).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)
}
