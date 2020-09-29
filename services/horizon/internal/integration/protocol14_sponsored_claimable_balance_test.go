package integration

import (
	"strconv"
	"testing"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestCreateSponsoredClaimableBalance(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)

	master := itest.Master()
	client := itest.Client()

	//
	// Confirms the operations, effects, and functionality of combining the
	// CAP-23 and CAP-33 features together as part of Protocol 14.
	//

	keys, accounts := itest.CreateAccounts(1, "50")
	sponsoree := keys[0]

	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: itest.MasterAccount(),
			SponsoredID:   sponsoree.Address(),
		},
		&txnbuild.CreateClaimableBalance{
			SourceAccount: accounts[0],
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(master.Address(), nil),
				txnbuild.NewClaimant(sponsoree.Address(), nil),
			},
			Amount: "25",
			Asset:  txnbuild.NativeAsset{},
		},
		&txnbuild.EndSponsoringFutureReserves{},
	}

	txResp, err := itest.SubmitMultiSigOperations(accounts[0],
		[]*keypair.Full{sponsoree, master}, ops...)
	itest.LogFailedTx(txResp, err)

	// Establish a baseline for the master account
	masterBalance := getAccountXLM(itest, master)

	// Check the global /claimable_balances list for success.
	balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
	tt.NoError(err)

	claims := balances.Embedded.Records
	tt.Len(claims, 1)
	balance := claims[0]
	tt.Equal(master.Address(), balance.Sponsor)

	// Claim the CB and validate balances:
	//  - sponsoree should go down for fulfilling the CB
	// 	- master should go up for claiming the CB
	txResp, err = itest.SubmitOperations(itest.MasterAccount(), master,
		&txnbuild.ClaimClaimableBalance{BalanceID: claims[0].BalanceID})
	itest.LogFailedTx(txResp, err)

	tt.Lessf(getAccountXLM(itest, sponsoree), float64(25), "sponsoree balance didn't decrease")
	tt.Greaterf(getAccountXLM(itest, master), masterBalance, "master balance didn't increase")

	// Check that operations populate.
	expectedOperations := map[string]bool{
		operations.TypeNames[xdr.OperationTypeBeginSponsoringFutureReserves]: false,
		operations.TypeNames[xdr.OperationTypeCreateClaimableBalance]:        false,
		operations.TypeNames[xdr.OperationTypeEndSponsoringFutureReserves]:   false,
	}

	opsPage, err := client.Operations(sdk.OperationRequest{})
	for _, op := range opsPage.Embedded.Records {
		opType := op.GetType()
		if _, ok := expectedOperations[opType]; ok {
			expectedOperations[opType] = true
			t.Logf("  operation %s found", opType)
		}
	}

	for expectedType, exists := range expectedOperations {
		tt.Truef(exists, "operation %s not found", expectedType)
	}

	// Check that effects populate.
	expectedEffects := map[string][]uint{
		effects.EffectTypeNames[effects.EffectClaimableBalanceSponsorshipCreated]: []uint{0, 1},
		effects.EffectTypeNames[effects.EffectClaimableBalanceCreated]:            []uint{0, 1},
		effects.EffectTypeNames[effects.EffectClaimableBalanceClaimantCreated]:    []uint{0, 2},
		effects.EffectTypeNames[effects.EffectClaimableBalanceSponsorshipRemoved]: []uint{0, 1},
		effects.EffectTypeNames[effects.EffectClaimableBalanceClaimed]:            []uint{0, 1},
	}

	effectsPage, err := client.Effects(sdk.EffectRequest{Order: "desc", Limit: 100})
	for _, effect := range effectsPage.Embedded.Records {
		effectType := effect.GetType()
		if _, ok := expectedEffects[effectType]; ok {
			expectedEffects[effectType][0] += 1
			t.Logf("  effect %s found", effectType)
		}
	}

	for expectedType, counts := range expectedEffects {
		actual, needed := counts[0], counts[1]
		tt.Equalf(needed, actual, "effect %s not found enough", expectedType)
	}
}

func getAccountXLM(i *test.IntegrationTest, account *keypair.Full) float64 {
	details := i.MustGetAccount(account)
	balance, err := strconv.ParseFloat(details.Balances[0].Balance, 64)
	if err != nil {
		panic(err)
	}
	return balance
}
