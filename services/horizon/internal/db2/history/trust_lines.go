package history

import (
	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (q *Q) CountTrustLines() (int, error) {
	sql := sq.Select("count(*)").From("trust_lines")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// InsertTrustLine creates a row in the trust lines table.
// Returns number of rows affected and error.
func (q *Q) InsertTrustLine(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	m := trustLineToMap(trustLine, lastModifiedLedger)
	sql := sq.Insert("trust_lines").SetMap(m)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// UpdateTrustLine updates a row in the trust lines table.
// Returns number of rows affected and error.
func (q *Q) UpdateTrustLine(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	m := trustLineToMap(trustLine, lastModifiedLedger)

	ledgerKey := xdr.LedgerKey{}
	err := ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
	if err != nil {
		return 0, errors.Wrap(err, "Error creating ledger key")
	}

	where := ledgerKeyTrustLineToMap(*ledgerKey.TrustLine)
	for key := range where {
		delete(m, key)
	}

	sql := sq.Update("trust_lines").SetMap(m).Where(where)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveTrustLine deletes a row in the trust lines table.
// Returns number of rows affected and error.
func (q *Q) RemoveTrustLine(key xdr.LedgerKeyTrustLine) (int64, error) {
	where := ledgerKeyTrustLineToMap(key)
	sql := sq.Delete("trust_lines").Where(where)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func ledgerKeyTrustLineToMap(key xdr.LedgerKeyTrustLine) map[string]interface{} {
	var assetType xdr.AssetType
	var assetCode, assetIssuer string
	key.Asset.MustExtract(&assetType, &assetCode, &assetIssuer)

	return map[string]interface{}{
		"accountid":   key.AccountId.Address(),
		"assettype":   assetType,
		"assetissuer": assetIssuer,
		"assetcode":   assetCode,
	}
}

func trustLineToMap(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) map[string]interface{} {
	var assetType xdr.AssetType
	var assetCode, assetIssuer string
	trustLine.Asset.MustExtract(&assetType, &assetCode, &assetIssuer)

	var buyingliabilities, sellingliabilities xdr.Int64
	if trustLine.Ext.V1 != nil {
		v1 := trustLine.Ext.V1
		buyingliabilities = v1.Liabilities.Buying
		sellingliabilities = v1.Liabilities.Selling
	}

	return map[string]interface{}{
		"accountid":            trustLine.AccountId.Address(),
		"assettype":            assetType,
		"assetissuer":          assetIssuer,
		"assetcode":            assetCode,
		"balance":              trustLine.Balance,
		"tlimit":               trustLine.Limit,
		"buyingliabilities":    buyingliabilities,
		"sellingliabilities":   sellingliabilities,
		"flags":                trustLine.Flags,
		"last_modified_ledger": lastModifiedLedger,
	}
}
