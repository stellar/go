package common

import (
	"encoding/hex"
	"errors"

	"github.com/stellar/go/network"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type Transaction struct {
	TransactionEnvelope *xdr.TransactionEnvelope
	TransactionResult   *xdr.TransactionResult
	LedgerHeader        *xdr.LedgerHeader
	TxIndex             int32

	NetworkPassphrase string
}

func (o *Transaction) TransactionHash() (string, error) {
	if o.NetworkPassphrase == "" {
		return "", errors.New("network passphrase unspecified")
	}

	hash, err := network.HashTransactionInEnvelope(*o.TransactionEnvelope, o.NetworkPassphrase)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash[:]), nil
}

func (o *Transaction) SourceAccount() xdr.MuxedAccount {
	return o.TransactionEnvelope.SourceAccount()
}

func (o *Transaction) TOID() int64 {
	return toid.New(
		int32(o.LedgerHeader.LedgerSeq),
		o.TxIndex+1,
		1,
	).ToInt64()
}
