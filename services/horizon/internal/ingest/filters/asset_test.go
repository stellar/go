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

func TestAssetFilterAllowsOnMatch(t *testing.T) {
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
		Name:         FilterAssetFilterName,
	}
	filter := NewAssetFilter()
	err := filter.RefreshAssetFilter(filterConfig)
	tt.NoError(err)

	result, err := filter.FilterTransaction(ctx, getAssetTestTx(t, "GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"))

	tt.NoError(err)
	tt.Equal(result, true)
}

func TestAssetFilterAllowsWhenEmptyWhitelist(t *testing.T) {
	tt := assert.New(t)
	ctx := context.Background()

	filterConfig := &history.FilterConfig{
		Rules: `{
			        "canonical_asset_whitelist": []	 
				}`,
		Enabled:      true,
		LastModified: 1,
		Name:         FilterAssetFilterName,
	}
	filter := NewAssetFilter()
	err := filter.RefreshAssetFilter(filterConfig)
	tt.NoError(err)

	result, err := filter.FilterTransaction(ctx, getAssetTestTx(t, "GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"))

	tt.NoError(err)
	tt.Equal(result, true)
}

func TestAssetFilterAllowsWhenDisabled(t *testing.T) {
	tt := assert.New(t)
	ctx := context.Background()

	filterConfig := &history.FilterConfig{
		Rules: `{
			        "canonical_asset_whitelist": [
			            "USDX:GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"
					]
				}`,
		Enabled:      false,
		LastModified: 1,
		Name:         FilterAssetFilterName,
	}
	filter := NewAssetFilter()
	err := filter.RefreshAssetFilter(filterConfig)
	tt.NoError(err)

	result, err := filter.FilterTransaction(ctx, getAssetTestTx(t, "GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"))

	tt.NoError(err)
	// there was no match on filter rules, but since filter was disabled also, it should allow all
	tt.Equal(result, true)
}

func TestAssetFilterDoesNotAllowWhenNoMatch(t *testing.T) {
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
		Name:         FilterAssetFilterName,
	}

	filter := NewAssetFilter()
	err := filter.RefreshAssetFilter(filterConfig)
	tt.NoError(err)

	result, err := filter.FilterTransaction(ctx, getAssetTestTx(t, "GD6WNNTW664WH7FXC5RUMUTF7P5QSURC2IT36VOQEEGFZ4UWUEQGECAL"))

	tt.NoError(err)
	tt.Equal(result, false)
}

func getAssetTestTx(t *testing.T, issuer string) ingest.LedgerTransaction {
	var xdrAssetCode [12]byte
	var xdrIssuer xdr.AccountId
	copy(xdrAssetCode[:], "USDC")
	require.NoError(t, xdrIssuer.SetAddress(issuer))

	return ingest.LedgerTransaction{
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
}
