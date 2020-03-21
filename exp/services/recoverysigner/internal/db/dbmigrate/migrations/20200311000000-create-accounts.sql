-- +migrate Up

CREATE TABLE accounts (
  id BIGSERIAL NOT NULL,

  PRIMARY KEY (id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  address TEXT NOT NULL
);

CREATE UNIQUE INDEX ON accounts (UPPER(address)) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE accounts;
