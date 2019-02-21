package core

import (
	sq "github.com/Masterminds/squirrel"
)

// SignersByAddress loads all signer rows for `addy`
func (q *Q) SignersByAddress(dest interface{}, addy string) error {
	schemaVersion, err := q.SchemaVersion()
	if err != nil {
		return err
	}

	if schemaVersion >= 9 {
		result := struct {
			signers string `db:"signers"`
		}{}
		sql := selectSignerVersion9.Where("accountid = ?", addy)
		return q.Select(&result, sql)

		// TODO xdr decode signers
	} else {
		sql := selectSigner.Where("accountid = ?", addy)
		return q.Select(dest, sql)
	}
}

var selectSigner = sq.Select(
	"si.accountid",
	"si.publickey",
	"si.weight",
).From("signers si")

var selectSignerVersion9 = sq.Select("a.signers").From("accounts a")
