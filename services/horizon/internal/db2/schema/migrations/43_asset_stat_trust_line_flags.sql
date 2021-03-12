-- +migrate Up

ALTER TABLE asset_stats
ADD COLUMN accounts JSONB;
ADD COLUMN balances JSONB;
-- Previously, all the asset_stats we stored were authorized.
UPDATE asset_stats
  SET
    accounts = jsonb_build_object('authorized', num_accounts),
    balances = jsonb_build_object('authorized', amount);

-- +migrate Down

ALTER TABLE asset_stats
DROP COLUMN accounts;
DROP COLUMN balances;
