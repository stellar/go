-- +migrate Up
ALTER TABLE history_trades ADD price_n BIGINT;
ALTER TABLE history_trades ADD price_d BIGINT;

-- +migrate Down
ALTER TABLE history_trades DROP price_n;
ALTER TABLE history_trades DROP price_d;