package io

import (
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestStatsLedgerTransactionProcessor(t *testing.T) {
	processor := &StatsLedgerTransactionProcessor{}

	// Successful
	assert.NoError(t, processor.ProcessTransaction(LedgerTransaction{
		Result: xdr.TransactionResultPair{
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					Code: xdr.TransactionResultCodeTxSuccess,
				},
			},
		},
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					Operations: []xdr.Operation{
						{Body: xdr.OperationBody{Type: xdr.OperationTypeCreateAccount}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypePayment}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypePathPaymentStrictReceive}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeManageSellOffer}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeCreatePassiveSellOffer}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeSetOptions}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeChangeTrust}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeAllowTrust}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeAccountMerge}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeInflation}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeManageData}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeBumpSequence}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeManageBuyOffer}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypePathPaymentStrictSend}},
					},
				},
			},
		},
	}))

	// Failed
	assert.NoError(t, processor.ProcessTransaction(LedgerTransaction{
		Result: xdr.TransactionResultPair{
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					Code: xdr.TransactionResultCodeTxFailed,
				},
			},
		},
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					Operations: []xdr.Operation{
						{Body: xdr.OperationBody{Type: xdr.OperationTypeCreateAccount}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypePayment}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypePathPaymentStrictReceive}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeManageSellOffer}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeCreatePassiveSellOffer}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeSetOptions}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeChangeTrust}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeAllowTrust}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeAccountMerge}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeInflation}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeManageData}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeBumpSequence}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeManageBuyOffer}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypePathPaymentStrictSend}},
					},
				},
			},
		},
	}))

	results := processor.GetResults()

	assert.Equal(t, int64(2), results.Transactions)
	assert.Equal(t, int64(1), results.TransactionsSuccessful)
	assert.Equal(t, int64(1), results.TransactionsFailed)

	assert.Equal(t, int64(14*2), results.Operations)
	assert.Equal(t, int64(14), results.OperationsInSuccessful)
	assert.Equal(t, int64(14), results.OperationsInFailed)

	assert.Equal(t, int64(2), results.OperationsCreateAccount)
	assert.Equal(t, int64(2), results.OperationsPayment)
	assert.Equal(t, int64(2), results.OperationsPathPaymentStrictReceive)
	assert.Equal(t, int64(2), results.OperationsManageSellOffer)
	assert.Equal(t, int64(2), results.OperationsCreatePassiveSellOffer)
	assert.Equal(t, int64(2), results.OperationsSetOptions)
	assert.Equal(t, int64(2), results.OperationsChangeTrust)
	assert.Equal(t, int64(2), results.OperationsAllowTrust)
	assert.Equal(t, int64(2), results.OperationsAccountMerge)
	assert.Equal(t, int64(2), results.OperationsInflation)
	assert.Equal(t, int64(2), results.OperationsManageData)
	assert.Equal(t, int64(2), results.OperationsBumpSequence)
	assert.Equal(t, int64(2), results.OperationsManageBuyOffer)
	assert.Equal(t, int64(2), results.OperationsPathPaymentStrictSend)
}
