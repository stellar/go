-- +migrate Up

CREATE TABLE ingest_filter_rules (
    name character varying(256) NOT NULL UNIQUE
    enabled bool NOT NULL default false
    rules jsonb NOT NULL
);

-- +migrate Down

DROP TABLE ingest_filter_rules cascade;
