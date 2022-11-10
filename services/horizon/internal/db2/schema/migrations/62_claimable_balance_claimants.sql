-- +migrate Up

CREATE TABLE claimable_balance_claimants (
    id TEXT NOT NULL, -- ClaimableBalanceID in base64
    destination character varying(56) NOT NULL,
    PRIMARY KEY (id, destination)
);

-- +migrate Down

DROP TABLE claimable_balance_claimants cascade;
