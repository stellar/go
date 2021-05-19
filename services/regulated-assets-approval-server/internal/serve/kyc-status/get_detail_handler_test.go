package kycstatus

import (
	"database/sql"
	"testing"
	"time"

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
	h = GetDetailHandler{
		DB: conn,
	}
	err = h.validate()
	require.NoError(t, err)
}

func TestGetDetailHandlerTimePointerIfValid(t *testing.T) {
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
	// TEST if timePointer is valid; timePointerIfValid will return the time.Time ptr.
	timePointer = timePointerIfValid(nullTimePtr)
	require.NotNil(t, timePointer)
	assert.Equal(t, &timeNow, timePointer)
}
