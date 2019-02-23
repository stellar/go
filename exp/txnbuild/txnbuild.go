package txnbuild

import (
	"bytes"
	"encoding/base64"
	"strconv"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// StellarNetwork ...
var StellarNetwork string

// UseTestNetwork ...
func UseTestNetwork() {
	StellarNetwork = network.TestNetworkPassphrase
}

// UsePublicNetwork ...
func UsePublicNetwork() {
	StellarNetwork = network.PublicNetworkPassphrase
}

// OperationTypeCreateAccount      OperationType = 0
// OperationTypePayment            OperationType = 1
// OperationTypePathPayment        OperationType = 2
// OperationTypeManageOffer        OperationType = 3
// OperationTypeCreatePassiveOffer OperationType = 4
// OperationTypeSetOptions         OperationType = 5
// OperationTypeChangeTrust        OperationType = 6
// OperationTypeAllowTrust         OperationType = 7
// OperationTypeAccountMerge       OperationType = 8
// OperationTypeInflation          OperationType = 9
// OperationTypeManageData         OperationType = 10
// OperationTypeBumpSequence       OperationType = 11
func getXDROpType(op Operation) (xdr.OperationType, error) {
	switch t := op.(type) {
	case CreateAccount:
		return xdr.OperationTypeCreateAccount, nil
	case Inflation:
		return xdr.OperationTypeInflation, nil
	default:
		return 0, errors.Errorf("initialiseOperation: Bad operation type '%T'", t)
	}
}

// type Account struct {
// 	AccountID string
// 	Sequence  string
// }

// Operation ...
type Operation interface{}

// Inflation ...
type Inflation struct {
	xdrOp xdr.OperationType
}

// CreateAccount ...
type CreateAccount struct {
	Destination string
	Amount      string
	Asset       string
}

// Transaction ...
type Transaction struct {
	SourceAccount horizon.Account
	Operations    []Operation
	TX            *xdr.Transaction
	BaseFee       uint64
	Envelope      *xdr.TransactionEnvelope
}

// Hash ...
func (tx *Transaction) Hash() ([32]byte, error) {
	return network.HashTransaction(tx.TX, StellarNetwork)
}

// Bytes ...
func (tx *Transaction) Bytes() ([]byte, error) {
	var txBytes bytes.Buffer
	_, err := xdr.Marshal(&txBytes, tx.Envelope)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal XDR")
	}

	return txBytes.Bytes(), nil
}

// Base64 ...
func (tx *Transaction) Base64() (string, error) {
	bs, err := tx.Bytes()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get XDR bytestring")
	}

	return base64.StdEncoding.EncodeToString(bs), nil
}

// SetDefaults ...
func (tx *Transaction) SetDefaultFee() {
	// TODO: Check if default base fee used elsewhere - otherwise just use int
	var DefaultBaseFee uint64 = 100
	if tx.BaseFee == 0 {
		tx.BaseFee = DefaultBaseFee
	}
	if tx.TX.Fee == 0 {
		tx.TX.Fee = xdr.Uint32(int(tx.BaseFee) * len(tx.TX.Operations))
	}
}

// Build ...
func (tx *Transaction) Build() error {
	// Initialise TX (XDR) struct if needed
	if tx.TX == nil {
		tx.TX = &xdr.Transaction{}
	}

	// Set account ID in TX
	tx.TX.SourceAccount.SetAddress(tx.SourceAccount.ID)

	// Set sequence number in TX
	seqNum, err := SeqNumFromAccount(tx.SourceAccount)
	if err != nil {
		return err
	}
	tx.TX.SeqNum = seqNum + 1

	for _, op := range tx.Operations {
		// Create operation body
		opType, err := getXDROpType(op)
		body, err := xdr.NewOperationBody(opType, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to create XDR")
		}
		// Append relevant operation to TX.operations
		xdrOperation := xdr.Operation{Body: body}
		tx.TX.Operations = append(tx.TX.Operations, xdrOperation)
	}

	// Set a default fee, if it hasn't been set yet
	tx.SetDefaultFee()

	return nil
}

// Sign ...
func (tx *Transaction) Sign(seed string) error {
	// Initialise transaction envelope
	if tx.Envelope == nil {
		tx.Envelope = &xdr.TransactionEnvelope{}
		tx.Envelope.Tx = *tx.TX
	}

	// Hash the transaction
	hash, err := tx.Hash()
	if err != nil {
		return errors.Wrap(err, "Failed to hash transaction")
	}

	// Sign the hash
	// TODO: Allow multiple signers
	kp, err := keypair.Parse(seed)
	if err != nil {
		return errors.Wrap(err, "Failed to parse seed")
	}

	sig, err := kp.SignDecorated(hash[:])
	if err != nil {
		return errors.Wrap(err, "Failed to sign transaction")
	}

	// Append the signature to the envelope
	tx.Envelope.Signatures = append(tx.Envelope.Signatures, sig)

	return nil
}

// SeqNumFromAccount ...
func SeqNumFromAccount(account horizon.Account) (xdr.SequenceNumber, error) {
	seqNum, err := strconv.ParseUint(account.Sequence, 10, 64)

	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse account sequence number")
	}

	return xdr.SequenceNumber(seqNum), nil
}
