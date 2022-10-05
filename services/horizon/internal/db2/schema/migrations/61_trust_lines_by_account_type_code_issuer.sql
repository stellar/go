-- +migrate Up

CREATE INDEX "trust_lines_by_account_type_code_issuer" ON trust_lines USING btree (account_id, asset_type, asset_code, asset_issuer);

-- +migrate Down

DROP INDEX "trust_lines_by_account_type_code_issuer";
