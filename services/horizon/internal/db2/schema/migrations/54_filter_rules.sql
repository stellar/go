-- +migrate Up

CREATE TABLE account_filter_rules (
    enabled bool NOT NULL default false,
    whitelist varchar[] NOT NULL,
    last_modified bigint NOT NULL
);

CREATE TABLE asset_filter_rules (
    enabled bool NOT NULL default false,
    whitelist varchar[] NOT NULL,
    last_modified bigint NOT NULL
);

CREATE TABLE txsub_response (
    transaction_submitted bigint NOT NULL,
    transaction_hash character varying(64) NOT NULL,
    transaction_result text
);

-- insert the default disabled state for each supported filter implementation
INSERT INTO account_filter_rules VALUES (false, '{}', 0);
INSERT INTO asset_filter_rules VALUES (false, '{}', 0);

-- +migrate Down

DROP TABLE account_filter_rules cascade;
DROP TABLE asset_filter_rules cascade;
