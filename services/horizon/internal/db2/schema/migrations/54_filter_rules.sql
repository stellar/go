-- +migrate Up

CREATE TABLE ingest_filter_rules (
    name character varying(256) NOT NULL UNIQUE,
    enabled bool NOT NULL default false,
    rules jsonb NOT NULL,
    last_modified bigint NOT NULL
);

-- insert the default disabled state for each supported filter implementation
INSERT INTO ingest_filter_rules VALUES ('asset', false, '{}'::jsonb, 0);
INSERT INTO ingest_filter_rules VALUES ('account', false, '{}'::jsonb, 0);

-- +migrate Down

DROP TABLE ingest_filter_rules cascade;
