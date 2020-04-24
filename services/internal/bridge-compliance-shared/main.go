package shared

import (
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

// BuildTransaction is used in compliance server. The sequence number in built transaction will be equal 0!
func BuildTransaction(accountID, networkPassphrase string, operation []txnbuild.Operation, memo txnbuild.Memo) (string, error) {

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: accountID, Sequence: 0},
			IncrementSequenceNum: false,
			Operations:           operation,
			BaseFee:              txnbuild.MinBaseFee,
			Memo:                 memo,
			Timebounds:           txnbuild.NewInfiniteTimeout(),
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "unable to build transaction")
	}
	txeB64, err := tx.Base64()
	if err != nil {
		return "", errors.Wrap(err, "unable to serialize transaction")
	}

	var txXDR xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(txeB64, &txXDR)
	if err != nil {
		return "", errors.Wrap(err, "unable to decode transaction envelope")
	}
	txB64, err := xdr.MarshalBase64(txXDR.V0.Tx)
	if err != nil {
		return "", errors.Wrap(err, "unable to encode transaction")
	}

	return txB64, err
}

// IsValidAccountID returns true if account ID is valid
func IsValidAccountID(accountID string) bool {
	_, err := keypair.Parse(accountID)
	if err != nil {
		return false
	}

	if accountID[0] != 'G' {
		return false
	}

	return true
}

// IsValidSecret returns true if secret is valid
func IsValidSecret(secret string) bool {
	_, err := keypair.Parse(secret)
	if err != nil {
		return false
	}

	if secret[0] != 'S' {
		return false
	}

	return true
}

// IsValidAssetCode returns true if asset code is valid
func IsValidAssetCode(code string) bool {
	if len(code) < 1 || len(code) > 12 {
		return false
	}
	return true
}
