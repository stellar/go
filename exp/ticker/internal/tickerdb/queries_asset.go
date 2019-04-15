package tickerdb

import (
	"strings"

	"github.com/stellar/go/exp/ticker/internal/utils"
)

// InsertAsset inserts a new Asset into the database
func (s *TickerSession) InsertAsset(a *Asset) (err error) {
	tbl := s.GetTable("assets")
	_, err = tbl.Insert(a).IgnoreCols("id").Exec()
	return
}

// InsertOrUpdateAsset inserts an Asset on the database (if new),
// or updates an existing one
func (s *TickerSession) InsertOrUpdateAsset(a *Asset, preserveFields []string) (err error) {
	dbFields := getDBFieldTags(*a, true)
	dbFieldsString := strings.Join(dbFields, ", ")
	dbValues := getDBFieldValues(*a, true)

	cleanPreservedFields := sanitizeFieldNames(preserveFields)
	toUpdateFields := utils.SliceDiff(dbFields, cleanPreservedFields)

	qs := "INSERT INTO assets (" + dbFieldsString + ")"
	qs += " VALUES (" + generatePlaceholders(dbValues) + ")"
	qs += " " + createOnConflictFragment("assets_code_issuer_account", toUpdateFields) + ";"
	_, err = s.ExecRaw(qs, dbValues...)
	return
}

// GetAssetByCodeAndIssuerAccount searches for an Asset with the given code
// and public key, and returns its ID in case it is found.
func (s *TickerSession) GetAssetByCodeAndIssuerAccount(
	code string,
	issuerAccount string,
) (found bool, id int32, err error) {
	var assets []Asset
	tbl := s.GetTable("assets")

	err = tbl.Select(
		&assets,
		"assets.code = ? AND assets.issuer_account = ?",
		code,
		issuerAccount,
	).Exec()
	if err != nil {
		return
	}

	if len(assets) > 0 {
		id = assets[0].ID
		found = true
	}
	return
}

// GetAllValidAssets returns a slice with all assets in the database
// with is_valid = true
func (s *TickerSession) GetAllValidAssets() (assets []Asset, err error) {
	tbl := s.GetTable("assets")

	err = tbl.Select(
		&assets,
		"assets.is_valid = TRUE",
	).Exec()

	return
}
