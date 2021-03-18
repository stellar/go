-- +migrate Up

ALTER TABLE exp_asset_stats
ADD COLUMN accounts JSONB;
ADD COLUMN balances JSONB;
-- Previously, all the exp_asset_stats we stored were authorized.
UPDATE exp_asset_stats
  SET
    accounts = jsonb_build_object('authorized', num_accounts),
    balances = jsonb_build_object('authorized', amount);

-- +migrate Down

ALTER TABLE exp_asset_stats
DROP COLUMN accounts;
DROP COLUMN balances;
