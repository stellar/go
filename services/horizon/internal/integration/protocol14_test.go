package integration

import (
	"testing"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var protocol14Config = test.IntegrationConfig{ProtocolVersion: 14}

func TestProtocol14SanityCheck(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	root, err := itest.Client().Root()
	assert.NoError(t, err)
	assert.Equal(t, int32(14), root.CoreSupportedProtocolVersion)
	assert.Equal(t, int32(14), root.CurrentProtocolVersion)

	// Submit a simple tx
	op := txnbuild.Payment{
		Destination: master.Address(),
		Amount:      "10",
		Asset:       txnbuild.NativeAsset{},
	}

	txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &op)
	assert.Equal(t, master.Address(), txResp.Account)
	assert.Equal(t, "1", txResp.AccountSequence)
}

func TestCreateClaimableBalance(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	// Submit a self-referencing claimable balance
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
	opsResults := *txResult.Result.Results
	opResult := opsResults[0].MustTr().MustCreateClaimableBalanceResult()
	assert.Equal(t,
		xdr.CreateClaimableBalanceResultCodeCreateClaimableBalanceSuccess,
		opResult.Code,
	)
	assert.NotNil(t, opResult.BalanceId)
}

func TestFilteringClaimableBalances(t *testing.T) {
	// We test on all three types of assets. This convenience function just
	// prepares the image (docker, etc.), sets up the scenario, then delegates
	// to the real test w/ parameters set.
	prepareAndRun := func(t *testing.T, assetType txnbuild.AssetType) {
		itest := test.NewIntegrationTest(t, protocol14Config)
		defer itest.Close()

		keys, _ := itest.CreateAccounts(2)
		runFilteringTest(itest, keys[0], keys[1],
			createAsset(assetType, keys[0].Address()))
	}

	t.Run("Native", func(t *testing.T) { prepareAndRun(t, txnbuild.AssetTypeNative) })
	t.Run("4-Char Asset", func(t *testing.T) { prepareAndRun(t, txnbuild.AssetTypeCreditAlphanum4) })
	t.Run("12-Char Asset", func(t *testing.T) { prepareAndRun(t, txnbuild.AssetTypeCreditAlphanum12) })
}

func runFilteringTest(i *test.IntegrationTest, source *keypair.Full, dest *keypair.Full, asset txnbuild.Asset) {
	t := i.CurrentTest()
	client := i.Client()
	request := sdk.AccountRequest{AccountID: source.Address()}
	sourceAccount, err := client.AccountDetail(request)
	assert.NoError(t, err)

	op := txnbuild.CreateClaimableBalance{
		Destinations: []txnbuild.Claimant{
			txnbuild.NewClaimant(dest.Address(), nil),
		},
		Amount: "10",
		Asset:  asset,
	}

	// Submit a simple claimable balance from A -> B.
	_, err = i.SubmitOperations(&sourceAccount, source, &op)
	assert.NoError(t, err)

	// Ensure it exists in the global list
	balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
	assert.NoError(t, err)

	claims := balances.Embedded.Records
	assert.Len(t, claims, 1)
	assert.Equal(t, source.Address(), claims[0].Sponsor)
	id := claims[0].BalanceID

	// Ensure we can look it up explicitly by ID
	balance, err := client.ClaimableBalance(id)
	assert.NoError(t, err)
	assert.Equal(t, claims[0], balance)

	//
	// Ensure it shows up with the various filters (and *doesn't* show up with
	// non-matching filters, of course).
	//

	t.Log("Filtering by sponsor")
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: source.Address()})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 1)
	assert.Equal(t, claims[0], balances.Embedded.Records[0])

	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: dest.Address()})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)

	t.Log("Filtering by claimant")
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Claimant: source.Address()})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)

	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Claimant: dest.Address()})
	assert.NoError(t, err)
	assert.Equal(t, claims[0], balances.Embedded.Records[0])

	t.Log("Filtering by assets")
	t.Log("  by exact")
	xdrAsset, err := asset.ToXDR()
	assert.NoError(t, err)

	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Asset: xdrAsset.StringCanonical()})

	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 1)

	// a native asset shouldn't show up when filtering by non-native
	t.Log("  by mismatching")
	randomAsset := txnbuild.CreditAsset{Code: "RAND", Issuer: source.Address()}
	xdrAsset, err = randomAsset.ToXDR()
	assert.NoError(t, err)

	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Asset: xdrAsset.StringCanonical()})

	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)

	// similarly, a non-native asset shouldn't show up when filtering by a
	// *different* non-native NOR when filtering by native
	aType, err := asset.GetType()
	assert.NoError(t, err)

	t.Log("  by native")
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Asset: "native"})
	assert.NoError(t, err)

	expectedLength := 0
	if aType == txnbuild.AssetTypeNative {
		expectedLength++
	}

	assert.Len(t, balances.Embedded.Records, expectedLength)
}

/* Utility functions below */

func createAsset(assetType txnbuild.AssetType, issuer string) txnbuild.Asset {
	switch assetType {
	case txnbuild.AssetTypeNative:
		return txnbuild.NativeAsset{}
	case txnbuild.AssetTypeCreditAlphanum4:
		return txnbuild.CreditAsset{Code: "HEYO", Issuer: issuer}
	case txnbuild.AssetTypeCreditAlphanum12:
		return txnbuild.CreditAsset{Code: "HEYYYAAAAAAA", Issuer: issuer}
	default:
		panic(-1)
	}
}
