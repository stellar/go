-- +migrate Up

CREATE TABLE accounts (
  id UUID NOT NULL,

  PRIMARY KEY (id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  address TEXT NOT NULL
);

CREATE UNIQUE INDEX accounts_address ON accounts (UPPER(address)) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE accounts;
