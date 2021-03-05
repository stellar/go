-- +migrate Up

ALTER TABLE asset_stats
ADD COLUMN trust_line_flags integer DEFAULT 0;
-- Previously, all the asset_stats we stored were authorized.
UPDATE asset_stats SET trust_line_flags = 1;

-- +migrate Down

ALTER TABLE asset_stats
DROP COLUMN trust_line_flags;
