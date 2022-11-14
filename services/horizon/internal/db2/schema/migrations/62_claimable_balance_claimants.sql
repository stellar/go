-- +migrate Up

CREATE TABLE claimable_balance_claimants (
    id TEXT NOT NULL, -- ClaimableBalanceID in base64
    destination character varying(56) NOT NULL,
    PRIMARY KEY (id, destination)
);

CREATE INDEX "claimable_balance_claimants_by_destination" ON claimable_balance_claimants USING btree (destination);

-- +migrate Down

DROP TABLE claimable_balance_claimants cascade;
