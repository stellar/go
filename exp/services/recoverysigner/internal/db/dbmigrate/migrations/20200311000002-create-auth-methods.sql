-- +migrate Up

CREATE TYPE auth_method_type AS ENUM ('stellar_address', 'phone_number', 'email');

CREATE TABLE auth_methods (
  account_id UUID NOT NULL,
  identity_id UUID NOT NULL,
  id UUID NOT NULL,

  FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
  FOREIGN KEY (account_id, identity_id) REFERENCES identities (account_id, id) ON DELETE CASCADE,
  PRIMARY KEY (account_id, identity_id, id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  "type" auth_method_type NOT NULL,
  value text NOT NULL
);

CREATE INDEX auth_methods_accountid_identityid ON auth_methods (account_id, identity_id) WHERE deleted_at IS NULL;
CREATE INDEX auth_methods_type_value ON auth_methods ("type", value) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE auth_methods;
DROP TYPE auth_method_type;
