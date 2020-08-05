-- +migrate Up

CREATE TABLE claimable_balances (
    id character varying(64) NOT NULL, -- sha256 of balance_id in HEX
    balance_id character varying(48) NOT NULL, -- ClaimableBalanceID in base64
    asset text NOT NULL,
    amount bigint NOT NULL CHECK (amount > 0),
    sponsor character varying(56),
    last_modified_ledger integer NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE claimable_balances_claimants (
    id character varying(64) NOT NULL, -- sha256 of balance_id in HEX
    destination character varying(64) NOT NULL,
    predicate TEXT NOT NULL,
    PRIMARY KEY (id, destination)
);

CREATE INDEX claimable_balances_by_asset ON claimable_balances USING BTREE(asset);
CREATE INDEX claimable_balances_by_sponsor ON claimable_balances USING BTREE(sponsor);
CREATE INDEX claimable_balances_claimants_by_destination ON claimable_balances_claimants USING BTREE(destination);

-- +migrate Down

DROP TABLE claimable_balances cascade;
DROP TABLE claimable_balances_claimants cascade;
