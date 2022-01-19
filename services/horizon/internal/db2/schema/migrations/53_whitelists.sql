-- +migrate Up

CREATE TABLE account_filter_whitelist (
    account_id character varying(56) NOT NULL UNIQUE
);

-- +migrate Down

DROP TABLE account_filter_whitelist cascade;
