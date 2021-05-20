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

	// Prepare and send empty deleteRequest. TEST error "Missing stellar address."
	in := deleteRequest{}
	err = h.handle(ctx, in)
	require.EqualError(t, err, "Missing stellar address.")

	// Prepare and send deleteRequest to an account not in the db. TEST error "Not found.".
	accountKP := keypair.MustRandom()
	in = deleteRequest{StellarAddress: accountKP.Address()}
	err = h.handle(ctx, in)
	require.EqualError(t, err, "Not found.")

	// INSERT new account in db's accounts_kyc_status table; new account was approved after submitting kyc.
	insertNewAccountQuery := `
	INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
	VALUES ($1, $2, $3, NOW(), NOW(), NULL)
	`
	callbackID := uuid.New().String()
	emailAddress := "email@approved.com"
	_, err = h.DB.ExecContext(ctx, insertNewAccountQuery, accountKP.Address(), callbackID, emailAddress)
	require.NoError(t, err)

	// Send deleteRequest to an account in the db. TEST if nil error returned (success).
	err = h.handle(ctx, in)
	require.NoError(t, err)

	// Prepare and execute SELECT query for account that was deleted; ensure its no longer in db
	existQuery := `
		SELECT EXISTS(
			SELECT stellar_address
			FROM accounts_kyc_status
			WHERE stellar_address = $1
		)`
	var exists bool
	err = h.DB.QueryRowContext(ctx, existQuery, accountKP.Address()).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)
}
