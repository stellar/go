-- +migrate Up

-- unused indices
DROP INDEX by_account;
DROP INDEX by_fee_account;


-- +migrate Down
CREATE INDEX by_account ON history_transactions USING btree (account, account_sequence);
CREATE INDEX by_fee_account ON history_transactions USING btree (fee_account) WHERE fee_account IS NOT NULL;
