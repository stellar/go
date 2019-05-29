package keystore

import (
	"context"
	"encoding/base64"
	"testing"
	"time"
)

func TestStoreKeys(t *testing.T) {
	db := openKeystoreDB(t)
	defer db.Close() // drop test db

	conn := db.Open()
	defer conn.Close() // close db connection

	ctx := withUserID(context.Background(), "test-user")
	s := &Service{conn.DB}

	blob := `{
		"type": "plaintextKey",
		"pubkey": "stellar-pubkey",
		"encrypted_seed": "encrypted-stellar-privatekey"
	}`
	encodedBlob := base64.RawURLEncoding.EncodeToString([]byte(blob))
	encrypterName := "identity"
	salt := "random-salt"
	req := storeKeysRequest{
		KeysBlob:      encodedBlob,
		EncrypterName: encrypterName,
		Salt:          salt,
	}

	got, err := s.storeKeys(ctx, req)
	if err != nil {
		t.Fatal(err)
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
