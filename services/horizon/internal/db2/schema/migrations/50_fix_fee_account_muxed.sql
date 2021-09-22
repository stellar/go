-- +migrate Up

UPDATE history_transactions SET fee_account_muxed = NULL WHERE fee_account IS NOT NULL AND fee_account_muxed LIKE 'G%';

-- +migrate Down
