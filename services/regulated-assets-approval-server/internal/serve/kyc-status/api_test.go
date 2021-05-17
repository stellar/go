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
		EmailAddress: "xxtest@email.com",
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
		EmailAddress: "xxtest@email.com",
	}
	query, args = in.buildUpdateKYCQuery()
	expectedQuery = "WITH updated_row AS (UPDATE accounts_kyc_status SET kyc_submitted_at = NOW(), email_address = $1, rejected_at = NOW(), approved_at = NULL WHERE callback_id = $2 RETURNING * )\n\t\tSELECT EXISTS(\n\t\t\tSELECT * FROM updated_row\n\t\t)\n\t"
	expectedArgs[0] = in.EmailAddress
	expectedArgs[1] = in.CallbackID
	require.Equal(t, expectedQuery, query)
	require.Equal(t, expectedArgs, args)
}

func TestAPI_POSTKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// INSERT new account in accounts_kyc_status that needs kyc verified.
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
	// create kyc-status PostHandler.
	postHandler := PostHandler{
		DB: conn,
	}
	err := postHandler.DB.QueryRowContext(ctx, q, approveKP.Address(), intendedCallbackIDApprove).Scan(&callbackID)
	require.NoError(t, err)
	assert.Equal(t, intendedCallbackIDApprove, callbackID)
	// Test POST successful APPROVED KYC response.
	m := chi.NewMux()
	m.Post("/kyc-status/{callback_id}", postHandler.ServeHTTP)
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
	var kycStatusPOSTResponseApprove kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseApprove)
	require.NoError(t, err)
	wantPostResponse := kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseApprove)
	// Test POST successful REJECTED KYC response. Based on arbitrary rule where emails begin with "xx".
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
	var kycStatusPOSTResponseRejected kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejected)
	require.NoError(t, err)
	wantPostResponse = kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejected)
	// Test repeated KYC request after REJECTED w/ new email. Should succeed as approved.
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
	var kycStatusPOSTResponseRejectedNewEmail kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejectedNewEmail)
	require.NoError(t, err)
	wantPostResponse = kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejectedNewEmail)
	// Test POST no email in request.
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
	// Test "Not Found", callback_id not registered.
	callbackIDNotFound := uuid.New().String()
	reqBody = `{
		"email_address": "notFound@email.com"
	}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackIDNotFound), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantPostResponseNotFound := `{
		"error": "Not found."
	}`
	require.JSONEq(t, wantPostResponseNotFound, string(body))
}
