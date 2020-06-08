-- +migrate Up

CREATE TABLE signers (
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  id BIGINT NOT NULL PRIMARY KEY GENERATED ALWAYS AS IDENTITY,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE,

  public_key TEXT NOT NULL,
  encrypted_secret_key BYTEA NOT NULL,
  encryption_key_id TEXT NOT NULL
);


CREATE INDEX ON signers (account_id);
CREATE INDEX ON signers (public_key);

-- +migrate Down

DROP TABLE signers;
