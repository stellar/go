package history

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
)

func TestTransactionToMap_muxed(t *testing.T) {
	muxed := xdr.MustMuxedAccountAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG")
	tx := io.LedgerTransaction{
		Index: 1,
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: muxed,
					Operations: []xdr.Operation{
						{
							SourceAccount: &muxed,
							Body: xdr.OperationBody{
								Type: xdr.OperationTypePayment,
								PaymentOp: &xdr.PaymentOp{
									Destination: muxed,
									Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
									Amount:      100,
								},
							},
						},
					},
				},
			},
		},
		Result: xdr.TransactionResultPair{
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					InnerResultPair: &xdr.InnerTransactionResultPair{
						Result: xdr.InnerTransactionResult{
							Result: xdr.InnerTransactionResultResult{
								Results: &[]xdr.OperationResult{},
							},
						},
					},
					Results: &[]xdr.OperationResult{},
				},
			},
		},
		Meta: xdr.TransactionMeta{
			V:          1,
			Operations: &[]xdr.OperationMeta{},
			V1: &xdr.TransactionMetaV1{
				TxChanges:  []xdr.LedgerEntryChange{},
				Operations: []xdr.OperationMeta{},
			},
		},
	}
	result, err := transactionToMap(tx, 20)
	if assert.NoError(t, err) {
		assert.Equal(t, "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ", result["account"])
	}

}
