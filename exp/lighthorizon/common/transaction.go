package common

import (
	"encoding/hex"

	"github.com/stellar/go/network"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type Transaction struct {
	TransactionEnvelope *xdr.TransactionEnvelope
	TransactionResult   *xdr.TransactionResult
	LedgerHeader        *xdr.LedgerHeader
	TxIndex             int32
}

func (o *Transaction) TransactionHash() (string, error) {
	hash, err := network.HashTransactionInEnvelope(*o.TransactionEnvelope, network.PublicNetworkPassphrase)
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
