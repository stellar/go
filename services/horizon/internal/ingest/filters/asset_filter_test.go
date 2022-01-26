package filters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func TestFilterHasMatch(t *testing.T) {
	// TODO, make this test real
	tt := assert.New(t)
	ctx := context.Background()

	filterParams := &AssetFilterParms{
		Activated:          true,
		CanonicalAssetList: []string{"USDC:GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"},
	}
	filter := NewAssetFilterFromParams(filterParams)

	var xdrAssetCode [12]byte
	copy(xdrAssetCode[:], "USDC")
	var xdrIssuer xdr.AccountId
	require.NoError(t, xdrIssuer.SetAddress("GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"))

	ledgerTx := ingest.LedgerTransaction{
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
						{Body: xdr.OperationBody{
							Type: xdr.OperationTypePayment,
							PaymentOp: &xdr.PaymentOp{
								Destination: xdr.MustMuxedAddress("GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"),
								Asset: xdr.Asset{
									Type: xdr.AssetTypeAssetTypeCreditAlphanum12,
									AlphaNum12: &xdr.AlphaNum12{
										AssetCode: xdrAssetCode,
										Issuer:    xdrIssuer,
									},
								},
								Amount: 100,
							},
						}},
					},
				},
			},
		},
	}

	result, err := filter.FilterTransaction(ctx, ledgerTx)

	tt.NoError(err)
	tt.Equal(result, true)
}

func TestFilterHasNoMatch(t *testing.T) {
	// TODO, make this test real
	tt := assert.New(t)
	ctx := context.Background()

	filterParams := &AssetFilterParms{
		Activated:          true,
		CanonicalAssetList: []string{"USDX:GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"},
	}
	filter := NewAssetFilterFromParams(filterParams)

	var xdrAssetCode [12]byte
	copy(xdrAssetCode[:], "USDC")
	var xdrIssuer xdr.AccountId
	require.NoError(t, xdrIssuer.SetAddress("GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"))

	ledgerTx := ingest.LedgerTransaction{
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
						{Body: xdr.OperationBody{
							Type: xdr.OperationTypePayment,
							PaymentOp: &xdr.PaymentOp{
								Destination: xdr.MustMuxedAddress("GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"),
								Asset: xdr.Asset{
									Type: xdr.AssetTypeAssetTypeCreditAlphanum12,
									AlphaNum12: &xdr.AlphaNum12{
										AssetCode: xdrAssetCode,
										Issuer:    xdrIssuer,
									},
								},
								Amount: 100,
							},
						}},
					},
				},
			},
		},
	}

	result, err := filter.FilterTransaction(ctx, ledgerTx)

	tt.NoError(err)
	tt.Equal(result, false)
}

func TestParamsFromFile(t *testing.T) {
	tt := assert.New(t)

	filter, err := NewAssetFilterFromParamsFile("../testdata/test_asset_filter_params.json")

	tt.NoError(err)
	tt.Equal(filter.CurrentFilterParameters().ResolveLiquidityPoolAsAsset, true)
	tt.Equal(filter.CurrentFilterParameters().CanonicalAssetList, []string{"BITC:ABC123456", "DOGT:DEF123456"})
}
