package kycstatus

import (
	"testing"

	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
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

func TestIsKYCRuleRespected(t *testing.T) {
	// Test if email approved.
	in := kycPostRequest{
		EmailAddress: "test@email.com",
	}
	approved := in.isKYCRuleRespected()
	assert.True(t, approved)
	// Test if email approved rejected.
	in = kycPostRequest{
		EmailAddress: "xtest@email.com",
	}
	approved = in.isKYCRuleRespected()
	assert.False(t, approved)
}

func TestBuildUpdateKYCQuery(t *testing.T) {
	// Test query returned if email approved.
	in := kycPostRequest{
		CallbackID:   "1234567890-12345",
		EmailAddress: "test@email.com",
	}
	query, args := in.buildUpdateKYCQuery()
	expectedQuery := "WITH updated_row AS (UPDATE accounts_kyc_status SET kyc_submitted_at = NOW(), email_address = $1, approved_at = NOW(), rejected_at = NULL WHERE callback_id = $2 RETURNING * )\n\t\tSELECT EXISTS(\n\t\t\tSELECT * FROM updated_row\n\t\t)\n\t"
	var expectedArgs []interface{}
	expectedArgs = append(expectedArgs, in.EmailAddress, in.CallbackID)
	require.Equal(t, expectedQuery, query)
	require.Equal(t, expectedArgs, args)
	// Test query returned if email rejected.
	in = kycPostRequest{
		CallbackID:   "9999999999-9999",
		EmailAddress: "xtest@email.com",
	}
	query, args = in.buildUpdateKYCQuery()
	expectedQuery = "WITH updated_row AS (UPDATE accounts_kyc_status SET kyc_submitted_at = NOW(), email_address = $1, rejected_at = NOW(), approved_at = NULL WHERE callback_id = $2 RETURNING * )\n\t\tSELECT EXISTS(\n\t\t\tSELECT * FROM updated_row\n\t\t)\n\t"
	expectedArgs[0] = in.EmailAddress
	expectedArgs[1] = in.CallbackID
	require.Equal(t, expectedQuery, query)
	require.Equal(t, expectedArgs, args)
}
