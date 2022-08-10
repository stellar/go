package common

import (
	"encoding/hex"
	"errors"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type Transaction struct {
	*archive.LedgerTransaction
	LedgerHeader *xdr.LedgerHeader
	TxIndex      int32

	NetworkPassphrase string
}

// type Transaction struct {
// 	TransactionEnvelope *xdr.TransactionEnvelope
// 	TransactionResult   *xdr.TransactionResult
// }

func (tx *Transaction) TransactionHash() (string, error) {
	if tx.NetworkPassphrase == "" {
		return "", errors.New("network passphrase unspecified")
	}

	hash, err := network.HashTransactionInEnvelope(tx.Envelope, tx.NetworkPassphrase)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash[:]), nil
}

func (o *Transaction) SourceAccount() xdr.MuxedAccount {
	return o.Envelope.SourceAccount()
}

func (tx *Transaction) TOID() int64 {
	return toid.New(
		int32(tx.LedgerHeader.LedgerSeq),
		// TOID indexing is 1-based, so the 1st tx comes at position 1,
		tx.TxIndex+1,
		// but the TOID of a transaction comes BEFORE any operation
		0,
	).ToInt64()
}

func (tx *Transaction) HasPreconditions() bool {
	switch pc := tx.Envelope.Preconditions(); pc.Type {
	case xdr.PreconditionTypePrecondNone:
		return false
	case xdr.PreconditionTypePrecondTime:
		return pc.TimeBounds != nil
	case xdr.PreconditionTypePrecondV2:
		// TODO: 2x check these
		return (pc.V2.TimeBounds != nil ||
			pc.V2.LedgerBounds != nil ||
			pc.V2.MinSeqNum != nil ||
			pc.V2.MinSeqAge > 0 ||
			pc.V2.MinSeqLedgerGap > 0 ||
			len(pc.V2.ExtraSigners) > 0)
	}

	return false
}
