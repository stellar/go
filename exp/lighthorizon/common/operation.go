package common

import (
	"encoding/hex"

	"github.com/stellar/go/network"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type Operation struct {
	TransactionEnvelope *xdr.TransactionEnvelope
	TransactionResult   *xdr.TransactionResult
	LedgerHeader        *xdr.LedgerHeader
	OpIndex             int32
	TxIndex             int32
}

func (o *Operation) Get() *xdr.Operation {
	return &o.TransactionEnvelope.Operations()[o.OpIndex]
}

func (o *Operation) OperationResult() *xdr.OperationResultTr {
	results, _ := o.TransactionResult.OperationResults()
	tr := results[o.OpIndex].MustTr()
	return &tr
}

func (o *Operation) TransactionHash() (string, error) {
	hash, err := network.HashTransactionInEnvelope(*o.TransactionEnvelope, network.PublicNetworkPassphrase)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash[:]), nil
}

func (o *Operation) SourceAccount() xdr.AccountId {
	sourceAccount := o.TransactionEnvelope.SourceAccount().ToAccountId()
	if o.Get().SourceAccount != nil {
		sourceAccount = o.Get().SourceAccount.ToAccountId()
	}
	return sourceAccount
}

func (o *Operation) TOID() int64 {
	return toid.New(
		int32(o.LedgerHeader.LedgerSeq),
		o.TxIndex+1,
		o.OpIndex+1,
	).ToInt64()
}
