-- +migrate Up

ALTER TABLE history_transactions ADD fee_charged integer;

-- +migrate Down

ALTER TABLE history_transactions DROP COLUMN fee_charged;
