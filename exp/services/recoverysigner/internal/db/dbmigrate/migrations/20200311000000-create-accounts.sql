-- +migrate Up

CREATE TABLE accounts (
  id BIGSERIAL NOT NULL,

  PRIMARY KEY (id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,

  address TEXT NOT NULL
);

CREATE UNIQUE INDEX ON accounts (UPPER(address));

-- +migrate Down

DROP TABLE accounts;
