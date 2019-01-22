-- +migrate Up

ALTER TABLE history_ledgers ADD failed_transaction_count integer DEFAULT NULL;
ALTER TABLE history_ledgers RENAME column transaction_count TO successful_transaction_count;

-- +migrate Down

ALTER TABLE history_ledgers RENAME column successful_transaction_count TO transaction_count;
ALTER TABLE history_ledgers DROP COLUMN failed_transaction_count;