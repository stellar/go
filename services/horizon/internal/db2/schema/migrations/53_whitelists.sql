-- +migrate Up

CREATE TABLE filter_config (
    config jsonb NOT NULL
);

-- +migrate Down

DROP TABLE filter_config cascade;
