-- +migrate Up


ALTER TABLE trust_lines ADD liquidity_pool_id TEXT;
CREATE INDEX trust_lines_by_liquidity_pool_id ON trust_lines USING BTREE(liquidity_pool_id);

-- +migrate Down

DROP INDEX trust_lines_by_liquidity_pool_id;
ALTER TABLE trust_lines DROP liquidity_pool_id;
