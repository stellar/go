package common

import (
	"encoding/hex"

	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

type Operation struct {
	TransactionEnvelope *xdr.TransactionEnvelope
	TransactionResult   *xdr.TransactionResult
	LedgerHeader        *xdr.LedgerHeader
	Index               int32
}

func (o *Operation) Get() *xdr.Operation {
	return &o.TransactionEnvelope.Operations()[o.Index]
}

func (o *Operation) OperationResult() *xdr.OperationResultTr {
	results, _ := o.TransactionResult.OperationResults()
	tr := results[o.Index].MustTr()
	return &tr
}

func (o *Operation) TransactionHash() (string, error) {
	hash, err := network.HashTransactionInEnvelope(*o.TransactionEnvelope, network.PublicNetworkPassphrase)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash[:]), nil
}

func (o *Operation) SourceAccount() string {
	sourceAccount := o.TransactionEnvelope.SourceAccount().ToAccountId().Address()
	if o.Get().SourceAccount != nil {
		sourceAccount = o.Get().SourceAccount.Address()
	}
	return sourceAccount
}
