package filters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func TestEmpty(t *testing.T) {
	tt := assert.New(t)
	ctx := context.Background()

	filterParams := &AssetFilterParms{}
	filter := NewAssetFilter(filterParams)

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
