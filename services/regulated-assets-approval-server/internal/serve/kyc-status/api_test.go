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
	approveKP := keypair.MustRandom()
	intendedCallbackIDApprove := uuid.New().String()
	var (
		callbackID string
	)
	err := postHandler.DB.QueryRowContext(ctx, q, approveKP.Address(), intendedCallbackIDApprove).Scan(&callbackID)
	require.NoError(t, err)
	assert.Equal(t, intendedCallbackIDApprove, callbackID)
	// Test POST successful APPROVED KYC response.
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
	var kycStatusPOSTResponseApprove postResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseApprove)
	require.NoError(t, err)
	wantPostResponse := postResponse{
		Result:  "no_further_action_required",
		Message: "Your KYC has been approved!",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseApprove)
	// Test repeated KYC request after approval.
	// ?: Is this the response we want?
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackID), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var kycStatusPOSTResponseApproveRepeatApprove postResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseApproveRepeatApprove)
	require.NoError(t, err)
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseApprove)
	// Test POST successful REJECTED KYC response. Based on arbitrary rule where emails begin with "xx"
	rejectedKP := keypair.MustRandom()
	intendedCallbackIDRejected := uuid.New().String()
	err = postHandler.DB.QueryRowContext(ctx, q, rejectedKP.Address(), intendedCallbackIDRejected).Scan(&callbackID)
	require.NoError(t, err)
	assert.Equal(t, intendedCallbackIDRejected, callbackID)
	reqBody = `{
		"email_address": "xxTestEmail@email.com"
	}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackID), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var kycStatusPOSTResponseRejected postResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejected)
	require.NoError(t, err)
	wantPostResponse = postResponse{
		Message: "Your KYC has been rejected!",
		Result:  "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejected)
	// Test repeated KYC request after approval w/ new email.
	// Should succeed as approved.
	reqBody = `{
		"email_address": "TestEmailxx@email.com"
	}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackID), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var kycStatusPOSTResponseRejectedNewEmail postResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejectedNewEmail)
	require.NoError(t, err)
	wantPostResponse = postResponse{
		Message: "Your KYC has been approved!",
		Result:  "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejectedNewEmail)
	// Test POST no email in request
	noEmailKP := keypair.MustRandom()
	intendedCallbackIDNoEmail := uuid.New().String()
	err = postHandler.DB.QueryRowContext(ctx, q, noEmailKP.Address(), intendedCallbackIDNoEmail).Scan(&callbackID)
	require.NoError(t, err)
	assert.Equal(t, intendedCallbackIDNoEmail, callbackID)
	reqBody = `{
		"email_address": ""
	}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackID), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantPostResponseMissingEmail := `{
		"error": "Missing email_address."
	}`
	require.JSONEq(t, wantPostResponseMissingEmail, string(body))
}
