-- +migrate Up

CREATE TYPE auth_method_type AS ENUM ('stellar_address', 'phone_number', 'email');

CREATE TABLE auth_methods (
  id UUID NOT NULL PRIMARY KEY,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  deleted_at TIMESTAMP WITH TIME ZONE,

  identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
  "type" auth_method_type NOT NULL,
  value text NOT NULL
);

CREATE INDEX auth_methods_identity_id ON auth_methods (identity_id) WHERE deleted_at IS NULL;
CREATE INDEX auth_methods_type_value ON auth_methods ("type", value) WHERE deleted_at IS NULL;

-- +migrate Down

DROP TABLE auth_methods;
DROP TYPE auth_method_type;
