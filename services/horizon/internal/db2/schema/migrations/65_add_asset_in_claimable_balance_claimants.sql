-- +migrate Up

ALTER TABLE claimable_balance_claimants ADD asset text;
CREATE INDEX "claimants_by_asset_destination_last_modified_ledger_id" ON claimable_balance_claimants USING btree (asset, destination, last_modified_ledger, id);

-- +migrate Down

DROP INDEX "claimants_by_asset_destination_last_modified_ledger_id";
ALTER TABLE claimable_balance_claimants DROP COLUMN asset;
