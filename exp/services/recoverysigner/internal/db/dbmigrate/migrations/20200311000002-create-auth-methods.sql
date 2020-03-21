-- +migrate Up

CREATE TYPE auth_method_type AS ENUM (
  'stellar_address',
  'phone_number',
  'email'
);

CREATE TABLE auth_methods (
  account_id BIGINT NOT NULL,
  identity_id BIGINT NOT NULL,
  id BIGSERIAL NOT NULL,

  FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
  FOREIGN KEY (identity_id) REFERENCES identities (id) ON DELETE CASCADE,
  PRIMARY KEY (id),

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

  "type" auth_method_type NOT NULL,
  value text NOT NULL
);

CREATE INDEX ON auth_methods (account_id);
CREATE INDEX ON auth_methods (identity_id);
CREATE INDEX ON auth_methods ("type", value);

-- +migrate Down

DROP TABLE auth_methods;
DROP TYPE auth_method_type;
