-- +migrate Up

CREATE TABLE liquidity_pools (
    id text NOT NULL, -- PoolID in base64
    type smallint NOT NULL,
    fee integer NOT NULL,
    trustline_count bigint NOT NULL CHECK (trustline_count > 0),
    share_count bigint NOT NULL DEFAULT 0 CHECK(share_count >= 0),
    sponsor text,
    last_modified_ledger integer NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX liquidity_pools_by_sponsor ON liquidity_pools USING BTREE(sponsor);

CREATE TABLE liquidity_pool_assets (
    liquidity_pool_id TEXT NOT NULL,
    asset text NOT NULL,
    reserve bigint DEFAULT 0 CHECK(reserve >= 0),
    FOREIGN KEY (liquidity_pool_id) REFERENCES liquidity_pools(id),
    PRIMARY KEY(liquidity_pool_id, asset) -- no asset duplicates in the same pool
);

CREATE INDEX liquidity_pool_assets_by_asset ON liquidity_pool_assets USING BTREE(asset);
CREATE INDEX liquidity_pool_assets_by_pool ON liquidity_pool_assets USING BTREE(liquidity_pool_id);

-- +migrate Down

DROP TABLE liquidity_pool_assets cascade; -- remove it first, due to the foreign key
DROP TABLE liquidity_pools cascade;

