-- +migrate Up

CREATE TABLE accounts (
  id BIGSERIAL NOT NULL PRIMARY KEY,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE,

  address TEXT NOT NULL
);

CREATE UNIQUE INDEX ON accounts (UPPER(address));

-- +migrate Down

DROP TABLE accounts;
