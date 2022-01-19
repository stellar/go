package filters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func TestWIP(t *testing.T) {
	// TODO, make this test real
	tt := assert.New(t)
	ctx := context.Background()

	filterParams := &AssetFilterParms{
		CanonicalAssetList: []string{"USDC:XYZ1234567890"},
	}
	filter := NewAssetFilterFromParams(filterParams)

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
						{Body: xdr.OperationBody{Type: xdr.OperationTypeCreateClaimableBalance}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeClaimClaimableBalance}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeBeginSponsoringFutureReserves}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeEndSponsoringFutureReserves}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeRevokeSponsorship}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeClawback}},
						{Body: xdr.OperationBody{Type: xdr.OperationTypeClawbackClaimableBalance}},
					},
				},
			},
		},
	}

	result, err := filter.FilterTransaction(ctx, ledgerTx)

	tt.NoError(err)
	tt.Equal(result, true)
}

func TestParamsFromFile(t *testing.T) {
	tt := assert.New(t)

	filter, err := NewAssetFilterFromParamsFile("../testdata/test_filter_params.json")

	tt.NoError(err)
	tt.Equal(filter.CurrentFilterParameters().ResolveLiquidityPoolAsAsset, true)
	tt.Equal(filter.CurrentFilterParameters().CanonicalAssetList, []string{"BITC:ABC123456", "DOGT:DEF123456"})
}
