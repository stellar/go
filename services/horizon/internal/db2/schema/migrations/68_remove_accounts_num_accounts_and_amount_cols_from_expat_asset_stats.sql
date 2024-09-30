-- + migrate Up

ALTER TABLE exp_asset_stats
    DROP COLUMN num_accounts,
    DROP COLUMN amount;

-- + migrate Down
ALTER TABLE exp_asset_stats
    ADD COLUMN num_accounts INTEGER DEFAULT 0,
    ADD COLUMN amount VARCHAR DEFAULT '';
