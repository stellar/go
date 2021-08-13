-- +migrate Up

CREATE TABLE liquidity_pools (
    id text NOT NULL, -- hex-encoded PoolID
    type smallint NOT NULL,
    fee integer NOT NULL,
    trustline_count bigint NOT NULL CHECK (trustline_count > 0),
    share_count bigint NOT NULL DEFAULT 0 CHECK(share_count >= 0),
    asset_reserves jsonb NOT NULL,
    last_modified_ledger integer NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX liquidity_pools_by_asset_reserves ON liquidity_pools USING gin(asset_reserves jsonb_path_ops);

-- +migrate Down

DROP TABLE liquidity_pools cascade;

