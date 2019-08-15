-- +migrate Up
ALTER TABLE offers ADD last_modified_ledger INT NOT NULL;
CREATE INDEX offers_by_last_modified_ledger ON offers USING BTREE(last_modified_ledger);

-- +migrate Down
ALTER TABLE offers DROP last_modified_ledger;
