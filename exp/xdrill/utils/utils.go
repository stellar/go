package utils

import (
	"encoding/hex"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

// HashToHexString is utility function that converts and xdr.Hash type to a hex string
func HashToHexString(inputHash xdr.Hash) string {
	sliceHash := inputHash[:]
	hexString := hex.EncodeToString(sliceHash)
	return hexString
}

type ID struct {
	LedgerSequence   int32
	TransactionOrder int32
	OperationOrder   int32
}

const (
	// LedgerMask is the bitmask to mask out ledger sequences in a
	// TotalOrderID
	LedgerMask = (1 << 32) - 1
	// TransactionMask is the bitmask to mask out transaction indexes
	TransactionMask = (1 << 20) - 1
	// OperationMask is the bitmask to mask out operation indexes
	OperationMask = (1 << 12) - 1

	// LedgerShift is the number of bits to shift an int64 to target the
	// ledger component
	LedgerShift = 32
	// TransactionShift is the number of bits to shift an int64 to
	// target the transaction component
	TransactionShift = 12
	// OperationShift is the number of bits to shift an int64 to target
	// the operation component
	OperationShift = 0
)

// New creates a new total order ID
func NewID(ledger int32, tx int32, op int32) *ID {
	return &ID{
		LedgerSequence:   ledger,
		TransactionOrder: tx,
		OperationOrder:   op,
	}
}

// ToInt64 converts this struct back into an int64
func (id ID) ToInt64() (result int64) {

	if id.LedgerSequence < 0 {
		panic("invalid ledger sequence")
	}

	if id.TransactionOrder > TransactionMask {
		panic("transaction order overflow")
	}

	if id.OperationOrder > OperationMask {
		panic("operation order overflow")
	}

	result = result | ((int64(id.LedgerSequence) & LedgerMask) << LedgerShift)
	result = result | ((int64(id.TransactionOrder) & TransactionMask) << TransactionShift)
	result = result | ((int64(id.OperationOrder) & OperationMask) << OperationShift)
	return
}

// TODO: This should be moved into the go monorepo xdr functions
// Or nodeID should just be an xdr.AccountId but the error message would be incorrect
func GetAddress(nodeID xdr.NodeId) (string, bool) {
	switch nodeID.Type {
	case xdr.PublicKeyTypePublicKeyTypeEd25519:
		ed, ok := nodeID.GetEd25519()
		if !ok {
			return "", false
		}
		raw := make([]byte, 32)
		copy(raw, ed[:])
		encodedAddress, err := strkey.Encode(strkey.VersionByteAccountID, raw)
		if err != nil {
			return "", false
		}
		return encodedAddress, true
	default:
		return "", false
	}
}

func CreateSampleTx(sequence int64, operationCount int) xdr.TransactionEnvelope {
	kp, err := keypair.Random()
	PanicOnError(err)

	operations := []txnbuild.Operation{}
	operationType := &txnbuild.BumpSequence{
		BumpTo: 0,
	}
	for i := 0; i < operationCount; i++ {
		operations = append(operations, operationType)
	}

	sourceAccount := txnbuild.NewSimpleAccount(kp.Address(), int64(0))
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    operations,
			BaseFee:       txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		},
	)
	PanicOnError(err)

	env := tx.ToXDR()
	return env
}

// PanicOnError is a function that panics if the provided error is not nil
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

// GetAccountAddressFromMuxedAccount takes in a muxed account and returns the address of the account
func GetAccountAddressFromMuxedAccount(account xdr.MuxedAccount) (string, error) {
	providedID := account.ToAccountId()
	pointerToID := &providedID
	return pointerToID.GetAddress()
}

func GetAccountBalanceFromLedgerEntryChanges(changes xdr.LedgerEntryChanges, sourceAccountAddress string) (int64, int64) {
	var accountBalanceStart int64
	var accountBalanceEnd int64

	for _, change := range changes {
		switch change.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			accountEntry, ok := change.Updated.Data.GetAccount()
			if !ok {
				continue
			}

			if accountEntry.AccountId.Address() == sourceAccountAddress {
				accountBalanceEnd = int64(accountEntry.Balance)
			}
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			accountEntry, ok := change.State.Data.GetAccount()
			if !ok {
				continue
			}

			if accountEntry.AccountId.Address() == sourceAccountAddress {
				accountBalanceStart = int64(accountEntry.Balance)
			}
		}
	}

	return accountBalanceStart, accountBalanceEnd
}

func GetTxSigners(xdrSignatures []xdr.DecoratedSignature) ([]string, error) {
	signers := make([]string, len(xdrSignatures))

	for i, sig := range xdrSignatures {
		signerAccount, err := strkey.Encode(strkey.VersionByteAccountID, sig.Signature)
		if err != nil {
			return nil, err
		}
		signers[i] = signerAccount
	}

	return signers, nil
}
