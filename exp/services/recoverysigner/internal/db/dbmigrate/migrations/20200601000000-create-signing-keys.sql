-- +migrate Up

CREATE TABLE signing_keys (
  id BIGINT NOT NULL PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  encrypted_seed BYTEA NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


CREATE INDEX ON signing_keys (account_id);
CREATE INDEX ON signing_keys (key);

-- +migrate Down

DROP TABLE signing_keys;
