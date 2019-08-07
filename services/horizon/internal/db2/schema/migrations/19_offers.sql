-- +migrate Up

CREATE TABLE offers (
    sellerid character varying(56) NOT NULL,
    offerid bigint PRIMARY KEY,
    sellingasset text NOT NULL,
    buyingasset text NOT NULL,
    amount bigint NOT NULL,
    pricen integer NOT NULL,
    priced integer NOT NULL,
    price double precision NOT NULL,
    flags integer NOT NULL
);

CREATE INDEX offers_by_seller ON offers USING BTREE(sellerid);
CREATE INDEX offers_by_selling_asset ON offers USING BTREE(sellingasset);
CREATE INDEX offers_by_buying_asset ON offers USING BTREE(buyingasset);

-- Distributed ingestion relies on a single value locked for updating
-- in a DB. When Horizon starts clear there is no value so we create it
-- here. If there's a conflict it means the value is already there so
-- we do nothing.
INSERT INTO key_value_store (key, value)
    VALUES ('exp_ingest_last_ledger', '0')
    ON CONFLICT (key) DO NOTHING;

-- +migrate Down

DROP INDEX offers_by_seller;
DROP INDEX offers_by_selling_asset;
DROP INDEX offers_by_buying_asset;

DROP TABLE offers cascade;
