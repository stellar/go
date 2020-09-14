package integration

import (
	"testing"

	sdk "github.com/stellar/go/clients/horizonclient"
	hEffects "github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestCreateClaimableBalanceSuccessfulOperationsEffects(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	op := txnbuild.CreateClaimableBalance{
		Destinations: []txnbuild.Claimant{
			txnbuild.NewClaimant(master.Address(), nil),
		},
		Amount: "10",
		Asset:  txnbuild.NativeAsset{},
	}

	txResp, err := itest.SubmitOperations(itest.MasterAccount(), master, &op)
	assert.NoError(t, err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	assert.NoError(t, err)
	assert.Equal(t, xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	response, err := itest.Client().Operations(sdk.OperationRequest{})
	ops := response.Embedded.Records
	tt.NoError(err)
	tt.Len(ops, 1)
	cb := ops[0].(operations.CreateClaimableBalance)
	tt.Equal("native", cb.Asset)
	tt.Equal("10.0000000", cb.Amount)
	tt.Equal(itest.MasterAccount().GetAccountID(), cb.SourceAccount)
	tt.Len(cb.Claimants, 1)

	claimant := cb.Claimants[0]
	tt.Equal(itest.MasterAccount().GetAccountID(), claimant.Destination)
	tt.Equal(xdr.ClaimPredicateTypeClaimPredicateUnconditional, claimant.Predicate.Type)

	eResponse, err := itest.Client().Effects(sdk.EffectRequest{ForOperation: cb.ID})
	effects := eResponse.Embedded.Records
	tt.Len(effects, 4)

	tt.Equal("claimable_balance_created", effects[0].GetType())
	tt.Equal("claimable_balance_claimant_created", effects[1].GetType())

	accountDebitedEffect := effects[2].(hEffects.AccountDebited)
	tt.Equal("10.0000000", accountDebitedEffect.Amount)
	tt.Equal("native", accountDebitedEffect.Asset.Type)
	tt.Equal(itest.MasterAccount().GetAccountID(), accountDebitedEffect.Account)

	tt.Equal("10.0000000", accountDebitedEffect.Amount)
	tt.Equal("native", accountDebitedEffect.Asset.Type)
	tt.Equal(itest.MasterAccount().GetAccountID(), accountDebitedEffect.Account)

	tt.Equal("claimable_balance_sponsorship_created", effects[3].GetType())
}
