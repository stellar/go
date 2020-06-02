-- +migrate Up

CREATE TABLE signing_keys (
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  id BIGINT NOT NULL PRIMARY KEY GENERATED ALWAYS AS IDENTITY,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

  public_key TEXT NOT NULL,
  encrypted_secret_key BYTEA NOT NULL,
  encryption_keyid TEXT NOT NULL
);


CREATE INDEX ON signing_keys (account_id);
CREATE INDEX ON signing_keys (public_key);

-- +migrate Down

DROP TABLE signing_keys;
