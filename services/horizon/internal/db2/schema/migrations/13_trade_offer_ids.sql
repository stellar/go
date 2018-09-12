-- +migrate Up

ALTER TABLE history_trades ADD base_offer_id VARCHAR(21);
ALTER TABLE history_trades ADD counter_offer_id VARCHAR(21);

CREATE INDEX htrd_by_base_offer ON history_trades USING btree(base_offer_id);
CREATE INDEX htrd_by_counter_offer ON history_trades USING btree(base_offer_id);

-- +migrate Down

DROP INDEX htrd_by_base_offer
DROP INDEX htrd_by_counter_offer

ALTER TABLE history_trades DROP COLUMN base_offer_id;
ALTER TABLE history_trades DROP COLUMN counter_offer_id;
