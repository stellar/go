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

CREATE SEQUENCE history_liquidity_pools_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE history_liquidity_pools (
    id bigint NOT NULL DEFAULT nextval('history_liquidity_pools_id_seq'::regclass),
    liquidity_pool_id text NOT NULL
);

CREATE UNIQUE INDEX index_history_liquidity_pools_on_id ON history_liquidity_pools USING btree (id);
CREATE UNIQUE INDEX index_history_liquidity_pools_on_liquidity_pool_id ON history_liquidity_pools USING btree (liquidity_pool_id);

CREATE TABLE history_operation_liquidity_pools (
    history_operation_id bigint NOT NULL,
    history_liquidity_pool_id bigint NOT NULL
);

CREATE UNIQUE INDEX index_history_operation_liquidity_pools_on_ids ON history_operation_liquidity_pools USING btree (history_operation_id , history_liquidity_pool_id);
CREATE INDEX index_history_operation_liquidity_pools_on_operation_id ON history_operation_liquidity_pools USING btree (history_operation_id);

CREATE TABLE history_transaction_liquidity_pools (
    history_transaction_id bigint NOT NULL,
    history_liquidity_pool_id bigint NOT NULL
);

CREATE UNIQUE INDEX index_history_transaction_liquidity_pools_on_ids ON history_transaction_liquidity_pools USING btree (history_transaction_id , history_liquidity_pool_id);
CREATE INDEX index_history_transaction_liquidity_pools_on_transaction_id ON history_transaction_liquidity_pools USING btree (history_transaction_id);

-- +migrate Down

DROP TABLE history_liquidity_pools cascade;

DROP SEQUENCE history_liquidity_pools_id_seq;

DROP TABLE history_operation_liquidity_pools cascade;

DROP TABLE history_transaction_liquidity_pools cascade;

DROP TABLE liquidity_pools cascade;

