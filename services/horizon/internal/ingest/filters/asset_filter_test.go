package filters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func TestFilterHasMatch(t *testing.T) {
	tt := assert.New(t)
	ctx := context.Background()

	filterConfig := &history.FilterConfig{
		Rules: `{
			        "canonical_asset_whitelist": [
			            "USDC:GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"
					]	 
				}`,
		Enabled:      true,
		LastModified: 1,
		Name:         history.FilterAssetFilterName,
	}
	filter, err := GetAssetFilter(filterConfig)
	tt.NoError(err)

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
	tt := assert.New(t)
	ctx := context.Background()

	filterConfig := &history.FilterConfig{
		Rules: `{
			        "canonical_asset_whitelist": [ 
		                "USDX:GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"
		            ]	 
				 }`,

		Enabled:      true,
		LastModified: 1,
		Name:         history.FilterAssetFilterName,
	}

	filter, err := GetAssetFilter(filterConfig)
	tt.NoError(err)

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
