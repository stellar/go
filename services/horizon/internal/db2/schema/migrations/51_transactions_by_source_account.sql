-- +migrate Up

-- unused indices
DROP INDEX by_account;
DROP INDEX by_fee_account;

-- recreate by_account index without the sequence, which will be used to query transactions by_source_account
CREATE INDEX history_transactions_by_source_account ON history_transactions USING btree (account);



-- +migrate Down

DROP INDEX history_transactions_by_source_account;

CREATE INDEX by_account ON history_transactions USING btree (account, account_sequence);
CREATE INDEX by_fee_account ON history_transactions USING btree (fee_account) WHERE fee_account IS NOT NULL;
