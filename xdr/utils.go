package xdr

import (
	"encoding/hex"
	"math/big"

	"github.com/stellar/go/strkey"
)

// HashToHexString is utility function that converts and xdr.Hash type to a hex string
func HashToHexString(inputHash Hash) string {
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
func GetAddress(nodeID NodeId) (string, bool) {
	switch nodeID.Type {
	case PublicKeyTypePublicKeyTypeEd25519:
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

func ConvertStroopValueToReal(input Int64) float64 {
	output, _ := big.NewRat(int64(input), int64(10000000)).Float64()
	return output
}
