-- +migrate Up

-- CAP-40 signed payload strkeys can be 165 characters long, see
-- strkey/main.go:maxEncodedSize. But we use text here, so we don't need to
-- adjust it *ever again*.
ALTER TABLE accounts_signers
  ALTER COLUMN signer TYPE text;

-- +migrate Down
ALTER TABLE accounts_signers
  ALTER COLUMN signer TYPE character varying(64);
