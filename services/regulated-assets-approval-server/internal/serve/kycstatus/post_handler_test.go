package kycstatus

import (
	"testing"

	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostHandlerValidate(t *testing.T) {
	// Test no db.
	h := PostHandler{}
	err := h.validate()
	require.EqualError(t, err, "database cannot be nil")
	// Success.
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	h = PostHandler{
		DB: conn,
	}
	err = h.validate()
	require.NoError(t, err)
}

func TestIsKYCRejected(t *testing.T) {
	in := kycPostRequest{
		EmailAddress: "test@email.com",
	}
	isRejected := in.isKYCRejected()
	assert.False(t, isRejected)

	// emails starting with "x" should be rejected
	in = kycPostRequest{
		EmailAddress: "xtest@email.com",
	}
	isRejected = in.isKYCRejected()
	assert.True(t, isRejected)
}

func TestIsKYCPending(t *testing.T) {
	in := kycPostRequest{
		EmailAddress: "test@email.com",
	}
	isPending := in.isKYCPending()
	assert.False(t, isPending)

	// emails starting with "y" should be marked as pending
	in = kycPostRequest{
		EmailAddress: "ytest@email.com",
	}
	isPending = in.isKYCPending()
	assert.True(t, isPending)
}

func TestBuildUpdateKYCQuery(t *testing.T) {
	// test rejected query
	in := kycPostRequest{
		CallbackID:   "9999999999-9999",
		EmailAddress: "xtest@email.com",
	}
	query, args := in.buildUpdateKYCQuery()
	expectedQuery := "WITH updated_row AS (UPDATE accounts_kyc_status SET kyc_submitted_at = NOW(), email_address = $1, rejected_at = NOW(), pending_at = NULL, approved_at = NULL WHERE callback_id = $2 RETURNING * )\n\t\tSELECT EXISTS(\n\t\t\tSELECT * FROM updated_row\n\t\t)\n\t"
	expectedArgs := []interface{}{in.EmailAddress, in.CallbackID}
	require.Equal(t, expectedQuery, query)
	require.Equal(t, expectedArgs, args)

	// test pending query
	in = kycPostRequest{
		CallbackID:   "1234567890-12345",
		EmailAddress: "ytest@email.com",
	}
	query, args = in.buildUpdateKYCQuery()
	expectedQuery = "WITH updated_row AS (UPDATE accounts_kyc_status SET kyc_submitted_at = NOW(), email_address = $1, rejected_at = NULL, pending_at = NOW(), approved_at = NULL WHERE callback_id = $2 RETURNING * )\n\t\tSELECT EXISTS(\n\t\t\tSELECT * FROM updated_row\n\t\t)\n\t"
	expectedArgs = []interface{}{in.EmailAddress, in.CallbackID}
	require.Equal(t, expectedQuery, query)
	require.Equal(t, expectedArgs, args)

	// test approved query
	in = kycPostRequest{
		CallbackID:   "1234567890-12345",
		EmailAddress: "test@email.com",
	}
	query, args = in.buildUpdateKYCQuery()
	expectedQuery = "WITH updated_row AS (UPDATE accounts_kyc_status SET kyc_submitted_at = NOW(), email_address = $1, rejected_at = NULL, pending_at = NULL, approved_at = NOW() WHERE callback_id = $2 RETURNING * )\n\t\tSELECT EXISTS(\n\t\t\tSELECT * FROM updated_row\n\t\t)\n\t"
	expectedArgs = []interface{}{in.EmailAddress, in.CallbackID}
	require.Equal(t, expectedQuery, query)
	require.Equal(t, expectedArgs, args)
}

func TestRxEmail(t *testing.T) {
	// Test empty email string.
	assert.NotRegexp(t, RxEmail, "")

	// Test empty prefix.
	assert.NotRegexp(t, RxEmail, "email.com")

	// Test only domain given.
	assert.NotRegexp(t, RxEmail, "@email.com")

	// Test correct email.
	assert.Regexp(t, RxEmail, "t@email.com")
}
