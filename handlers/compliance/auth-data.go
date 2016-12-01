package compliance

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/stellar/go/address"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Memo returns Memo from the the object. This method
// assumes that Memo is valid JSON (checked in Validate).
func (d *AuthData) Memo() (memo *Memo) {
	json.Unmarshal([]byte(d.MemoJSON), &memo)
	return
}

// Validate checks if fields are of required form:
//  * `Sender` field is valid address
//  * `Tx` is valid and it's memo_hash equals sha256 hash of memo preimage
//  * `Memo` is valid JSON
func (d *AuthData) Validate() error {
	_, _, err := address.Split(d.Sender)
	if err != nil {
		return errors.Wrap(err, "Invalid Data.Sender value")
	}

	var tx xdr.Transaction
	err = xdr.SafeUnmarshalBase64(d.Tx, &tx)
	if err != nil {
		return errors.Wrap(err, "Tx is invalid")
	}

	if tx.Memo.Hash == nil {
		return errors.New("Memo.Hash is nil")
	}

	// Check if Memo.Hash is sha256 hash of memo preimage
	memoPreimageHashBytes := sha256.Sum256([]byte(d.MemoJSON))
	memoBytes := [32]byte(*tx.Memo.Hash)
	if memoPreimageHashBytes != memoBytes {
		return errors.New("Memo preimage hash does not equal Memo.Hash in Tx")
	}

	// Check if d.Memo is valid JSON
	memo := Memo{}
	err = json.Unmarshal([]byte(d.MemoJSON), &memo)
	if err != nil {
		return errors.Wrap(err, "Memo is not valid JSON")
	}

	return nil
}
