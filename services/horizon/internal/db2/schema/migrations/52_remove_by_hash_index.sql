-- +migrate Up
ALTER TABLE history_transactions ADD COLUMN transaction_hash_prefix bigint;
UPDATE history_transactions SET transaction_hash_prefix = ('x' || lpad(transaction_hash, 16, '0'))::bit(64)::bigint;
CREATE INDEX transaction_hash_prefix on history_transactions (transaction_hash_prefix);
DROP INDEX by_hash;

-- +migrate Down
CREATE INDEX by_hash ON history_transactions USING btree (transaction_hash);
DROP INDEX transaction_hash_prefix;
ALTER TABLE history_transactions DROP COLUMN transaction_hash_prefix;