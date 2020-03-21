-- +migrate Up

CREATE TABLE identities (
  account_id BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  id BIGSERIAL NOT NULL PRIMARY KEY,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

  role TEXT NOT NULL
);

CREATE INDEX ON identities (account_id);

-- +migrate Down

DROP TABLE identities;
