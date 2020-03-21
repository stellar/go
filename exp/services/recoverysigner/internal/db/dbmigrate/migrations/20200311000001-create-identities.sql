-- +migrate Up

CREATE TABLE identities (
  account_id BIGINT NOT NULL,
  id BIGSERIAL NOT NULL,

  FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
  PRIMARY KEY (id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

  role TEXT NOT NULL
);

CREATE INDEX ON identities (account_id);

-- +migrate Down

DROP TABLE identities;
