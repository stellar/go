-- +migrate Up

CREATE TABLE identities (
  id UUID NOT NULL PRIMARY KEY,
  created_at TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP,

  account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  role TEXT NOT NULL
);

CREATE INDEX identities_account_id ON identities (account_id) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE identities;
