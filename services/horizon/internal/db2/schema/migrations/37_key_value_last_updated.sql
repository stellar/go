-- +migrate Up

ALTER TABLE key_value_store
    ADD updated_at timestamp without time zone DEFAULT NOW();

-- +migrate Down

ALTER TABLE key_value_store
    DROP COLUMN updated_at;