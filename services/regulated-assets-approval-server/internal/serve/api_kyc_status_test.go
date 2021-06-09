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
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/kycstatus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_postKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	handler := kycstatus.PostHandler{DB: conn}
	m := chi.NewMux()
	m.Post("/kyc-status/{callback_id}", handler.ServeHTTP)

	q := `
		INSERT INTO accounts_kyc_status (stellar_address, callback_id)
		VALUES ($1, $2)
	`
	clientKP := keypair.MustRandom()
	callbackID := uuid.New().String()
	_, err := handler.DB.ExecContext(ctx, q, clientKP.Address(), callbackID)
	require.NoError(t, err)

	r := httptest.NewRequest("POST", "/kyc-status/"+callbackID, strings.NewReader(`{"email_address": "email@test.com"}`))
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	type kycStatusPOSTResponse struct {
		Result string `json:"result"`
	}
	var kycStatusPOSTResponseApprove kycStatusPOSTResponse
	err = json.Unmarshal(body, &kycStatusPOSTResponseApprove)
	require.NoError(t, err)
	wantPostResponse := kycStatusPOSTResponse{
		Result: "no_further_action_required",
	}
	require.Equal(t, wantPostResponse, kycStatusPOSTResponseApprove)

	q = `
		SELECT rejected_at, pending_at, approved_at
		FROM accounts_kyc_status
		WHERE stellar_address = $1 AND callback_id = $2
	`
	var rejectedAt, pendingAt, approvedAt sql.NullTime
	err = handler.DB.QueryRowContext(ctx, q, clientKP.Address(), callbackID).Scan(&rejectedAt, &pendingAt, &approvedAt)
	require.NoError(t, err)

	assert.True(t, approvedAt.Valid)
	assert.False(t, rejectedAt.Valid)
	assert.False(t, pendingAt.Valid)
}

func TestAPI_getKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	handler := kycstatus.GetDetailHandler{DB: conn}
	m := chi.NewMux()
	m.Get("/kyc-status/{stellar_address_or_callback_id}", handler.ServeHTTP)

	const q = `
		INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at, pending_at, created_at)
		VALUES ($1, $2, $3, $4::timestamptz, $4::timestamptz, NULL, NULL, $5::timestamptz)
	`
	clientKP := keypair.MustRandom()
	callbackID := uuid.New().String()
	emailAddress := "email@test.com"
	approvedAt := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	createdAt := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339)
	_, err := handler.DB.ExecContext(ctx, q, clientKP.Address(), callbackID, emailAddress, approvedAt, createdAt)
	require.NoError(t, err)

	r := httptest.NewRequest("GET", fmt.Sprintf("/kyc-status/%s", clientKP.Address()), nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := fmt.Sprintf(`{
		"stellar_address": "%s",
		"callback_id": "%s",
		"email_address": "%s",
		"created_at": "%s",
		"kyc_submitted_at": "%s",
		"approved_at": "%s"
	}`, clientKP.Address(), callbackID, emailAddress, createdAt, approvedAt, approvedAt)
	require.JSONEq(t, wantBody, string(body))
}

func TestAPI_deleteKYCStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	handler := kycstatus.DeleteHandler{DB: conn}
	m := chi.NewMux()
	m.Delete("/kyc-status/{stellar_address}", handler.ServeHTTP)

	q := `
		INSERT INTO accounts_kyc_status (stellar_address, callback_id, email_address, kyc_submitted_at, approved_at, rejected_at, pending_at)
		VALUES ($1, $2, $3, NOW(), NOW(), NULL, NULL)
	`
	approveKP := keypair.MustRandom()
	approveCallbackID := uuid.New().String()
	approveEmailAddress := "email@test.com"
	_, err := handler.DB.ExecContext(ctx, q, approveKP.Address(), approveCallbackID, approveEmailAddress)
	require.NoError(t, err)

	r := httptest.NewRequest("DELETE", fmt.Sprintf("/kyc-status/%s", approveKP.Address()), nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"message":"ok"
	}`
	require.JSONEq(t, wantBody, string(body))

	q = `
		SELECT EXISTS(
			SELECT stellar_address
			FROM accounts_kyc_status
			WHERE stellar_address = $1
		)
	`
	var exists bool
	err = handler.DB.QueryRowContext(ctx, q, approveKP.Address()).Scan(&exists)
	require.NoError(t, err)
	require.False(t, exists)
}
