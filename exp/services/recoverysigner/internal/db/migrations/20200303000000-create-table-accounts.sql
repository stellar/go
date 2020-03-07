
-- +migrate Up

CREATE TABLE accounts (
  id TEXT NOT NULL PRIMARY KEY,
  created_at TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP,

  address TEXT NOT NULL
);

CREATE UNIQUE INDEX accounts_address ON accounts (upper(address)) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE accounts;
