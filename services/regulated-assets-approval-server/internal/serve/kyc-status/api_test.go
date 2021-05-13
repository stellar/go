package kycstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_POSTKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// Setup /kyc-status route
	m := chi.NewMux()
	postHandler := PostHandler{
		DB: conn,
	}
	m.Route("/kyc-status", func(mux chi.Router) {
		mux.Post("/{callback_id}", postHandler.ServeHTTP)
	})
	// INSERT new account in accounts_kyc_status that needs kyc verified
	const q = `
		WITH new_row AS (
			INSERT INTO accounts_kyc_status (stellar_address, callback_id)
			VALUES ($1, $2)
			ON CONFLICT(stellar_address) DO NOTHING
			RETURNING *
		)
		SELECT callback_id FROM new_row
		UNION
		SELECT callback_id
		FROM accounts_kyc_status
		WHERE stellar_address = $1
	`
	sourceKP := keypair.MustRandom()
	intendedCallbackID := uuid.New().String()
	var (
		callbackID string
	)
	err := postHandler.DB.QueryRowContext(ctx, q, sourceKP.Address(), intendedCallbackID).Scan(&callbackID)
	require.NoError(t, err)
	assert.Equal(t, intendedCallbackID, callbackID)
	// Test POST successful.
	reqBody := `{
		"email_address": "TestEmail@email.com"
	}`
	r := httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackID), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var kycStatusPOSTResponse postResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponse)
	require.NoError(t, err)
	wantPostResponse := postResponse{
		Result:  "no_further_action_required",
		Message: "Your KYC has been approved!",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponse)
}
