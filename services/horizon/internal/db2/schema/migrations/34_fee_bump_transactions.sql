-- +migrate Up

ALTER TABLE history_transactions ADD inner_transaction_hash character varying(64) DEFAULT '' NOT NULL;
ALTER TABLE history_transactions ADD fee_account character varying(64) DEFAULT '' NOT NULL;
ALTER TABLE history_transactions ADD inner_signatures character varying(96)[] DEFAULT '{}'::character varying[] NOT NULL;
ALTER TABLE history_transactions ADD inner_max_fee integer DEFAULT 0 NOT NULL;

CREATE INDEX by_inner_hash ON history_transactions USING btree (inner_transaction_hash);
CREATE INDEX by_fee_account ON history_transactions USING btree (fee_account);

-- +migrate Down

DROP INDEX by_inner_hash;
DROP INDEX by_fee_account;

ALTER TABLE history_transactions DROP COLUMN inner_transaction_hash;
ALTER TABLE history_transactions DROP COLUMN fee_account;
ALTER TABLE history_transactions DROP COLUMN inner_signatures;
ALTER TABLE history_transactions DROP COLUMN inner_max_fee;