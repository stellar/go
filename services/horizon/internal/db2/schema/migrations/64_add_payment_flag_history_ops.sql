-- +migrate Up

ALTER TABLE history_operations ADD is_payment boolean;
CREATE INDEX "index_history_operations_on_is_payment" ON history_operations (is_payment)
    WHERE is_payment IS NOT NULL;

-- +migrate Down

DROP INDEX "index_history_operations_on_is_payment";
ALTER TABLE history_operations DROP COLUMN is_payment;
