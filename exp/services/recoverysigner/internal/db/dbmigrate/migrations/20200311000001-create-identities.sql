-- +migrate Up

CREATE TABLE identities (
  account_id BIGINT NOT NULL,
  id BIGSERIAL NOT NULL,

  FOREIGN KEY (account_id) REFERENCES accounts (id),
  PRIMARY KEY (id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  role TEXT NOT NULL
);

CREATE INDEX identities_accountid ON identities (account_id) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE identities;
