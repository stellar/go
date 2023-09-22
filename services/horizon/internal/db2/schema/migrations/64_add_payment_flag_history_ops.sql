-- +migrate Up

ALTER TABLE history_operations ADD is_payment boolean;
CREATE INDEX "index_history_operations_on_is_payment" ON history_operations USING btree (is_payment);

-- +migrate Down

DROP INDEX "index_history_operations_on_is_payment";
ALTER TABLE history_operations DROP COLUMN is_payment;
