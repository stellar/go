/*
Package txnbuild implements transactions and operations on the Stellar network.
This library provides an interface to the Stellar transaction model. It supports the building of Go applications on
top of the Stellar network (https://www.stellar.org/). Transactions constructed by this library may be submitted
to any Horizon instance for processing onto the ledger, using any Stellar SDK client. The recommended client for Go
programmers is horizonclient (https://github.com/stellar/go/tree/master/clients/horizonclient). Together, these two
libraries provide a complete Stellar SDK.
For more information and further examples, see https://www.stellar.org/developers/go/reference/index.html.
*/
package txnbuild

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Account represents the aspects of a Stellar account necessary to construct transactions. See
// https://www.stellar.org/developers/guides/concepts/accounts.html
type Account interface {
	GetAccountID() string
	IncrementSequenceNumber() (xdr.SequenceNumber, error)
	// Action needed in release: horizonclient-v2.0.0
	// add GetSequenceNumber method
	// GetSequenceNumber() (xdr.SequenceNumber, error)
}

// Transaction represents a Stellar transaction. See
// https://www.stellar.org/developers/guides/concepts/transactions.html
type Transaction struct {
	SourceAccount  Account
	Operations     []Operation
	BaseFee        uint32
	Memo           Memo
	Timebounds     Timebounds
	Network        string
	xdrTransaction xdr.Transaction
	xdrEnvelope    *xdr.TransactionEnvelope
}

// Hash provides a signable object representing the Transaction on the specified network.
func (tx *Transaction) Hash() ([32]byte, error) {
	return network.HashTransaction(&tx.xdrTransaction, tx.Network)
}

// MarshalBinary returns the binary XDR representation of the transaction envelope.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	var txBytes bytes.Buffer
	_, err := xdr.Marshal(&txBytes, tx.xdrEnvelope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal XDR")
	}

	return txBytes.Bytes(), nil
}

// Base64 returns the base 64 XDR representation of the transaction envelope.
func (tx *Transaction) Base64() (string, error) {
	bs, err := tx.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "failed to get XDR bytestring")
	}

	return base64.StdEncoding.EncodeToString(bs), nil
}

// SetDefaultFee sets a sensible minimum default for the Transaction fee, if one has not
// already been set. It is a linear function of the number of Operations in the Transaction.
// Deprecated: This will be removed in v2.0.0 and setting `Transaction.BaseFee` will be mandatory.
// Action needed in release: horizonclient-v2.0.0
func (tx *Transaction) SetDefaultFee() {
	// TODO: Generalise to pull this from a client call
	var DefaultBaseFee uint32 = 100
	if tx.BaseFee == 0 {
		tx.BaseFee = DefaultBaseFee
	}

	err := tx.setTransactionFee()
	if err != nil {
		panic(err)
	}
}

// Build for Transaction completely configures the Transaction. After calling Build,
// the Transaction is ready to be serialised or signed.
func (tx *Transaction) Build() error {
	// to do: check if tx is already signed

	accountID := tx.SourceAccount.GetAccountID()
	// Public keys start with 'G'
	if accountID[0] != 'G' {
		return errors.New("invalid public key for transaction source account")
	}
	_, err := keypair.Parse(accountID)
	if err != nil {
		return err
	}

	// Set account ID in XDR
	tx.xdrTransaction.SourceAccount.SetAddress(accountID)

	// Action needed in release: horizonclient-v2.0.0
	// Validate Seq Num is present in struct. Requires Account.GetSequenceNumber (v.2.0.0)
	seqnum, err := tx.SourceAccount.IncrementSequenceNumber()
	if err != nil {
		return errors.Wrap(err, "failed to parse sequence number")
	}
	tx.xdrTransaction.SeqNum = seqnum

	for _, op := range tx.Operations {
		xdrOperation, err2 := op.BuildXDR()
		if err2 != nil {
			return errors.Wrap(err2, fmt.Sprintf("failed to build operation %T", op))
		}
		tx.xdrTransaction.Operations = append(tx.xdrTransaction.Operations, xdrOperation)
	}

	// Check and set the timebounds
	err = tx.Timebounds.Validate()
	if err != nil {
		return err
	}
	tx.xdrTransaction.TimeBounds = &xdr.TimeBounds{MinTime: xdr.TimePoint(tx.Timebounds.MinTime),
		MaxTime: xdr.TimePoint(tx.Timebounds.MaxTime)}

	// Handle the memo, if one is present
	if tx.Memo != nil {
		xdrMemo, err := tx.Memo.ToXDR()
		if err != nil {
			return errors.Wrap(err, "couldn't build memo XDR")
		}
		tx.xdrTransaction.Memo = xdrMemo
	}

	// Set a default fee, if it hasn't been set yet
	// Action needed in release: horizonclient-v2.0.0
	// replace with tx.setTransactionfee
	tx.SetDefaultFee()

	// Initialise transaction envelope
	if tx.xdrEnvelope == nil {
		tx.xdrEnvelope = &xdr.TransactionEnvelope{}
		tx.xdrEnvelope.Tx = tx.xdrTransaction
	}

	return nil
}

// Sign for Transaction signs a previously built transaction. A signed transaction may be
// submitted to the network.
func (tx *Transaction) Sign(kps ...*keypair.Full) error {
	// TODO: Only sign if Transaction has been previously built
	// TODO: Validate network set before sign

	// Hash the transaction
	hash, err := tx.Hash()
	if err != nil {
		return errors.Wrap(err, "failed to hash transaction")
	}

	// Sign the hash
	for _, kp := range kps {
		sig, err := kp.SignDecorated(hash[:])
		if err != nil {
			return errors.Wrap(err, "failed to sign transaction")
		}
		// Append the signature to the envelope
		tx.xdrEnvelope.Signatures = append(tx.xdrEnvelope.Signatures, sig)
	}

	return nil
}

// BuildSignEncode performs all the steps to produce a final transaction suitable
// for submitting to the network.
func (tx *Transaction) BuildSignEncode(keypairs ...*keypair.Full) (string, error) {
	err := tx.Build()
	if err != nil {
		return "", errors.Wrap(err, "couldn't build transaction")
	}

	err = tx.Sign(keypairs...)
	if err != nil {
		return "", errors.Wrap(err, "couldn't sign transaction")
	}

	txeBase64, err := tx.Base64()
	if err != nil {
		return "", errors.Wrap(err, "couldn't encode transaction")
	}

	return txeBase64, err
}

// BuildChallengeTx is a factory method that creates a valid SEP 10 challenge, for use in web authentication.
// "timebound" is the time duration the transaction should be valid for, O means infinity.
// More details on SEP 10: https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0010.md
func BuildChallengeTx(serverSignerSecret, clientAccountID, anchorName, network string, timebound time.Duration) (string, error) {
	serverKP, err := keypair.Parse(serverSignerSecret)
	if err != nil {
		return "", err
	}

	// SEP10 spec requires 48 byte cryptographic-quality random string
	randomNonce, err := generateRandomNonce(48)
	if err != nil {
		return "", err
	}
	// Encode 48-byte nonce to base64 for a total of 64-bytes
	randomNonceToString := base64.StdEncoding.EncodeToString(randomNonce)
	if len(randomNonceToString) != 64 {
		return "", errors.New("64 byte long random nonce required")
	}

	// represent server signing account as SimpleAccount
	sa := SimpleAccount{
		AccountID: serverKP.Address(),
		// Action needed in release: v2.0.0
		// TODO: remove this and use "Sequence: 0" and build transaction with optional argument
		//  (https://github.com/stellar/go/issues/1259)
		Sequence: int64(-1),
	}

	// represent client account as SimpleAccount
	ca := SimpleAccount{
		AccountID: clientAccountID,
	}

	txTimebound := NewInfiniteTimeout()
	if timebound > 0 {
		currentTime := time.Now().UTC()
		maxTime := currentTime.Add(timebound)
		txTimebound = NewTimebounds(currentTime.Unix(), maxTime.Unix())
	}

	// Create a SEP 10 compatible response. See
	// https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0010.md#response
	tx := Transaction{
		SourceAccount: &sa,
		Operations: []Operation{
			&ManageData{
				SourceAccount: &ca,
				Name:          anchorName + " auth",
				Value:         []byte(randomNonceToString),
			},
		},
		Timebounds: txTimebound,
		Network:    network,
		BaseFee:    uint32(100),
	}

	txeB64, err := tx.BuildSignEncode(serverKP.(*keypair.Full))
	if err != nil {
		return "", err
	}
	return txeB64, nil
}

// generateRandomNonce creates a cryptographically secure random slice of `n` bytes.
func generateRandomNonce(n int) ([]byte, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)

	if err != nil {
		return []byte{}, err
	}

	return bytes, err
}

// HashHex returns the hex-encoded hash of the transaction.
func (tx *Transaction) HashHex() (string, error) {
	hashByte, err := tx.Hash()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hashByte[:]), nil
}

// TxEnvelope returns the TransactionEnvelope XDR struct.
func (tx *Transaction) TxEnvelope() *xdr.TransactionEnvelope {
	return tx.xdrEnvelope
}

func (tx *Transaction) setTransactionFee() error {
	if tx.BaseFee == 0 {
		return errors.New("base fee can not be zero")
	}

	tx.xdrTransaction.Fee = xdr.Uint32(int(tx.BaseFee) * len(tx.xdrTransaction.Operations))
	return nil
}

// TransactionFee returns the fee to be paid for a transaction.
func (tx *Transaction) TransactionFee() int {
	err := tx.setTransactionFee()
	// error is returned when BaseFee is zero
	if err != nil {
		return 0
	}
	return int(tx.xdrTransaction.Fee)
}

// SignHashX signs a transaction with HashX signature type.
// See description here: https://www.stellar.org/developers/guides/concepts/multi-sig.html#hashx.
func (tx *Transaction) SignHashX(preimage []byte) error {
	if tx.xdrEnvelope == nil {
		tx.xdrEnvelope = &xdr.TransactionEnvelope{}
		tx.xdrEnvelope.Tx = tx.xdrTransaction
	}

	if len(preimage) > xdr.Signature(preimage).XDRMaxSize() {
		return errors.New("preimage cannnot be more than 64 bytes")
	}

	preimageHash := sha256.Sum256(preimage)
	var hint [4]byte
	// copy the last 4-bytes of the signer public key to be used as hint
	copy(hint[:], preimageHash[28:])

	sig := xdr.DecoratedSignature{
		Hint:      xdr.SignatureHint(hint),
		Signature: xdr.Signature(preimage),
	}

	tx.xdrEnvelope.Signatures = append(tx.xdrEnvelope.Signatures, sig)

	return nil
}

// TransactionFromXDR parses the supplied transaction envelope in base64 XDR and returns a Transaction object.
func TransactionFromXDR(txeB64 string) (Transaction, error) {
	var xdrEnv xdr.TransactionEnvelope
	err := xdr.SafeUnmarshalBase64(txeB64, &xdrEnv)
	if err != nil {
		return Transaction{}, errors.Wrap(err, "unable to unmarshal transaction envelope")
	}

	var newTx Transaction
	newTx.xdrTransaction = xdrEnv.Tx
	newTx.xdrEnvelope = &xdrEnv

	if len(xdrEnv.Tx.Operations) > 0 {
		newTx.BaseFee = uint32(xdrEnv.Tx.Fee) / uint32(len(xdrEnv.Tx.Operations))
	}

	newTx.SourceAccount = &SimpleAccount{
		AccountID: xdrEnv.Tx.SourceAccount.Address(),
		Sequence:  int64(xdrEnv.Tx.SeqNum),
	}

	if xdrEnv.Tx.TimeBounds != nil {
		newTx.Timebounds = NewTimebounds(int64(xdrEnv.Tx.TimeBounds.MinTime), int64(xdrEnv.Tx.TimeBounds.MaxTime))
	}

	newTx.Memo, err = memoFromXDR(xdrEnv.Tx.Memo)
	if err != nil {
		return Transaction{}, errors.Wrap(err, "unable to parse memo")
	}

	for _, op := range xdrEnv.Tx.Operations {
		newOp, err := operationFromXDR(op)
		if err != nil {
			return Transaction{}, err
		}
		newTx.Operations = append(newTx.Operations, newOp)
	}

	return newTx, nil
}
