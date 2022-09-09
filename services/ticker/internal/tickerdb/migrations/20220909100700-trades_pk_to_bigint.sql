
-- +migrate Up
ALTER TABLE trades ALTER COLUMN id TYPE BIGINT;
ALTER SEQUENCE trades_id_seq CYCLE;


-- +migrate Down
ALTER TABLE trades ALTER COLUMN id TYPE INT;
ALTER SEQUENCE trades_id_seq NO CYCLE;
