-- +migrate Up

DROP INDEX htrd_by_offer;
DROP INDEX htrd_counter_lookup;

ALTER TABLE history_trades DROP offer_id,
                           ALTER base_account_id DROP NOT NULL,
                           ALTER counter_account_id DROP NOT NULL,
                           ADD base_liquidity_pool_id BIGINT,
                           ADD counter_liquidity_pool_id BIGINT,
                           ADD liquidity_pool_fee INT;

CREATE INDEX htrd_by_base_liquidity_pool_id ON history_trades USING BTREE(base_liquidity_pool_id);
CREATE INDEX htrd_by_counter_liquidity_pool_id ON history_trades USING BTREE(counter_liquidity_pool_id);

-- +migrate Down

DROP INDEX htrd_by_counter_liquidity_pool_id;
DROP INDEX htrd_by_base_liquidity_pool_id;

DELETE FROM history_trades WHERE (counter_account_id IS NULL) OR (base_account_id IS NULL);

ALTER TABLE history_trades DROP liquidity_pool_fee,
                           DROP counter_liquidity_pool_id,
                           DROP base_liquidity_pool_id,
                           ALTER counter_account_id SET NOT NULL,
                           ALTER base_account_id SET NOT NULL,
                           ADD offer_id BIGINT;

CREATE INDEX htrd_counter_lookup on history_trades USING BTREE(counter_asset_id);
CREATE INDEX htrd_by_offer ON history_trades USING BTREE(offer_id);
