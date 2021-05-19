package kycstatus

import (
	"testing"

	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
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
	h = GetDetailHandler{
		DB: conn,
	}
	err = h.validate()
	require.NoError(t, err)
}
