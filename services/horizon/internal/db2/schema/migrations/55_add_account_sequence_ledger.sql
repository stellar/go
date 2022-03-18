-- +migrate Up
ALTER TABLE accounts ADD sequence_ledger integer null;
ALTER TABLE accounts ADD sequence_time timestamp without time zone null;

-- +migrate Down
ALTER TABLE accounts DROP sequence_ledger;
ALTER TABLE accounts DROP sequence_time;
