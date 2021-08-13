-- +migrate Up


ALTER TABLE trust_lines ADD liquidity_pool_id TEXT NOT NULL DEFAULT '';

-- +migrate Down

ALTER TABLE trust_lines DROP liquidity_pool_id;
