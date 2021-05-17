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

	// INSERT new account in accounts_kyc_status that needs kyc verified.
	const q = `
		INSERT INTO accounts_kyc_status (stellar_address, callback_id)
		VALUES ($1, $2)
	`
	approveKP := keypair.MustRandom()
	callbackID := uuid.New().String()
	// create kyc-status PostHandler.
	postHandler := PostHandler{
		DB: conn,
	}
	_, err := postHandler.DB.ExecContext(ctx, q, approveKP.Address(), callbackID)
	require.NoError(t, err)
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
	// Test POST successful REJECTED KYC response. Based on arbitrary rule where emails begin with "x".
	rejectedKP := keypair.MustRandom()
	callbackIDRejected := uuid.New().String()
	_, err = postHandler.DB.ExecContext(ctx, q, rejectedKP.Address(), callbackIDRejected)
	require.NoError(t, err)
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
	var kycStatusPOSTResponseRejected kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejected)
	require.NoError(t, err)
	wantPostResponse = kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejected)
	// Test repeated KYC request after REJECTED w/ new email. Should succeed as approved.
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
	var kycStatusPOSTResponseRejectedNewEmail kycPostResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseRejectedNewEmail)
	require.NoError(t, err)
	wantPostResponse = kycPostResponse{
		Result: "no_further_action_required",
	}
	assert.Equal(t, wantPostResponse, kycStatusPOSTResponseRejectedNewEmail)
	// Test POST no email in request.
	noEmailKP := keypair.MustRandom()
	callbackIDNoEmail := uuid.New().String()
	_, err = postHandler.DB.ExecContext(ctx, q, noEmailKP.Address(), callbackIDNoEmail)
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

func TestAPI_GETKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// create kyc-status GetDetailHandler.
	getHandler := GetDetailHandler{
		DB: conn,
	}
	// INSERT new account in accounts_kyc_status.
	const q = `
		INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at)
		VALUES ($1, $2, $3, NOW(), NOW(), NULL)
	`
	approveKP := keypair.MustRandom()
	intendedCallbackIDApprove := uuid.New().String()
	email := "test.email.com"
	_, err := getHandler.DB.ExecContext(ctx, q, approveKP.Address(), intendedCallbackIDApprove, email)
	require.NoError(t, err)
	// Test GET successful; Approved KYC record returned.
	m := chi.NewMux()
	m.Get("/kyc-status/{stellar_address_or_callback_id}", getHandler.ServeHTTP)
	r := httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", intendedCallbackIDApprove), nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var kycRecordGETResponse kycRecord
	err = json.Unmarshal(body, &kycRecordGETResponse)
	require.NoError(t, err)
	wantKYCRecord := kycRecord{
		StellarAddress: approveKP.Address(),
		CallbackID:     intendedCallbackIDApprove,
		EmailAddress:   email,
		KYCSubmittedAt: kycRecordGETResponse.KYCSubmittedAt,
		ApprovedAt:     kycRecordGETResponse.ApprovedAt,
		RejectedAt:     kycRecordGETResponse.RejectedAt,
		CreatedAt:      kycRecordGETResponse.CreatedAt,
	}
	assert.Equal(t, wantKYCRecord, kycRecordGETResponse)

	// Test GET successful; Approved KYC record returned using stellar address.
	r = httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", approveKP.Address()), nil)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	kycRecordGETResponse = kycRecord{}
	err = json.Unmarshal(body, &kycRecordGETResponse)
	require.NoError(t, err)
	assert.Equal(t, wantKYCRecord, kycRecordGETResponse)

	// Test GET Not found, with stellar address.
	notPresentKP := keypair.MustRandom()
	r = httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", notPresentKP.Address()), nil)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantGetResponseNotFound := `{
		"error": "Not found."
	}`
	require.JSONEq(t, wantGetResponseNotFound, string(body))

	// Test GET Not found, with callbackID.
	callbackIDNotFound := uuid.New().String()
	r = httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", callbackIDNotFound), nil)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, wantGetResponseNotFound, string(body))
}
