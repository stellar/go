package compliance

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/stellar/go/address"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Attachment returns attachment from the the object.
func (d AuthData) Attachment() (attachment attachment.Attachment, err error) {
	err = json.Unmarshal([]byte(d.AttachmentJSON), &attachment)
	return
}

// AttachmentPreimageHash returns sha-256 hash of memo preimage.
// The hash is base64 encoded.
func (d AuthData) AttachmentPreimageHash() string {
	attachmentPreimageHashBytes := sha256.Sum256([]byte(d.AttachmentJSON))
	return base64.StdEncoding.EncodeToString(attachmentPreimageHashBytes[:])
}

// Validate checks if fields are of required form:
//  * `Sender` field is valid address
//  * `Tx` is valid and it's memo_hash equals sha256 hash of attachment preimage
//  * `Attachment` is valid JSON
func (d AuthData) Validate() error {
	_, _, err := address.Split(d.Sender)
	if err != nil {
		return errors.New("Invalid Data.Sender value")
	}

	var tx xdr.Transaction
	err = xdr.SafeUnmarshalBase64(d.Tx, &tx)
	if err != nil {
		return errors.New("Tx is invalid")
	}

	if tx.Memo.Hash == nil {
		return errors.New("Memo.Hash is nil")
	}

	// Check if Memo.Hash is sha256 hash of attachment preimage
	attachmentPreimageHashBytes := sha256.Sum256([]byte(d.AttachmentJSON))
	memoBytes := [32]byte(*tx.Memo.Hash)
	if attachmentPreimageHashBytes != memoBytes {
		return errors.New("Attachment preimage hash does not equal Memo.Hash in Tx")
	}

	// Check if d.AttachmentJSON is valid JSON
	attachment := Attachment{}
	err = json.Unmarshal([]byte(d.AttachmentJSON), &attachment)
	if err != nil {
		return errors.New("Attachment is not valid JSON")
	}

	return nil
}
