package tickerdb

// InsertAsset inserts a new Asset into the database
func (s *TickerSession) InsertAsset(a *Asset) (err error) {
	tbl := s.GetTable("assets")
	_, err = tbl.Insert(a).IgnoreCols("id").Exec()
	return
}

// InsertOrUpdateAsset inserts an Asset on the database (if new),
// or updates an existing one
func (s *TickerSession) InsertOrUpdateAsset(a *Asset) (err error) {
	_, err = s.ExecRaw(
		rawUpsertQuery,
		a.Code, a.Issuer, a.Type, a.NumAccounts, a.AuthRequired, a.AuthRevocable,
		a.Amount, a.AssetControlledByDomain, a.AnchorAssetCode, a.AnchorAssetType,
		a.IsValid, a.ValidationError, a.LastValid, a.LastChecked,
	)
	return
}

var rawUpsertQuery = `
INSERT INTO assets (
	code,
	issuer,
	type,
	num_accounts,
	auth_required,
	auth_revocable,
	amount,
	asset_controlled_by_domain,
	anchor_asset_code,
	anchor_asset_type,
	is_valid,
	validation_error,
	last_valid,
	last_checked
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT ON CONSTRAINT assets_code_issuer_key
DO
	UPDATE
		SET
			type = EXCLUDED.type,
			num_accounts = EXCLUDED.num_accounts,
			auth_required = EXCLUDED.auth_required,
			auth_revocable = EXCLUDED.auth_revocable,
			amount = EXCLUDED.amount,
			asset_controlled_by_domain = EXCLUDED.asset_controlled_by_domain,
			anchor_asset_code = EXCLUDED.anchor_asset_code,
			anchor_asset_type = EXCLUDED.anchor_asset_type,
			is_valid = EXCLUDED.is_valid,
			validation_error = EXCLUDED.validation_error,
			last_valid = EXCLUDED.last_valid,
			last_checked = EXCLUDED.last_checked;
`
