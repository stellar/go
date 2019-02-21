package core

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// SignersByAddress loads all signer rows for `addy`
func (q *Q) SignersByAddress(dest interface{}, addy string) error {
	schemaVersion, err := q.SchemaVersion()
	if err != nil {
		return err
	}

	if schemaVersion >= 9 {
		var signersXDRString string
		sql := selectSignerVersion9.Where("accountid = ?", addy)
		err2 := q.Get(&signersXDRString, sql)
		if err2 != nil {
			return err2
		}

		var signersXDR []xdr.Signer
		err2 = xdr.SafeUnmarshalBase64(signersXDRString, &signersXDR)
		if err2 != nil {
			return errors.Wrap(err2, "Error decoding []xdr.Signer")
		}

		signers := make([]Signer, 0, len(signersXDR))
		for _, signer := range signersXDR {
			signers = append(signers, Signer{
				Accountid: addy,
				Publickey: signer.Key.Address(),
				Weight:    int32(signer.Weight),
			})
		}

		*dest.(*[]Signer) = signers
		return nil
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
