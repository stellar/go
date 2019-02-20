-- +migrate Up

ALTER TABLE history_transactions ADD successful boolean DEFAULT true;

-- +migrate Down

ALTER TABLE history_transactions DROP COLUMN successful;
