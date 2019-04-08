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
	qs += "VALUES (" + generatePlaceholders(dbValues) + ")"
	qs += createOnConflictFragment("assets_code_issuer_key", toUpdateFields)
	_, err = s.ExecRaw(qs, dbValues...)
	return
}
