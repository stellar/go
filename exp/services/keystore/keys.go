package keystore

import (
	"context"
	"encoding/base64"
)

type storeKeysRequest struct {
	KeysBlob      string `json:"keys_blob"`
	Salt          string `json:"salt"`
	EncrypterName string `json:"encrypter_name"`
}

func (s *Service) storeKeys(ctx context.Context, in storeKeysRequest) (string, error) {
	userID := userID(ctx)
	if userID == "" {
		return "", errNotAuthorized
	}

	data, err := base64.RawURLEncoding.DecodeString(string(in.KeysBlob))
	if err != nil {
		return "", errBadKeysBlob
	}

	q := `
		INSERT INTO encrypted_keys (user_id, encrypted_keys_data, salt, encrypter_name)
		VALUES ($1, $2, $3, $4)
		RETURNING encrypted_keys_data, salt, encrypter_name, created_at
	`
	_, err = s.db.ExecContext(ctx, q, userID, data, in.Salt, in.EncrypterName)
	return "", nil
}
