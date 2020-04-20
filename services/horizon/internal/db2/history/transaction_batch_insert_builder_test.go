package history

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
)

func TestTransactionToMap_muxed(t *testing.T) {
	innerSource := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      1,
			Ed25519: xdr.Uint256{3, 2, 1},
		},
	}
	innerAccountID := innerSource.ToAccountId()
	feeSource := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      1,
			Ed25519: xdr.Uint256{0, 1, 2},
		},
	}
	feeSourceAccountID := feeSource.ToAccountId()
	tx := io.LedgerTransaction{
		Index: 1,
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTxFeeBump,
			FeeBump: &xdr.FeeBumpTransactionEnvelope{
				Tx: xdr.FeeBumpTransaction{
					FeeSource: feeSource,
					Fee:       200,
					InnerTx: xdr.FeeBumpTransactionInnerTx{
						Type: xdr.EnvelopeTypeEnvelopeTypeTx,
						V1: &xdr.TransactionV1Envelope{
							Tx: xdr.Transaction{
								SourceAccount: innerSource,
								Operations: []xdr.Operation{
									{
										SourceAccount: &innerSource,
										Body: xdr.OperationBody{
											Type: xdr.OperationTypePayment,
											PaymentOp: &xdr.PaymentOp{
												Destination: innerSource,
												Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
												Amount:      100,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Result: xdr.TransactionResultPair{
			TransactionHash: xdr.Hash{1, 2, 3},
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					Code: xdr.TransactionResultCodeTxFeeBumpInnerSuccess,
					InnerResultPair: &xdr.InnerTransactionResultPair{
						TransactionHash: xdr.Hash{3, 2, 1},
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
	assert.NoError(t, err)

	assert.NotEqual(t, innerSource.Address(), result["account"])
	assert.Equal(t, innerAccountID.Address(), result["account"])

	assert.NotEqual(t, feeSource.Address(), result["fee_account"])
	assert.Equal(t, feeSourceAccountID.Address(), result["fee_account"])

}
