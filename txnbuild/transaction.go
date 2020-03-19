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
	"strings"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/strkey"
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
	xdrEnvelope    *xdr.TransactionV1Envelope
}

// Hash provides a signable object representing the Transaction on the specified network.
func (tx *Transaction) Hash() ([32]byte, error) {
	return network.HashTransaction(&tx.xdrTransaction, tx.Network)
}

// MarshalBinary returns the binary XDR representation of the transaction envelope.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	var txBytes bytes.Buffer
	_, err := xdr.Marshal(&txBytes, tx.TxEnvelope())
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
	// If transaction envelope has been signed, don't build transaction
	if tx.xdrEnvelope != nil {
		if len(tx.xdrEnvelope.Signatures) > 0 {
			return errors.New("transaction has already been signed, so cannot be rebuilt.")
		}
		// clear the existing XDR so we don't append to any existing fields
		tx.xdrEnvelope = &xdr.TransactionV1Envelope{}
	}

	// reset tx.xdrTransaction
	tx.xdrTransaction = xdr.Transaction{}

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
		if verr := op.Validate(); verr != nil {
			return errors.Wrap(verr, fmt.Sprintf("validation failed for %T operation", op))
		}

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
	tx.xdrEnvelope = &xdr.TransactionV1Envelope{
		Tx: tx.xdrTransaction,
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
// "timebound" is the time duration the transaction should be valid for, and must be greater than 1s (300s is recommended).
// More details on SEP 10: https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0010.md
func BuildChallengeTx(serverSignerSecret, clientAccountID, anchorName, network string, timebound time.Duration) (string, error) {

	if timebound < time.Second {
		return "", errors.New("provided timebound must be at least 1s (300s is recommended)")
	}

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

	currentTime := time.Now().UTC()
	maxTime := currentTime.Add(timebound)
	txTimebound := NewTimebounds(currentTime.Unix(), maxTime.Unix())

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
	return &xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1:   tx.xdrEnvelope,
	}
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
		tx.xdrEnvelope = &xdr.TransactionV1Envelope{}
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

	if xdrEnv.IsFeeBump() {
		return Transaction{}, errors.New("fee bump transactions are not supported")
	}
	var newTx Transaction
	newTx.xdrTransaction = xdr.Transaction{
		SourceAccount: xdrEnv.SourceAccount(),
		Fee:           xdr.Uint32(xdrEnv.Fee()),
		Memo:          xdrEnv.Memo(),
		Operations:    xdrEnv.Operations(),
		SeqNum:        xdr.SequenceNumber(xdrEnv.SeqNum()),
		TimeBounds:    xdrEnv.TimeBounds(),
	}
	newTx.xdrEnvelope = &xdr.TransactionV1Envelope{
		Tx: newTx.xdrTransaction,
	}
	// only include signatures if the transaction is a V1 transaction
	// fee bump and V0 transactions have different hashes from V1 transactions
	if xdrEnv.Type == xdr.EnvelopeTypeEnvelopeTypeTx {
		newTx.xdrEnvelope.Signatures = xdrEnv.Signatures()
	}
	if numOps := len(xdrEnv.Operations()); numOps > 0 {
		newTx.BaseFee = xdrEnv.Fee() / uint32(numOps)
	}

	sourceAccount := xdrEnv.SourceAccount()
	newTx.SourceAccount = &SimpleAccount{
		AccountID: sourceAccount.Address(),
		Sequence:  xdrEnv.SeqNum(),
	}

	if timeBounds := xdrEnv.TimeBounds(); timeBounds != nil {
		newTx.Timebounds = NewTimebounds(int64(timeBounds.MinTime), int64(timeBounds.MaxTime))
	}

	newTx.Memo, err = memoFromXDR(xdrEnv.Memo())
	if err != nil {
		return Transaction{}, errors.Wrap(err, "unable to parse memo")
	}

	for _, op := range xdrEnv.Operations() {
		newOp, err := operationFromXDR(op)
		if err != nil {
			return Transaction{}, err
		}
		newTx.Operations = append(newTx.Operations, newOp)
	}

	return newTx, nil
}

// SignWithKeyString for Transaction signs a previously built transaction with the secret key
// as a string. This can be used when you don't have access to a Stellar keypair.
// A signed transaction may be submitted to the network.
func (tx *Transaction) SignWithKeyString(keys ...string) error {
	signers := []*keypair.Full{}
	for _, k := range keys {
		kp, err := keypair.Parse(k)
		if err != nil {
			return errors.Wrapf(err, "provided string %s is not a valid Stellar key", k)
		}
		kpf, ok := kp.(*keypair.Full)
		if !ok {
			return errors.New("provided string %s is not a valid Stellar secret key")
		}
		signers = append(signers, kpf)
	}

	return tx.Sign(signers...)
}

// ReadChallengeTx reads a SEP 10 challenge transaction and returns the decoded
// transaction and client account ID contained within.
//
// It also verifies that transaction is signed by the server.
//
// It does not verify that the transaction has been signed by the client or
// that any signatures other than the servers on the transaction are valid. Use
// one of the following functions to completely verify the transaction:
// - VerifyChallengeTxThreshold
// - VerifyChallengeTxSigners
func ReadChallengeTx(challengeTx, serverAccountID, network string) (tx Transaction, clientAccountID string, err error) {
	tx, err = TransactionFromXDR(challengeTx)
	if err != nil {
		return tx, clientAccountID, err
	}
	tx.Network = network

	// verify transaction source
	if tx.SourceAccount == nil {
		return tx, clientAccountID, errors.New("transaction requires a source account")
	}
	if tx.SourceAccount.GetAccountID() != serverAccountID {
		return tx, clientAccountID, errors.New("transaction source account is not equal to server's account")
	}

	// verify sequence number
	txSourceAccount, ok := tx.SourceAccount.(*SimpleAccount)
	if !ok {
		return tx, clientAccountID, errors.New("source account is not of type SimpleAccount unable to verify sequence number")
	}
	if txSourceAccount.Sequence != 0 {
		return tx, clientAccountID, errors.New("transaction sequence number must be 0")
	}

	// verify timebounds
	if tx.Timebounds.MaxTime == TimeoutInfinite {
		return tx, clientAccountID, errors.New("transaction requires non-infinite timebounds")
	}
	currentTime := time.Now().UTC().Unix()
	if currentTime < tx.Timebounds.MinTime || currentTime > tx.Timebounds.MaxTime {
		return tx, clientAccountID, errors.Errorf("transaction is not within range of the specified timebounds (currentTime=%d, MinTime=%d, MaxTime=%d)",
			currentTime, tx.Timebounds.MinTime, tx.Timebounds.MaxTime)
	}

	// verify operation
	if len(tx.Operations) != 1 {
		return tx, clientAccountID, errors.New("transaction requires a single manage_data operation")
	}
	op, ok := tx.Operations[0].(*ManageData)
	if !ok {
		return tx, clientAccountID, errors.New("operation type should be manage_data")
	}
	if op.SourceAccount == nil {
		return tx, clientAccountID, errors.New("operation should have a source account")
	}
	clientAccountID = op.SourceAccount.GetAccountID()

	// verify manage data value
	nonceB64 := string(op.Value)
	if len(nonceB64) != 64 {
		return tx, clientAccountID, errors.New("random nonce encoded as base64 should be 64 bytes long")
	}
	nonceBytes, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return tx, clientAccountID, errors.Wrap(err, "failed to decode random nonce provided in manage_data operation")
	}
	if len(nonceBytes) != 48 {
		return tx, clientAccountID, errors.New("random nonce before encoding as base64 should be 48 bytes long")
	}

	err = verifyTxSignature(tx, serverAccountID)
	if err != nil {
		return tx, clientAccountID, err
	}

	return tx, clientAccountID, nil
}

// VerifyChallengeTxThreshold verifies that for a SEP 10 challenge transaction
// all signatures on the transaction are accounted for and that the signatures
// meet a threshold on an account. A transaction is verified if it is signed by
// the server account, and all other signatures match a signer that has been
// provided as an argument, and those signatures meet a threshold on the
// account.
//
// Signers that are not prefixed as an address/account ID strkey (G...) will be
// ignored.
//
// Errors will be raised if:
//  - The transaction is invalid according to ReadChallengeTx.
//  - No client signatures are found on the transaction.
//  - One or more signatures in the transaction are not identifiable as the
//    server account or one of the signers provided in the arguments.
//  - The signatures are all valid but do not meet the threshold.
func VerifyChallengeTxThreshold(challengeTx, serverAccountID, network string, threshold Threshold, signerSummary SignerSummary) (signersFound []string, err error) {
	signers := make([]string, 0, len(signerSummary))
	for s := range signerSummary {
		signers = append(signers, s)
	}

	signersFound, err = VerifyChallengeTxSigners(challengeTx, serverAccountID, network, signers...)
	if err != nil {
		return nil, err
	}

	weight := int32(0)
	for _, s := range signersFound {
		weight += signerSummary[s]
	}

	if weight < int32(threshold) {
		return nil, errors.Errorf("signers with weight %d do not meet threshold %d", weight, threshold)
	}

	return signersFound, nil
}

// VerifyChallengeTxSigners verifies that for a SEP 10 challenge transaction
// all signatures on the transaction are accounted for. A transaction is
// verified if it is signed by the server account, and all other signatures
// match a signer that has been provided as an argument. Additional signers can
// be provided that do not have a signature, but all signatures must be matched
// to a signer for verification to succeed. If verification succeeds a list of
// signers that were found is returned, excluding the server account ID.
//
// Signers that are not prefixed as an address/account ID strkey (G...) will be
// ignored.
//
// Errors will be raised if:
//  - The transaction is invalid according to ReadChallengeTx.
//  - No client signatures are found on the transaction.
//  - One or more signatures in the transaction are not identifiable as the
//    server account or one of the signers provided in the arguments.
func VerifyChallengeTxSigners(challengeTx, serverAccountID, network string, signers ...string) ([]string, error) {
	// Read the transaction which validates its structure.
	tx, _, err := ReadChallengeTx(challengeTx, serverAccountID, network)
	if err != nil {
		return nil, err
	}

	// Ensure the server account ID is an address and not a seed.
	serverKP, err := keypair.ParseAddress(serverAccountID)
	if err != nil {
		return nil, err
	}

	// Deduplicate the client signers and ensure the server is not included
	// anywhere we check or output the list of signers.
	clientSigners := []string{}
	clientSignersSeen := map[string]struct{}{}
	for _, signer := range signers {
		// Ignore the server signer if it is in the signers list. It's
		// important when verifying signers of a challenge transaction that we
		// only verify and return client signers. If an account has the server
		// as a signer the server should not play a part in the authentication
		// of the client.
		if signer == serverKP.Address() {
			continue
		}
		// Deduplicate.
		if _, seen := clientSignersSeen[signer]; seen {
			continue
		}
		// Ignore non-G... account/address signers.
		strkeyVersionByte, strkeyErr := strkey.Version(signer)
		if strkeyErr != nil {
			continue
		}
		if strkeyVersionByte != strkey.VersionByteAccountID {
			continue
		}
		clientSigners = append(clientSigners, signer)
		clientSignersSeen[signer] = struct{}{}
	}

	// Don't continue if none of the signers provided are in the final list.
	if len(clientSigners) == 0 {
		return nil, errors.New("no verifiable signers provided, at least one G... address must be provided")
	}

	// Verify all the transaction's signers (server and client) in one
	// hit. We do this in one hit here even though the server signature was
	// checked in the ReadChallengeTx to ensure that every signature and signer
	// are consumed only once on the transaction.
	allSigners := append([]string{serverKP.Address()}, clientSigners...)
	allSignersFound, err := verifyTxSignatures(tx, allSigners...)
	if err != nil {
		return nil, err
	}

	// Confirm the server is in the list of signers found and remove it.
	serverSignerFound := false
	signersFound := make([]string, 0, len(allSignersFound)-1)
	for _, signer := range allSignersFound {
		if signer == serverKP.Address() {
			serverSignerFound = true
			continue
		}
		signersFound = append(signersFound, signer)
	}

	// Confirm we matched a signature to the server signer.
	if !serverSignerFound {
		return nil, errors.Errorf("transaction not signed by %s", serverKP.Address())
	}

	// Confirm we matched signatures to the client signers.
	if len(signersFound) == 0 {
		return nil, errors.Errorf("transaction not signed by %s", strings.Join(clientSigners, ", "))
	}

	// Confirm all signatures were consumed by a signer.
	if len(allSignersFound) != len(tx.xdrEnvelope.Signatures) {
		return signersFound, errors.Errorf("transaction has unrecognized signatures")
	}

	return signersFound, nil
}

// VerifyChallengeTx is a factory method that verifies a SEP 10 challenge transaction,
// for use in web authentication. It can be used by a server to verify that the challenge
// has been signed by the client account's master key.
// More details on SEP 10: https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0010.md
//
// Deprecated: Use VerifyChallengeTxThreshold or VerifyChallengeTxSigners.
func VerifyChallengeTx(challengeTx, serverAccountID, network string) (bool, error) {
	tx, clientAccountID, err := ReadChallengeTx(challengeTx, serverAccountID, network)
	if err != nil {
		return false, err
	}

	err = verifyTxSignature(tx, clientAccountID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// verifyTxSignature checks if a transaction has been signed by the provided Stellar account.
func verifyTxSignature(tx Transaction, signer string) error {
	signersFound, err := verifyTxSignatures(tx, signer)
	if len(signersFound) == 0 {
		return errors.Errorf("transaction not signed by %s", signer)
	}
	return err
}

// verifyTxSignature checks if a transaction has been signed by one or more of
// the signers, returning a list of signers that were found to have signed the
// transaction.
func verifyTxSignatures(tx Transaction, signers ...string) ([]string, error) {
	if tx.xdrEnvelope == nil {
		return nil, errors.New("transaction has no signatures")
	}

	txHash, err := tx.Hash()
	if err != nil {
		return nil, err
	}

	// find and verify signatures
	signatureUsed := map[int]bool{}
	signersFound := map[string]struct{}{}
	for _, signer := range signers {
		kp, err := keypair.ParseAddress(signer)
		if err != nil {
			return nil, errors.Wrap(err, "signer not address")
		}

		for i, decSig := range tx.xdrEnvelope.Signatures {
			if signatureUsed[i] {
				continue
			}
			if decSig.Hint != kp.Hint() {
				continue
			}
			err := kp.Verify(txHash[:], decSig.Signature)
			if err == nil {
				signatureUsed[i] = true
				signersFound[signer] = struct{}{}
				break
			}
		}
	}

	signersFoundList := make([]string, 0, len(signersFound))
	for _, signer := range signers {
		if _, ok := signersFound[signer]; ok {
			signersFoundList = append(signersFoundList, signer)
			delete(signersFound, signer)
		}
	}
	return signersFoundList, nil
}
