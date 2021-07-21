-- +migrate Up

ALTER TABLE accounts
ADD COLUMN sequence_time bigint,
ADD COLUMN sequence_ledger int;

-- +migrate Down

ALTER TABLE accounts
DROP COLUMN sequence_time,
DROP COLUMN sequence_ledger;
