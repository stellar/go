package history

import (
	"encoding/base64"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// IsAuthorized returns true if issuer has authorized account to perform
// transactions with its credit
func (trustLine TrustLine) IsAuthorized() bool {
	return xdr.TrustLineFlags(trustLine.Flags).IsAuthorized()
}

func (q *Q) CountTrustLines() (int, error) {
	sql := sq.Select("count(*)").From("trust_lines")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

func (q *Q) GetTrustLinesByAccountID(id string) ([]TrustLine, error) {
	var trustLines []TrustLine
	sql := selectTrustLines.Where(sq.Eq{"account_id": id})
	err := q.Select(&trustLines, sql)
	return trustLines, err
}

// GetTrustLinesByKeys loads a row from the `trust_lines` table, selected by multiple keys.
func (q *Q) GetTrustLinesByKeys(keys []xdr.LedgerKeyTrustLine) ([]TrustLine, error) {
	var trustLines []TrustLine
	lkeys := make([]string, 0, len(keys))
	for _, key := range keys {
		lkey, err := ledgerKeyTrustLineToString(key)
		if err != nil {
			return nil, errors.Wrap(err, "Error running ledgerKeyTrustLineToString")
		}
		lkeys = append(lkeys, lkey)
	}
	sql := selectTrustLines.Where(map[string]interface{}{"trust_lines.ledger_key": lkeys})
	err := q.Select(&trustLines, sql)
	return trustLines, err
}

// InsertTrustLine creates a row in the trust lines table.
// Returns number of rows affected and error.
func (q *Q) InsertTrustLine(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	m := trustLineToMap(trustLine, lastModifiedLedger)

	// Add lkey only when inserting rows
	key, err := trustLineEntryToLedgerKeyString(trustLine)
	if err != nil {
		return 0, errors.Wrap(err, "Error running trustLineEntryToLedgerKeyString")
	}
	m["ledger_key"] = key

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
	ledgerKey := xdr.LedgerKey{}
	err := ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
	if err != nil {
		return 0, errors.Wrap(err, "Error creating ledger key")
	}

	key, err := trustLineEntryToLedgerKeyString(trustLine)
	if err != nil {
		return 0, errors.Wrap(err, "Error running trustLineEntryToLedgerKeyString")
	}

	sql := sq.Update("trust_lines").
		SetMap(trustLineToMap(trustLine, lastModifiedLedger)).
		Where(map[string]interface{}{"ledger_key": key})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveTrustLine deletes a row in the trust lines table.
// Returns number of rows affected and error.
func (q *Q) RemoveTrustLine(ledgerKey xdr.LedgerKeyTrustLine) (int64, error) {
	key, err := ledgerKeyTrustLineToString(ledgerKey)
	if err != nil {
		return 0, errors.Wrap(err, "Error ledgerKeyTrustLineToString MarshalBinaryCompress")
	}

	sql := sq.Delete("trust_lines").
		Where(map[string]interface{}{"ledger_key": key})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetTrustLinesByAccountsID loads trust lines for a list of accounts ID
func (q *Q) GetTrustLinesByAccountsID(id []string) ([]TrustLine, error) {
	var data []TrustLine
	sql := selectTrustLines.Where(sq.Eq{"account_id": id})
	err := q.Select(&data, sql)
	return data, err
}

func trustLineEntryToLedgerKeyString(trustLine xdr.TrustLineEntry) (string, error) {
	ledgerKey := &xdr.LedgerKey{}
	err := ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
	if err != nil {
		return "", errors.Wrap(err, "Error running ledgerKey.SetTrustline")
	}
	key, err := ledgerKey.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Error running MarshalBinaryCompress")
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func ledgerKeyTrustLineToString(trustLineKey xdr.LedgerKeyTrustLine) (string, error) {
	ledgerKey := &xdr.LedgerKey{}
	err := ledgerKey.SetTrustline(trustLineKey.AccountId, trustLineKey.Asset)
	if err != nil {
		return "", errors.Wrap(err, "Error running ledgerKey.SetTrustline")
	}
	key, err := ledgerKey.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Error running MarshalBinaryCompress")
	}

	return base64.StdEncoding.EncodeToString(key), nil
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
		"account_id":           trustLine.AccountId.Address(),
		"asset_type":           assetType,
		"asset_issuer":         assetIssuer,
		"asset_code":           assetCode,
		"balance":              trustLine.Balance,
		"trust_line_limit":     trustLine.Limit,
		"buying_liabilities":   buyingliabilities,
		"selling_liabilities":  sellingliabilities,
		"flags":                trustLine.Flags,
		"last_modified_ledger": lastModifiedLedger,
	}
}

var selectTrustLines = sq.Select(`
	account_id,
	asset_type,
	asset_issuer,
	asset_code,
	balance,
	trust_line_limit,
	buying_liabilities,
	selling_liabilities,
	flags,
	last_modified_ledger
`).From("trust_lines")
