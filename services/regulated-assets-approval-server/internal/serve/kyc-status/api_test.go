package kycstatus

import (
	"context"
	"database/sql"
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

	// Create kyc-status PostHandler.
	postHandler := PostHandler{
		DB: conn,
	}

	// INSERT new unverified account in db's accounts_kyc_status table.
	const insertNewAccountQuery = `
	INSERT INTO accounts_kyc_status (stellar_address, callback_id)
	VALUES ($1, $2)
	`
	approveKP := keypair.MustRandom()
	callbackID := uuid.New().String()
	_, err := postHandler.DB.ExecContext(ctx, insertNewAccountQuery, approveKP.Address(), callbackID)
	require.NoError(t, err)

	// Preparing and send /kyc-status/{callback_id} POST request.
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

	// TEST "no_further_action_required" response for approved account.
	var kycStatusPOSTResponseApprove kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseApprove)
	require.NoError(t, err)
	wantPostResponse := kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseApprove)

	// Query db's accounts_kyc_status table account after /kyc-status/{callback_id} POST request.
	const selectAccountQuery = `
	SELECT approved_at, rejected_at
	FROM accounts_kyc_status
	WHERE callback_id = $1
	`
	var approvedAt, rejectedAt sql.NullTime
	err = postHandler.DB.QueryRowContext(ctx, selectAccountQuery, callbackID).Scan(&approvedAt, &rejectedAt)
	require.NoError(t, err)

	// TEST if account in db's accounts_kyc_status table was approved.
	// sql.NullTime.Valid is true if Time is not NULL
	assert.True(t, approvedAt.Valid)
	assert.False(t, rejectedAt.Valid)

	// INSERT new unverified account in db's accounts_kyc_status table.
	rejectedKP := keypair.MustRandom()
	callbackIDRejected := uuid.New().String()
	_, err = postHandler.DB.ExecContext(ctx, insertNewAccountQuery, rejectedKP.Address(), callbackIDRejected)
	require.NoError(t, err)

	// Preparing and send /kyc-status/{callback_id} POST request.
	reqBody = `{
		"email_address": "xTestEmail@email.com"
		}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackIDRejected), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "no_further_action_required" response for rejected account.
	var kycStatusPOSTResponseRejected kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejected)
	require.NoError(t, err)
	wantPostResponse = kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejected)

	// Query db's accounts_kyc_status table account after /kyc-status/{callback_id} POST request.
	err = postHandler.DB.QueryRowContext(ctx, selectAccountQuery, callbackIDRejected).Scan(&approvedAt, &rejectedAt)
	require.NoError(t, err)

	// TEST if account in db's accounts_kyc_status table was rejected.
	// Should be rejected based on arbitrary rule where emails begin with "x".
	// sql.NullTime.Valid is true if Time is not NULL
	assert.True(t, rejectedAt.Valid)
	assert.False(t, approvedAt.Valid)

	// Preparing and send /kyc-status/{callback_id} POST request; using the rejected account's callback_ID.
	reqBody = `{
		"email_address": "TestEmailx@email.com"
		}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackIDRejected), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "no_further_action_required" response for repeated KYC request after REJECTED w/ new email.
	var kycStatusPOSTResponseRejectedNewEmail kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejectedNewEmail)
	require.NoError(t, err)
	wantPostResponse = kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejectedNewEmail)

	// Query db's accounts_kyc_status table account after /kyc-status/{callback_id} POST request.
	selectUpdatedAccountEmailQuery := `
	SELECT approved_at, rejected_at, email_address
	FROM accounts_kyc_status
	WHERE callback_id = $1
	`
	var updatedEmail string
	err = postHandler.DB.QueryRowContext(ctx, selectUpdatedAccountEmailQuery, callbackIDRejected).Scan(&approvedAt, &rejectedAt, &updatedEmail)
	require.NoError(t, err)

	// TEST if account in db's accounts_kyc_status table was approved, and email was overwritten.
	// sql.NullTime.Valid is true if Time is not NULL
	assert.True(t, approvedAt.Valid)
	assert.False(t, rejectedAt.Valid)
	assert.NotEqual(t, "xTestEmail@email.com", updatedEmail)

	// Preparing and send /kyc-status/{callback_id} POST request; w/ empty email value.
	noEmailKP := keypair.MustRandom()
	callbackIDNoEmail := uuid.New().String()
	_, err = postHandler.DB.ExecContext(ctx, insertNewAccountQuery, noEmailKP.Address(), callbackIDNoEmail)
	require.NoError(t, err)
	reqBody = `{
		"email_address": ""
		}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackIDNoEmail), strings.NewReader(reqBody))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "Missing email_address" error response.
	wantPostResponseMissingEmail := `{
		"error": "Missing email_address."
	}`
	require.JSONEq(t, wantPostResponseMissingEmail, string(body))

	// Preparing and send /kyc-status/{callback_id} POST request; callback_id not registered.
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

	// TEST "Not Found" error response.
	wantPostResponseNotFound := `{
			"error": "Not found."
			}`
	require.JSONEq(t, wantPostResponseNotFound, string(body))
}
