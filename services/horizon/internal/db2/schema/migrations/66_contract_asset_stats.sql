-- +migrate Up
CREATE TABLE contract_asset_stats (
     contract_id BYTEA PRIMARY KEY ,
     stat       JSONB NOT NULL
);

-- +migrate Down
DROP TABLE contract_asset_stats cascade;