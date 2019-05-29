package keystore

import (
	"context"
	"database/sql"
	"encoding/base64"
	"time"

	"github.com/lib/pq"
	"github.com/stellar/go/support/errors"
)

type encryptedKeys struct {
	KeysBlob      string     `json:"keys_blob"`
	Salt          string     `json:"salt"`
	EncrypterName string     `json:"encrypter_name"`
	CreatedAt     time.Time  `json:"created_at"`
	ModifiedAt    *time.Time `json:"modified_at,omitempty"`
}

type storeKeysRequest struct {
	KeysBlob      string `json:"keys_blob"`
	Salt          string `json:"salt"`
	EncrypterName string `json:"encrypter_name"`
}

func (s *Service) storeKeys(ctx context.Context, in storeKeysRequest) (*encryptedKeys, error) {
	userID := userID(ctx)
	if userID == "" {
		return nil, probNotAuthorized
	}

	keysData, err := base64.RawURLEncoding.DecodeString(string(in.KeysBlob))
	if err != nil {
		// TODO: we need to implement a helper function in the
		// support/error package for keeping the stack trace from err
		// and substitude the root error for the one we want for better
		// debugging experience.
		// Thowing away the original err makes it harder for debugging.
		return nil, probInvalidKeysBlob
	}

	q := `
		INSERT INTO encrypted_keys (user_id, encrypted_keys_data, salt, encrypter_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET encrypted_keys_data = excluded.encrypted_keys_data, salt = excluded.salt, encrypter_name = excluded.encrypter_name,  modified_at = NOW()
		RETURNING encrypted_keys_data, salt, encrypter_name, created_at, modified_at
	`
	var (
		keysBlob   []byte
		out        encryptedKeys
		modifiedAt pq.NullTime
	)
	err = s.db.QueryRowContext(ctx, q, userID, keysData, in.Salt, in.EncrypterName).Scan(&keysBlob, &out.Salt, &out.EncrypterName, &out.CreatedAt, &modifiedAt)
	if err == sql.ErrNoRows {
		return nil, probDuplicateKeys
	}

	out.KeysBlob = base64.RawURLEncoding.EncodeToString(keysBlob)
	if modifiedAt.Valid {
		out.ModifiedAt = &modifiedAt.Time
	}
	return &out, errors.Wrap(err, "storing keys blob")
}
