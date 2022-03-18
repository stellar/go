-- +migrate Up

-- CAP-40 signed payload strkeys can be 165 characters long, see
-- strkey/main.go:maxEncodedSize
ALTER TABLE accounts_signers
  ALTER COLUMN signer TYPE character varying(165);

-- +migrate Down
ALTER TABLE accounts_signers
  ALTER COLUMN signer TYPE character varying(64);
