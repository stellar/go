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

-- +migrate Down

DROP INDEX offers_by_seller;
DROP INDEX offers_by_selling_asset;
DROP INDEX offers_by_buying_asset;

DROP TABLE offers cascade;