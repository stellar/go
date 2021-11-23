-- +migrate Up
ALTER TABLE history_transactions ADD COLUMN transaction_hash_prefix bigint;
CREATE INDEX transaction_hash_prefix on history_transactions (transaction_hash_prefix);

-- +migrate Down
DROP INDEX transaction_hash_prefix;
ALTER TABLE history_transactions DROP COLUMN transaction_hash_prefix;