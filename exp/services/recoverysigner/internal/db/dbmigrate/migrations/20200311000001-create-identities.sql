-- +migrate Up

CREATE TABLE identities (
  account_id UUID NOT NULL,
  id UUID NOT NULL,

  FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
  PRIMARY KEY (account_id, id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  role TEXT NOT NULL
);

CREATE INDEX identities_account_id ON identities (account_id) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE identities;
