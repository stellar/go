-- +migrate Up


ALTER TABLE claimable_balances ADD flags int NOT NULL;

-- +migrate Down

ALTER TABLE claimable_balances DROP flags;

