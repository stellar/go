package keystore

import (
	"context"
	"database/sql"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stellar/go/support/errors"
)

func TestPutKeys(t *testing.T) {
	db := openKeystoreDB(t)
	defer db.Close() // drop test db

	conn := db.Open()
	defer conn.Close() // close db connection

	ctx := withUserID(context.Background(), "test-user")
	s := &Service{conn.DB, nil}

	blob := `[{
		"id": "test-id,"
		"salt": "test-salt",
		"encrypterName": "test-encrypter-name",
		"encryptedBlob": "test-encryptedblob"
	}]`
	keysBlob := base64.RawURLEncoding.EncodeToString([]byte(blob))

	got, err := s.putKeys(ctx, putKeysRequest{KeysBlob: keysBlob})
	if err != nil {
		t.Fatal(err)
	}

	if got.KeysBlob != keysBlob {
		t.Errorf("got blob: %s, want: %s\n", got.KeysBlob, keysBlob)
	}

	if got.CreatedAt.Before(time.Now().Add(-time.Hour)) {
		t.Errorf("got CreatedAt=%s, want CreatedAt within the last hour", got.CreatedAt)
	}
}

func TestGetKeys(t *testing.T) {
	db := openKeystoreDB(t)
	defer db.Close() // drop test db

	conn := db.Open()
	defer conn.Close() // close db connection

	ctx := withUserID(context.Background(), "test-user")
	s := &Service{conn.DB, nil}

	blob := `[{
		"id": "test-id,"
		"salt": "test-salt",
		"encrypterName": "test-encrypter-name",
		"encryptedBlob": "test-encryptedblob"
	}]`
	keysBlob := base64.RawURLEncoding.EncodeToString([]byte(blob))

	_, err := s.putKeys(ctx, putKeysRequest{KeysBlob: keysBlob})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.getKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if got.KeysBlob != keysBlob {
		t.Errorf("got blob: %s, want: %s\n", got.KeysBlob, keysBlob)
	}

	if got.CreatedAt.Before(time.Now().Add(-time.Hour)) {
		t.Errorf("got CreatedAt=%s, want CreatedAt within the last hour", got.CreatedAt)
	}
}

func TestDeleteKeys(t *testing.T) {
	db := openKeystoreDB(t)
	defer db.Close() // drop test db

	conn := db.Open()
	defer conn.Close() // close db connection

	ctx := withUserID(context.Background(), "test-user")
	s := &Service{conn.DB, nil}

	blob := `[{
		"id": "test-id,"
		"salt": "test-salt",
		"encrypterName": "test-encrypter-name",
		"encryptedBlob": "test-encryptedblob"
	}]`
	keysBlob := base64.RawURLEncoding.EncodeToString([]byte(blob))

	_, err := s.putKeys(ctx, putKeysRequest{KeysBlob: keysBlob})
	if err != nil {
		t.Fatal(err)
	}

	err = s.deleteKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.getKeys(ctx)
	if errors.Cause(err) != sql.ErrNoRows {
		t.Errorf("expect the keys blob of the user %s to be deleted", userID(ctx))
	}
}
