package keystore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPutKeysAPI(t *testing.T) {
	db := openKeystoreDB(t)
	defer db.Close() // drop test db

	conn := db.Open()
	defer conn.Close() // close db connection

	h := ServeMux(&Service{conn.DB})

	blob := `[{
		"keyType": "plaintextKey",
		"publicKey": "stellar-pubkey",
		"privateKey": "encrypted-stellar-privatekey"
	}]`
	encodedBlob := base64.RawURLEncoding.EncodeToString([]byte(blob))
	encrypterName := "identity"
	salt := "random-salt"
	body, err := json.Marshal(putKeysRequest{
		KeysBlob:      encodedBlob,
		EncrypterName: encrypterName,
		Salt:          salt,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PUT", "/keys", bytes.NewReader([]byte(body)))
	req = req.WithContext(withUserID(context.Background(), "test-user"))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PUT %s responded with %s, want %s", req.URL, http.StatusText(rr.Code), http.StatusText(http.StatusOK))
	}
	got := &encryptedKeys{}
	json.Unmarshal(rr.Body.Bytes(), &got)
	if got == nil {
		t.Error("Expected to receive an encryptedKeys response but did not")
	}

	if got.KeysBlob != encodedBlob {
		t.Errorf("got blob: %s, want: %s\n", got.KeysBlob, encodedBlob)
	}

	if got.EncrypterName != encrypterName {
		t.Errorf("got encrypter name: %s, want: %s\n", got.EncrypterName, encrypterName)
	}

	if got.Salt != salt {
		t.Errorf("got salt: %s, want: %s\n", got.Salt, salt)
	}

	if got.CreatedAt.Before(time.Now().Add(-time.Hour)) {
		t.Errorf("got CreatedAt=%s, want CreatedAt within the last hour", got.CreatedAt)
	}
}

func TestGetKeysAPI(t *testing.T) {
	db := openKeystoreDB(t)
	defer db.Close() // drop test db

	conn := db.Open()
	defer conn.Close() // close db connection

	ctx := withUserID(context.Background(), "test-user")
	s := &Service{conn.DB}
	h := ServeMux(s)

	blob := `[{
		"keyType": "plaintextKey",
		"publicKey": "stellar-pubkey",
		"privateKey": "encrypted-stellar-privatekey"
	}]`
	encodedBlob := base64.RawURLEncoding.EncodeToString([]byte(blob))
	encrypterName := "identity"
	salt := "random-salt"

	_, err := s.putKeys(ctx, putKeysRequest{
		KeysBlob:      encodedBlob,
		EncrypterName: encrypterName,
		Salt:          salt,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/keys", nil)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET %s responded with %s, want %s", req.URL, http.StatusText(rr.Code), http.StatusText(http.StatusOK))
	}
	got := &encryptedKeys{}
	json.Unmarshal(rr.Body.Bytes(), &got)
	if got == nil {
		t.Error("Expected to receive an encryptedKeys response but did not")
	}

	if got.KeysBlob != encodedBlob {
		t.Errorf("got blob: %s, want: %s\n", got.KeysBlob, encodedBlob)
	}

	if got.EncrypterName != encrypterName {
		t.Errorf("got encrypter name: %s, want: %s\n", got.EncrypterName, encrypterName)
	}

	if got.Salt != salt {
		t.Errorf("got salt: %s, want: %s\n", got.Salt, salt)
	}

	if got.CreatedAt.Before(time.Now().Add(-time.Hour)) {
		t.Errorf("got CreatedAt=%s, want CreatedAt within the last hour", got.CreatedAt)
	}
}
