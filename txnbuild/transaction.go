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
	"math"
	"strings"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// MinBaseFee is the minimum base for the Stellar network
const MinBaseFee = 100

// Account represents the aspects of a Stellar account necessary to construct transactions. See
// https://www.stellar.org/developers/guides/concepts/accounts.html
type Account interface {
	GetAccountID() string
	IncrementSequenceNumber() (xdr.SequenceNumber, error)
	GetSequenceNumber() (xdr.SequenceNumber, error)
}

type Transaction struct {
	envelope      xdr.TransactionEnvelope
	baseFee       int64
	totalFee      int64
	isFeeBump     bool
	feeSource     string
	innerTx       *Transaction
	sourceAccount SimpleAccount
	operations    []Operation
	memo          Memo
	timebounds    Timebounds
	signatures    []xdr.DecoratedSignature
}

func (t *Transaction) BaseFee() int64 {
	return t.baseFee
}

func (t *Transaction) TotalFee() int64 {
	return t.totalFee
}

func (t *Transaction) IsFeeBump() bool {
	return t.isFeeBump
}

func (t *Transaction) FeeSource() string {
	return t.feeSource
}

func (t *Transaction) InnerTransaction() *Transaction {
	if t.isFeeBump {
		innerCopy := new(Transaction)
		*innerCopy = *t.innerTx
		return innerCopy
	}
	return nil
}

func (t *Transaction) SourceAccount() SimpleAccount {
	return t.sourceAccount
}

func (t *Transaction) Operations() []Operation {
	operations := make([]Operation, len(t.operations))
	copy(operations, t.operations)
	return operations
}

func (t *Transaction) Memo() Memo {
	return t.memo
}

func (t *Transaction) Timebounds() Timebounds {
	return t.timebounds
}

func (t *Transaction) Signatures() []xdr.DecoratedSignature {
	signatures := make([]xdr.DecoratedSignature, len(t.signatures))
	copy(signatures, t.signatures)
	return signatures
}

func (t *Transaction) Hash(networkStr string) ([32]byte, error) {
	return network.HashTransactionInEnvelope(t.envelope, networkStr)
}

func (t *Transaction) HashHex(network string) (string, error) {
	hash, err := t.Hash(network)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash[:]), nil
}

func (t *Transaction) Sign(network string, kps ...*keypair.Full) (*Transaction, error) {
	// Hash the transaction
	hash, err := t.Hash(network)
	if err != nil {
		return t, errors.Wrap(err, "failed to hash transaction")
	}

	newTx := new(Transaction)
	*newTx = *t
	newTx.signatures = make(
		[]xdr.DecoratedSignature,
		len(t.signatures),
		len(t.signatures)+len(kps),
	)
	copy(newTx.signatures, t.signatures)

	// Sign the hash
	for _, kp := range kps {
		sig, err := kp.SignDecorated(hash[:])
		if err != nil {
			return newTx, errors.Wrap(err, "failed to sign transaction")
		}
		newTx.signatures = append(newTx.signatures, sig)
	}

	return newTx, nil
}

func (t *Transaction) SignWithKeyString(network string, keys ...string) (*Transaction, error) {
	var signers []*keypair.Full
	for _, k := range keys {
		kp, err := keypair.Parse(k)
		if err != nil {
			return t, errors.Wrapf(err, "provided string %s is not a valid Stellar key", k)
		}
		kpf, ok := kp.(*keypair.Full)
		if !ok {
			return t, errors.New("provided string %s is not a valid Stellar secret key")
		}
		signers = append(signers, kpf)
	}

	return t.Sign(network, signers...)
}

// SignHashX signs a transaction with HashX signature type.
// See description here: https://www.stellar.org/developers/guides/concepts/multi-sig.html#hashx.
func (t *Transaction) SignHashX(preimage []byte) (*Transaction, error) {
	if len(preimage) > xdr.Signature(preimage).XDRMaxSize() {
		return t, errors.New("preimage cannnot be more than 64 bytes")
	}

	preimageHash := sha256.Sum256(preimage)
	var hint [4]byte
	// copy the last 4-bytes of the signer public key to be used as hint
	copy(hint[:], preimageHash[28:])

	sig := xdr.DecoratedSignature{
		Hint:      xdr.SignatureHint(hint),
		Signature: xdr.Signature(preimage),
	}

	newTx := new(Transaction)
	*newTx = *t
	newTx.signatures = make(
		[]xdr.DecoratedSignature,
		len(t.signatures),
		len(t.signatures)+1,
	)
	copy(newTx.signatures, t.signatures)
	newTx.signatures = append(newTx.signatures, sig)

	return newTx, nil
}

func (t *Transaction) TxEnvelope() (xdr.TransactionEnvelope, error) {
	var env xdr.TransactionEnvelope
	serialized, err := t.MarshalBinary()
	if err != nil {
		return env, errors.Wrap(err, "could not marshall envelope")
	}

	if err = xdr.SafeUnmarshal(serialized, &env); err != nil {
		return env, err
	}
	return env, nil
}

// MarshalBinary returns the binary XDR representation of the transaction envelope.
func (t *Transaction) MarshalBinary() ([]byte, error) {
	switch t.envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		t.envelope.V1.Signatures = t.signatures
	case xdr.EnvelopeTypeEnvelopeTypeTxV0:
		t.envelope.V0.Signatures = t.signatures
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		t.envelope.FeeBump.Signatures = t.signatures
	default:
		panic("invalid transaction type: " + t.envelope.Type.String())
	}

	var txBytes bytes.Buffer
	_, err := xdr.Marshal(&txBytes, t.envelope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal XDR")
	}

	return txBytes.Bytes(), nil
}

// Base64 returns the base 64 XDR representation of the transaction envelope.
func (t *Transaction) Base64() (string, error) {
	bs, err := t.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "failed to get XDR bytestring")
	}

	return base64.StdEncoding.EncodeToString(bs), nil
}

// TransactionFromXDR parses the supplied transaction envelope in base64 XDR and returns a Transaction object.
func TransactionFromXDR(txeB64 string) (*Transaction, error) {
	var xdrEnv xdr.TransactionEnvelope
	err := xdr.SafeUnmarshalBase64(txeB64, &xdrEnv)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal transaction envelope")
	}
	return transactionFromParsedXDR(xdrEnv)
}

func transactionFromParsedXDR(xdrEnv xdr.TransactionEnvelope) (*Transaction, error) {
	var err error
	newTx := &Transaction{}

	newTx.envelope = xdrEnv
	for _, signature := range xdrEnv.Signatures() {
		newTx.signatures = append(newTx.signatures, signature)
	}
	sourceAccount := xdrEnv.SourceAccount()
	newTx.sourceAccount = SimpleAccount{
		AccountID: sourceAccount.Address(),
		Sequence:  xdrEnv.SeqNum(),
	}
	newTx.feeSource = newTx.sourceAccount.AccountID

	numOps := len(xdrEnv.Operations())
	if xdrEnv.IsFeeBump() {
		feeBumpAccount := xdrEnv.FeeBumpAccount()
		newTx.feeSource = feeBumpAccount.Address()
		newTx.totalFee = xdrEnv.FeeBumpFee()
		newTx.baseFee = newTx.totalFee
		newTx.isFeeBump = true
		if numOps > 0 {
			newTx.baseFee = newTx.baseFee / int64(numOps+1)
		}

		newTx.innerTx, err = transactionFromParsedXDR(xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1:   xdrEnv.FeeBump.Tx.InnerTx.V1,
		})
		if err != nil {
			return nil, errors.New("could not parse inner transaction")
		}
	} else {
		newTx.totalFee = int64(xdrEnv.Fee())
		newTx.baseFee = newTx.totalFee
		if numOps > 0 {
			newTx.baseFee = newTx.baseFee / int64(numOps)
		}
	}

	if timeBounds := xdrEnv.TimeBounds(); timeBounds != nil {
		newTx.timebounds = NewTimebounds(int64(timeBounds.MinTime), int64(timeBounds.MaxTime))
	}

	newTx.memo, err = memoFromXDR(xdrEnv.Memo())
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse memo")
	}

	operations := xdrEnv.Operations()
	newTx.operations = make([]Operation, len(operations), len(operations))
	for i, op := range operations {
		newOp, err := operationFromXDR(op)
		if err != nil {
			return nil, err
		}
		newTx.operations[i] = newOp
	}

	return newTx, nil
}

// TransactionConfig is a container for configuration options
// which are used to construct new Transaction instances
type TransactionConfig struct {
	SourceAccount        Account
	IncrementSequenceNum bool
	Operations           []Operation
	BaseFee              int64
	Memo                 Memo
	Timebounds           Timebounds
}

// NewTransaction returns a new Transaction instance
func NewTransaction(config TransactionConfig) (*Transaction, error) {
	var sequence xdr.SequenceNumber
	var err error
	if config.IncrementSequenceNum {
		sequence, err = config.SourceAccount.IncrementSequenceNumber()
	} else {
		sequence, err = config.SourceAccount.GetSequenceNumber()
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not obtain account sequence")
	}
	tx := &Transaction{
		baseFee:   config.BaseFee,
		totalFee:  config.BaseFee * int64(len(config.Operations)),
		isFeeBump: false,
		feeSource: config.SourceAccount.GetAccountID(),
		innerTx:   nil,
		sourceAccount: SimpleAccount{
			AccountID: config.SourceAccount.GetAccountID(),
			Sequence:  int64(sequence),
		},
		operations: config.Operations,
		memo:       config.Memo,
		timebounds: config.Timebounds,
		signatures: nil,
	}

	accountID, err := xdr.AddressToAccountId(tx.sourceAccount.AccountID)
	if err != nil {
		return tx, errors.Wrap(err, "account id is not valid")
	}

	sourceAccountEd25519, ok := accountID.GetEd25519()
	if !ok {
		return tx, errors.New("invalid account id")
	}
	// check if totalFee fits in a uint32
	// 64 bit fees are only available in fee bump transactions
	if tx.totalFee > math.MaxUint32 {
		return tx, errors.New("fee overflows uint32")
	}
	if tx.baseFee < MinBaseFee {
		return tx, errors.New("base fee is lower than network minimum")
	}

	// Check and set the timebounds
	err = tx.timebounds.Validate()
	if err != nil {
		return tx, errors.Wrap(err, "invalid time bounds")
	}

	tx.envelope = xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTxV0,
		V0: &xdr.TransactionV0Envelope{
			Tx: xdr.TransactionV0{
				SourceAccountEd25519: sourceAccountEd25519,
				Fee:                  xdr.Uint32(tx.totalFee),
				SeqNum:               sequence,
				TimeBounds: &xdr.TimeBounds{
					MinTime: xdr.TimePoint(tx.timebounds.MinTime),
					MaxTime: xdr.TimePoint(tx.timebounds.MaxTime),
				},
			},
			Signatures: nil,
		},
	}

	// Handle the memo, if one is present
	if tx.memo != nil {
		xdrMemo, err := tx.memo.ToXDR()
		if err != nil {
			return tx, errors.Wrap(err, "couldn't build memo XDR")
		}
		tx.envelope.V0.Tx.Memo = xdrMemo
	}

	for _, op := range tx.operations {
		if verr := op.Validate(); verr != nil {
			return tx, errors.Wrap(verr, fmt.Sprintf("validation failed for %T operation", op))
		}

		xdrOperation, err2 := op.BuildXDR()
		if err2 != nil {
			return tx, errors.Wrap(err2, fmt.Sprintf("failed to build operation %T", op))
		}
		tx.envelope.V0.Tx.Operations = append(tx.envelope.V0.Tx.Operations, xdrOperation)
	}

	return tx, nil
}

// NewSignedTransaction performs all the steps to produce a final transaction suitable
// for submitting to the network.
func NewSignedTransaction(
	config TransactionConfig,
	network string,
	keypairs ...*keypair.Full,
) (string, error) {
	tx, err := NewTransaction(config)
	if err != nil {
		return "", errors.Wrap(err, "couldn't create transaction")
	}

	tx, err = tx.Sign(network, keypairs...)
	if err != nil {
		return "", errors.Wrap(err, "couldn't sign transaction")
	}

	txeBase64, err := tx.Base64()
	if err != nil {
		return "", errors.Wrap(err, "couldn't encode transaction")
	}

	return txeBase64, err
}

// FeeBumpTransactionConfig is a container for configuration options
// which are used to construct new fee bump Transaction instances
type FeeBumpTransactionConfig struct {
	Inner     *Transaction
	FeeSource string
	BaseFee   int64
}

// NewFeeBumpTransaction returns a new fee bump Transaction instance
func NewFeeBumpTransaction(config FeeBumpTransactionConfig) (*Transaction, error) {
	tx := &Transaction{
		baseFee:       config.BaseFee,
		totalFee:      config.BaseFee * int64(len(config.Inner.operations)+1),
		isFeeBump:     true,
		feeSource:     config.FeeSource,
		innerTx:       config.Inner,
		sourceAccount: config.Inner.sourceAccount,
		operations:    config.Inner.operations,
		memo:          config.Inner.memo,
		timebounds:    config.Inner.timebounds,
		signatures:    nil,
	}

	if tx.baseFee < tx.innerTx.baseFee {
		return tx, errors.New("base fee is lower than inner transaction")
	}
	if tx.baseFee < MinBaseFee {
		return tx, errors.New("base fee is lower than network minimum")
	}

	accountID, err := xdr.AddressToMuxedAccount(tx.feeSource)
	if err != nil {
		return tx, errors.Wrap(err, "fee source is not a valid")
	}

	innerEnv, err := tx.innerTx.TxEnvelope()
	if err != nil {
		return tx, errors.Wrap(err, "could not get inner transaction envelope")
	}
	if innerEnv.Type != xdr.EnvelopeTypeEnvelopeTypeTx {
		return tx, errors.Errorf("%v transactions cannot be fee bumped", innerEnv.Type.String())
	}

	tx.envelope = xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTxFeeBump,
		FeeBump: &xdr.FeeBumpTransactionEnvelope{
			Tx: xdr.FeeBumpTransaction{
				FeeSource: accountID,
				Fee:       xdr.Int64(tx.totalFee),
				InnerTx: xdr.FeeBumpTransactionInnerTx{
					Type: xdr.EnvelopeTypeEnvelopeTypeTx,
					V1:   tx.innerTx.envelope.V1,
				},
			},
		},
	}

	return tx, nil
}

// NewSignedFeeBumpTransaction performs all the steps to produce a final
// fee bump transaction suitable for submitting to the network.
func NewSignedFeeBumpTransaction(
	config FeeBumpTransactionConfig,
	network string,
	keypairs ...*keypair.Full,
) (string, error) {
	tx, err := NewFeeBumpTransaction(config)
	if err != nil {
		return "", errors.Wrap(err, "couldn't create transaction")
	}

	tx, err = tx.Sign(network, keypairs...)
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
		Sequence:  0,
	}

	// represent client account as SimpleAccount
	ca := SimpleAccount{
		AccountID: clientAccountID,
	}

	currentTime := time.Now().UTC()
	maxTime := currentTime.Add(timebound)

	// Create a SEP 10 compatible response. See
	// https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0010.md#response
	return NewSignedTransaction(
		TransactionConfig{
			SourceAccount:        &sa,
			IncrementSequenceNum: false,
			Operations: []Operation{
				&ManageData{
					SourceAccount: &ca,
					Name:          anchorName + " auth",
					Value:         []byte(randomNonceToString),
				},
			},
			BaseFee:    100,
			Memo:       nil,
			Timebounds: NewTimebounds(currentTime.Unix(), maxTime.Unix()),
		},
		network,
		serverKP.(*keypair.Full),
	)
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

func validAccountId(address string) error {
	var account xdr.AccountId
	return account.SetAddress(address)
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
func ReadChallengeTx(challengeTx, serverAccountID, network string) (tx *Transaction, clientAccountID string, err error) {
	tx, err = TransactionFromXDR(challengeTx)
	if err != nil {
		return tx, clientAccountID, errors.Wrap(err, "could not parse challenge")
	}

	// Enforce no muxed accounts (at least until we understand their impact)
	if err = validAccountId(tx.sourceAccount.AccountID); err != nil {
		err = errors.Wrap(err, "only valid Ed25519 accounts are allowed in challenge transactions")
		return tx, clientAccountID, err
	}
	for _, op := range tx.operations {
		sourceAccount := op.GetSourceAccount()
		address := sourceAccount.GetAccountID()

		if err = validAccountId(address); err != nil {
			err = errors.Wrap(err, "only valid Ed25519 accounts are allowed in challenge transactions")
			return tx, clientAccountID, err
		}
	}

	if tx.IsFeeBump() {
		return tx, clientAccountID, errors.New("challenge is a fee bump transaction")
	}

	// verify transaction source
	if tx.SourceAccount().AccountID != serverAccountID {
		return tx, clientAccountID, errors.New("transaction source account is not equal to server's account")
	}

	// verify sequence number
	if tx.SourceAccount().Sequence != 0 {
		return tx, clientAccountID, errors.New("transaction sequence number must be 0")
	}

	// verify timebounds
	if tx.Timebounds().MaxTime == TimeoutInfinite {
		return tx, clientAccountID, errors.New("transaction requires non-infinite timebounds")
	}
	currentTime := time.Now().UTC().Unix()
	if currentTime < tx.Timebounds().MinTime || currentTime > tx.Timebounds().MaxTime {
		return tx, clientAccountID, errors.Errorf("transaction is not within range of the specified timebounds (currentTime=%d, MinTime=%d, MaxTime=%d)",
			currentTime, tx.Timebounds().MinTime, tx.Timebounds().MaxTime)
	}

	// verify operation
	operations := tx.Operations()
	if len(operations) != 1 {
		return tx, clientAccountID, errors.New("transaction requires a single manage_data operation")
	}
	op, ok := operations[0].(*ManageData)
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
	allSignersFound, err := verifyTxSignatures(tx, network, allSigners...)
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
	if len(allSignersFound) != len(tx.Signatures()) {
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
func verifyTxSignature(tx *Transaction, signer string) error {
	signersFound, err := verifyTxSignatures(tx, signer)
	if len(signersFound) == 0 {
		return errors.Errorf("transaction not signed by %s", signer)
	}
	return err
}

// verifyTxSignature checks if a transaction has been signed by one or more of
// the signers, returning a list of signers that were found to have signed the
// transaction.
func verifyTxSignatures(tx *Transaction, network string, signers ...string) ([]string, error) {
	txHash, err := tx.Hash(network)
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

		for i, decSig := range tx.Signatures() {
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
