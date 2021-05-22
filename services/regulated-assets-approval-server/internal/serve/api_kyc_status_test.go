package serve

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
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	kycstatus "github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/kyc-status"
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
	postHandler := kycstatus.PostHandler{DB: conn}

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
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "no_further_action_required" response for approved account.
	type kycStatusPOSTResponse struct {
		Result string `json:"result"`
	}
	var kycStatusPOSTResponseApprove kycStatusPOSTResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseApprove)
	require.NoError(t, err)
	wantPostResponse := kycStatusPOSTResponse{
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
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "no_further_action_required" response for rejected account.
	var kycStatusPOSTResponseRejected kycStatusPOSTResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejected)
	require.NoError(t, err)
	wantPostResponse = kycStatusPOSTResponse{
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
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "no_further_action_required" response for repeated KYC request after REJECTED w/ new email.
	var kycStatusPOSTResponseRejectedNewEmail kycStatusPOSTResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejectedNewEmail)
	require.NoError(t, err)
	wantPostResponse = kycStatusPOSTResponse{
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
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
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
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "Not found." error response.
	wantPostResponseNotFound := `{
		"error": "Not found."
	}`
	require.JSONEq(t, wantPostResponseNotFound, string(body))
}

func TestAPI_GETKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// Create kyc-status GetDetailHandler.
	getHandler := kycstatus.GetDetailHandler{DB: conn}

	// INSERT new account in db's accounts_kyc_status table; new account was approved after submitting kyc.
	insertNewApprovedAccountQuery := `
	INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
	VALUES ($1, $2, $3, NOW(), NOW(), NULL)
	`
	approveKP := keypair.MustRandom()
	approveCallbackID := uuid.New().String()
	approveEmailAddress := "email@approved.com"
	_, err := getHandler.DB.ExecContext(ctx, insertNewApprovedAccountQuery, approveKP.Address(), approveCallbackID, approveEmailAddress)
	require.NoError(t, err)

	// Prepare and send /kyc-status/{stellar_address_or_callback_id} GET request; for approved account in the accounts_kyc_status table.
	m := chi.NewMux()
	m.Get("/kyc-status/{stellar_address_or_callback_id}", getHandler.ServeHTTP)
	r := httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", approveKP.Address()), nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST kycGetResponse response for approved account.
	type kycGETResponse struct {
		StellarAddress string     `json:"stellar_address"`
		CallbackID     string     `json:"callback_id"`
		EmailAddress   string     `json:"email_address,omitempty"`
		CreatedAt      *time.Time `json:"created_at"`
		KYCSubmittedAt *time.Time `json:"kyc_submitted_at,omitempty"`
		ApprovedAt     *time.Time `json:"approved_at,omitempty"`
		RejectedAt     *time.Time `json:"rejected_at,omitempty"`
	}
	var kycStatusGETResponseApprove kycGETResponse
	err = json.Unmarshal(body, &kycStatusGETResponseApprove)
	require.NoError(t, err)
	wantPostResponse := kycGETResponse{
		StellarAddress: approveKP.Address(),
		CallbackID:     approveCallbackID,
		EmailAddress:   approveEmailAddress,
		CreatedAt:      kycStatusGETResponseApprove.CreatedAt,
		KYCSubmittedAt: kycStatusGETResponseApprove.KYCSubmittedAt,
		ApprovedAt:     kycStatusGETResponseApprove.ApprovedAt,
		RejectedAt:     kycStatusGETResponseApprove.RejectedAt,
	}
	assert.Equal(t, wantPostResponse, kycStatusGETResponseApprove)

	// TEST if response timestamps are present or null.
	require.NotNil(t, kycStatusGETResponseApprove.CreatedAt)
	require.NotNil(t, kycStatusGETResponseApprove.KYCSubmittedAt)
	require.NotNil(t, kycStatusGETResponseApprove.ApprovedAt)
	require.Nil(t, kycStatusGETResponseApprove.RejectedAt)

	// INSERT new account in db's accounts_kyc_status table; new account was rejected after submiting kyc.
	insertNewRejectedAccountQuery := `
	INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
	VALUES ($1, $2, $3, NOW(), NULL, NOW())
	`
	rejectKP := keypair.MustRandom()
	rejectCallbackID := uuid.New().String()
	rejectEmailAddress := "xemail@rejected.com"
	_, err = getHandler.DB.ExecContext(ctx, insertNewRejectedAccountQuery, rejectKP.Address(), rejectCallbackID, rejectEmailAddress)
	require.NoError(t, err)

	// Prepare and send /kyc-status/{stellar_address_or_callback_id} GET request; for rejected account in the accounts_kyc_status table.
	r = httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", rejectKP.Address()), nil)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST kycGetResponse response for rejected account.
	var kycStatusGETResponseReject kycGETResponse
	err = json.Unmarshal(body, &kycStatusGETResponseReject)
	require.NoError(t, err)
	wantPostResponse = kycGETResponse{
		StellarAddress: rejectKP.Address(),
		CallbackID:     rejectCallbackID,
		EmailAddress:   rejectEmailAddress,
		CreatedAt:      kycStatusGETResponseReject.CreatedAt,
		KYCSubmittedAt: kycStatusGETResponseReject.KYCSubmittedAt,
		ApprovedAt:     kycStatusGETResponseReject.ApprovedAt,
		RejectedAt:     kycStatusGETResponseReject.RejectedAt,
	}
	assert.Equal(t, wantPostResponse, kycStatusGETResponseReject)

	// TEST if response timestamps are present or null.
	require.NotNil(t, kycStatusGETResponseReject.CreatedAt)
	require.NotNil(t, kycStatusGETResponseReject.KYCSubmittedAt)
	require.NotNil(t, kycStatusGETResponseReject.RejectedAt)
	require.Nil(t, kycStatusGETResponseReject.ApprovedAt)

	// INSERT new account in db's accounts_kyc_status table; new account hasn't submitted kyc.
	insertNewAccountNoKycQuery := `
		INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
		VALUES ($1, $2, NULL, NULL, NULL, NULL)
	`
	noKycAccountKP := keypair.MustRandom()
	noKycCallbackID := uuid.New().String()
	_, err = getHandler.DB.ExecContext(ctx, insertNewAccountNoKycQuery, noKycAccountKP.Address(), noKycCallbackID)
	require.NoError(t, err)

	// Prepare and send /kyc-status/{stellar_address_or_callback_id} GET request; for no KYC account in the accounts_kyc_status table (this time using their callbackID).
	r = httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", noKycCallbackID), nil)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST kycGetResponse response for account that hasn't submitted kyc.
	var kycStatusGETResponseNoKyc kycGETResponse
	err = json.Unmarshal(body, &kycStatusGETResponseNoKyc)
	require.NoError(t, err)
	wantPostResponse = kycGETResponse{
		StellarAddress: noKycAccountKP.Address(),
		CallbackID:     noKycCallbackID,
		EmailAddress:   "",
		CreatedAt:      kycStatusGETResponseNoKyc.CreatedAt,
		KYCSubmittedAt: kycStatusGETResponseNoKyc.KYCSubmittedAt,
		ApprovedAt:     kycStatusGETResponseNoKyc.ApprovedAt,
		RejectedAt:     kycStatusGETResponseNoKyc.RejectedAt,
	}
	assert.Equal(t, wantPostResponse, kycStatusGETResponseNoKyc)

	// TEST if response timestamps are present or null.
	require.NotNil(t, kycStatusGETResponseNoKyc.CreatedAt)
	require.Nil(t, kycStatusGETResponseNoKyc.KYCSubmittedAt)
	require.Nil(t, kycStatusGETResponseNoKyc.RejectedAt)
	require.Nil(t, kycStatusGETResponseNoKyc.ApprovedAt)

	// Prepare and send /kyc-status/{stellar_address_or_callback_id} GET request; for an account that isn't in the accounts_kyc_status table.
	notPresentKP := keypair.MustRandom()
	r = httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", notPresentKP.Address()), nil)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST "Not found." error response.
	wantGetResponseNotFound := `{
		"error": "Not found."
	}`
	require.JSONEq(t, wantGetResponseNotFound, string(body))
}

func TestAPI_DELETEKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// Create kyc-status DeleteHandler.
	deleteHandler := kycstatus.DeleteHandler{DB: conn}

	// INSERT new account in db's accounts_kyc_status table; new account was approved after submitting kyc.
	insertNewApprovedAccountQuery := `
	INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
	VALUES ($1, $2, $3, NOW(), NOW(), NULL)
	`
	approveKP := keypair.MustRandom()
	approveCallbackID := uuid.New().String()
	approveEmailAddress := "email@approved.com"
	_, err := deleteHandler.DB.ExecContext(ctx, insertNewApprovedAccountQuery, approveKP.Address(), approveCallbackID, approveEmailAddress)
	require.NoError(t, err)

	// Prepare and send /kyc-status/{stellar_address} DELETE request; for approved account in the accounts_kyc_status table.
	m := chi.NewMux()
	m.Delete("/kyc-status/{stellar_address}", deleteHandler.ServeHTTP)
	r := httptest.NewRequest("DELETE", fmt.Sprintf("/kyc-status/%s", approveKP.Address()), nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST ok response for deleteing account's kyc record from db's accounts_kyc_status table.
	wantBody := `{
		"message":"ok"
	}`
	require.JSONEq(t, wantBody, string(body))

	// Prepare and execute SELECT query for account that was deleted, TEST if the the account doesn't exist in the db.
	existQuery := `
	SELECT EXISTS(
		SELECT stellar_address
		FROM accounts_kyc_status
		WHERE stellar_address = $1
	)`
	var exists bool
	err = deleteHandler.DB.QueryRowContext(ctx, existQuery, approveKP.Address()).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)

	// Prepare and send /kyc-status/{stellar_address} DELETE request; for account that isn't in the accounts_kyc_status table.
	r = httptest.NewRequest("DELETE", fmt.Sprintf("/kyc-status/%s", approveKP.Address()), nil)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	// TEST error "Not found." response for attempting to delete an account that isn't in the accounts_kyc_status table.
	wantDeleteResponseNotFound := `{
		"error": "Not found."
	}`
	require.JSONEq(t, wantDeleteResponseNotFound, string(body))
}
