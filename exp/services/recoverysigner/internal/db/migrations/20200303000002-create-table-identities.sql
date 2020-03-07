
-- +migrate Up

CREATE TYPE identity_role AS ENUM ('registrant', 'other');

CREATE TYPE identity_type AS ENUM ('account', 'phone_number', 'email');

CREATE TABLE identities (
  id TEXT NOT NULL PRIMARY KEY,
  created_at TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP,

  account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  role identity_role NOT NULL,
  "type" identity_type NOT NULL,
  value text NOT NULL
);

CREATE INDEX identities_account_id ON identities (account_id) WHERE deleted_at IS NULL;
CREATE INDEX identities_type_value ON identities ("type", value) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE identities;
DROP TYPE identity_type;
DROP TYPE identity_role;
