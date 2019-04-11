/*
Package txnbuild implements transactions and operations on the Stellar network.
TODO: More explanation + links here
*/
package txnbuild

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TXNAccount represents a Stellar TXNAccount from the perspective of a Transaction.
type TXNAccount struct {
	ID             string
	SequenceNumber string
}

// GetAccountID is to be deleted once refactor is complete
func (t *TXNAccount) GetAccountID() string {
	return t.ID
}

// GetSequenceNumber is to be deleted once refactor is complete
func (t *TXNAccount) GetSequenceNumber() (xdr.SequenceNumber, error) {
	seqNum, err := strconv.ParseUint(t.SequenceNumber, 10, 64)

	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse account sequence number")
	}

	return xdr.SequenceNumber(seqNum), nil
}

// IncrementSequenceNumber is to be deleted once refactor is complete
func (t *TXNAccount) IncrementSequenceNumber() error {
	seqNum, err := t.GetSequenceNumber()
	if err != nil {
		return err
	}
	seqNum++
	t.SequenceNumber = strconv.FormatInt(int64(seqNum), 10)
	return nil
}

// Account represents the aspects of a Stellar account necessary to construct transactions.
type Account interface {
	GetAccountID() string
	GetSequenceNumber() (xdr.SequenceNumber, error)
	IncrementSequenceNumber() error
}

// Transaction represents a Stellar Transaction.
type Transaction struct {
	SourceAccount  TXNAccount
	Operations     []Operation
	xdrTransaction xdr.Transaction
	BaseFee        uint64 // TODO: Why is this a uint 64? Can it be a plain int?
	xdrEnvelope    *xdr.TransactionEnvelope
	Network        string
}

// Hash provides a signable object representing the Transaction on the specified network.
func (tx *Transaction) Hash() ([32]byte, error) {
	return network.HashTransaction(&tx.xdrTransaction, tx.Network)
}

// MarshalBinary returns the binary XDR representation of the Transaction.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	var txBytes bytes.Buffer
	_, err := xdr.Marshal(&txBytes, tx.xdrEnvelope)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal XDR")
	}

	return txBytes.Bytes(), nil
}

// Base64 returns the base 64 XDR representation of the Transaction.
func (tx *Transaction) Base64() (string, error) {
	bs, err := tx.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get XDR bytestring")
	}

	return base64.StdEncoding.EncodeToString(bs), nil
}

// SetDefaultFee sets a sensible minimum default for the Transaction fee, if one has not
// already been set. It is a linear function of the number of Operations in the Transaction.
func (tx *Transaction) SetDefaultFee() {
	// TODO: Check if default base fee used elsewhere - otherwise just use int
	// TODO: Generalise to pull this from a client call
	var DefaultBaseFee uint64 = 100
	if tx.BaseFee == 0 {
		tx.BaseFee = DefaultBaseFee
	}
	if tx.xdrTransaction.Fee == 0 {
		tx.xdrTransaction.Fee = xdr.Uint32(int(tx.BaseFee) * len(tx.xdrTransaction.Operations))
	}
}

// Build for Transaction completely configures the Transaction. After calling Build,
// the Transaction is ready to be serialised or signed.
func (tx *Transaction) Build() error {
	// Set account ID in XDR
	// TODO: Validate provided key before going further
	tx.xdrTransaction.SourceAccount.SetAddress(tx.SourceAccount.ID)

	// TODO: Validate Seq Num is present in struct
	tx.SourceAccount.IncrementSequenceNumber()
	var err error
	tx.xdrTransaction.SeqNum, err = tx.SourceAccount.GetSequenceNumber()
	if err != nil {
		return errors.Wrap(err, "Failed to parse sequence number")
	}

	for _, op := range tx.Operations {
		xdrOperation, err := op.BuildXDR()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to build operation %T", op))
		}
		tx.xdrTransaction.Operations = append(tx.xdrTransaction.Operations, xdrOperation)
	}

	// Set a default fee, if it hasn't been set yet
	tx.SetDefaultFee()

	return nil
}

// Sign for Transaction signs a previously built transaction. A signed transaction may be
// submitted to the network.
func (tx *Transaction) Sign(kp *keypair.Full) error {
	// TODO: Only sign if Transaction has been previously built
	// TODO: Validate network set before sign
	// Initialise transaction envelope
	if tx.xdrEnvelope == nil {
		tx.xdrEnvelope = &xdr.TransactionEnvelope{}
		tx.xdrEnvelope.Tx = tx.xdrTransaction
	}

	// Hash the transaction
	hash, err := tx.Hash()
	if err != nil {
		return errors.Wrap(err, "Failed to hash transaction")
	}

	// Sign the hash
	// TODO: Allow multiple signers
	sig, err := kp.SignDecorated(hash[:])
	if err != nil {
		return errors.Wrap(err, "Failed to sign transaction")
	}

	// Append the signature to the envelope
	tx.xdrEnvelope.Signatures = append(tx.xdrEnvelope.Signatures, sig)

	return nil
}
