package integration

import (
	"testing"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	hEffects "github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestClaimableBalanceCreationOperationsAndEffects(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	t.Run("Successful", func(t *testing.T) {
		op := txnbuild.CreateClaimableBalance{
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(master.Address(), nil),
			},
			Amount: "10",
			Asset:  txnbuild.NativeAsset{},
		}

		txResp, err := itest.SubmitOperations(itest.MasterAccount(), master, &op)
		tt.NoError(err)

		var txResult xdr.TransactionResult
		err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
		tt.NoError(err)
		tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

		opResults, _ := txResult.OperationResults()
		expectedBalanceID, err := xdr.MarshalHex(opResults[0].MustTr().CreateClaimableBalanceResult.BalanceId)
		tt.NoError(err)

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

		claimableBalanceCreatedEffect := effects[0].(hEffects.ClaimableBalanceCreated)
		tt.Equal("claimable_balance_created", claimableBalanceCreatedEffect.Type)
		tt.Equal("10.0000000", claimableBalanceCreatedEffect.Amount)
		tt.Equal("native", claimableBalanceCreatedEffect.Asset)
		tt.Equal(expectedBalanceID, claimableBalanceCreatedEffect.BalanceID)
		tt.Equal(master.Address(), claimableBalanceCreatedEffect.Account)

		claimableBalanceClaimantCreatedEffect := effects[1].(hEffects.ClaimableBalanceClaimantCreated)
		tt.Equal("claimable_balance_claimant_created", claimableBalanceClaimantCreatedEffect.Type)
		tt.Equal(master.Address(), claimableBalanceClaimantCreatedEffect.Account)
		tt.Equal(expectedBalanceID, claimableBalanceClaimantCreatedEffect.BalanceID)
		tt.Equal("10.0000000", claimableBalanceClaimantCreatedEffect.Amount)
		tt.Equal("native", claimableBalanceClaimantCreatedEffect.Asset)
		tt.Equal(
			xdr.ClaimPredicateTypeClaimPredicateUnconditional,
			claimableBalanceClaimantCreatedEffect.Predicate.Type,
		)

		accountDebitedEffect := effects[2].(hEffects.AccountDebited)
		tt.Equal("10.0000000", accountDebitedEffect.Amount)
		tt.Equal("native", accountDebitedEffect.Asset.Type)
		tt.Equal(master.Address(), accountDebitedEffect.Account)

		claimableBalanceSponsorshipCreated := effects[3].(hEffects.ClaimableBalanceSponsorshipCreated)
		tt.Equal("claimable_balance_sponsorship_created", claimableBalanceSponsorshipCreated.Type)
		tt.Equal(master.Address(), claimableBalanceSponsorshipCreated.Sponsor)
		tt.Equal(master.Address(), claimableBalanceSponsorshipCreated.Account)
		tt.Equal(expectedBalanceID, claimableBalanceSponsorshipCreated.BalanceID)
	})

	t.Run("Invalid", func(t *testing.T) {
		keys, accounts := itest.CreateAccounts(2, "50")
		op := txnbuild.CreateClaimableBalance{
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(master.Address(), nil),
				txnbuild.NewClaimant(keys[1].Address(), nil),
			},
			Amount: "100",
			Asset:  txnbuild.NativeAsset{},
		}

		// this operation will fail because the claimable balance is trying to
		// reserve 100 XLM but the account only has 50.
		_, err := itest.SubmitOperations(accounts[0], keys[0], &op)
		tt.Error(err)

		response, err := itest.Client().Operations(sdk.OperationRequest{
			Order:         "desc",
			Limit:         1,
			IncludeFailed: true,
		})
		ops := response.Embedded.Records
		tt.NoError(err)
		tt.Len(ops, 1)
		cb := ops[0].(operations.CreateClaimableBalance)
		tt.False(cb.TransactionSuccessful)
		tt.Equal("native", cb.Asset)
		tt.Equal("100.0000000", cb.Amount)
		tt.Equal(keys[0].Address(), cb.SourceAccount)
		tt.Len(cb.Claimants, 2)

		eResponse, err := itest.Client().Effects(sdk.EffectRequest{ForOperation: cb.ID})
		effects := eResponse.Embedded.Records
		tt.Len(effects, 0)
	})
}

func TestCreateSponsoredClaimableBalance(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	keys, accounts := itest.CreateAccounts(1, "50")
	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: master.Address(),
			},
			SponsoredID: keys[0].Address(),
		},
		&txnbuild.CreateClaimableBalance{
			SourceAccount: accounts[0],
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(master.Address(), nil),
			},
			Amount: "20",
			Asset:  txnbuild.NativeAsset{},
		},
		&txnbuild.EndSponsoringFutureReserves{},
	}

	txResp, err := itest.SubmitMultiSigOperations(accounts[0], []*keypair.Full{keys[0], master}, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	balances, err := itest.Client().ClaimableBalances(sdk.ClaimableBalanceRequest{})
	tt.NoError(err)

	claims := balances.Embedded.Records
	tt.Len(claims, 1)
	balance := claims[0]
	tt.Equal(master.Address(), balance.Sponsor)
}
