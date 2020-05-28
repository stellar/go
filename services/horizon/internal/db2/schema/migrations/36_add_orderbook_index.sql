-- +migrate Up

CREATE INDEX best_offer ON offers USING btree (selling_asset, buying_asset, price);

-- +migrate Down

DROP INDEX best_offer;