package integration

import (
	"testing"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	proto "github.com/stellar/go/protocols/horizon"
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
		Destinations: []string{master.Address()},
		Amount:       "10",
		Asset:        txnbuild.NativeAsset{},
	}

	txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &op)

	var txResult xdr.TransactionResult
	err := xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
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

func TestFilteringNonNativeClaimableBalances(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client := itest.Client()

	keypairs, _ := itest.CreateAccounts(2)
	sender, recipient := keypairs[0], keypairs[1]

	asset := txnbuild.CreditAsset{Code: "HELLO", Issuer: sender.Address()}
	_, err := itest.EstablishTrustline(recipient, asset)
	assert.NoError(t, err)
	t.Log("Created asset trustline.")

	runFilteringTest(t, client, sender, recipient, asset)
}

func TestFilteringClaimableBalances(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	// Create a couple of accounts to test the interactions.
	keypairs, _ := itest.CreateAccounts(2)
	sender, recipient := keypairs[0], keypairs[1]

	runFilteringTest(t, itest.Client(), sender, recipient, txnbuild.NativeAsset{})
}

func TestClaimingClaimableBalances(t *testing.T) {
	runClaimingCBsTest(t, txnbuild.AssetTypeNative)
}

func TestClaimingNonNativeClaimableBalances(t *testing.T) {
	runClaimingCBsTest(t, txnbuild.AssetTypeCreditAlphanum12)
	runClaimingCBsTest(t, txnbuild.AssetTypeCreditAlphanum4)
}

func runClaimingCBsTest(t *testing.T, assetType txnbuild.AssetType) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client := itest.Client()

	// Create a couple of accounts to test the interactions.
	keypairs, accounts := itest.CreateAccounts(2)
	sender, recipient := keypairs[0], keypairs[1]
	rAccount := accounts[1]

	var asset txnbuild.Asset
	if assetType != txnbuild.AssetTypeNative {
		asset = txnbuild.CreditAsset{Code: "HEYO", Issuer: sender.Address()}
		_, err := itest.EstablishTrustline(recipient, asset)
		assert.NoError(t, err)
		t.Log("Created asset trustline.")
	} else {
		asset = txnbuild.NativeAsset{}
	}

	// This is an easy shortcut to setting up the scenario.
	runFilteringTest(t, client, sender, recipient, asset)

	// Now let's retrieve what the above just created so we can claim it.
	balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: sender.Address()})
	assert.NoError(t, err)

	claims := balances.Embedded.Records
	assert.Len(t, claims, 1)
	assert.Equal(t, claims[0].Sponsor, sender.Address())

	op := txnbuild.ClaimClaimableBalance{BalanceID: claims[0].BalanceID}
	_, err = itest.SubmitOperations(rAccount, recipient, &op)
	assert.NoError(t, err)
	t.Log("Claimed balance.")

	// Ensure the balance is gone now.
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: sender.Address()})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)
}

func runFilteringTest(t *testing.T, client *sdk.Client,
	source *keypair.Full, dest *keypair.Full, asset txnbuild.Asset,
) {
	request := sdk.AccountRequest{AccountID: source.Address()}
	sourceAccount, err := client.AccountDetail(request)
	assert.NoError(t, err)

	op := txnbuild.CreateClaimableBalance{
		Destinations: []string{dest.Address()},
		Amount:       "10",
		Asset:        asset,
	}

	// Submit a simple claimable balance from A -> B.
	tx, err := makeAndSign(source, transact(&sourceAccount, &op))
	assert.NoError(t, err)
	submitOrLog(t, client, tx)

	// Ensure it exists in the global list
	balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
	assert.NoError(t, err)

	claims := balances.Embedded.Records
	assert.Len(t, claims, 1)
	assert.Equal(t, claims[0].Sponsor, source.Address())
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

	if aType != txnbuild.AssetTypeNative {
		assert.Len(t, balances.Embedded.Records, 0)
	} else {
		assert.Len(t, balances.Embedded.Records, 1)
	}
}

/* Utility functions below */

func makeAndSign(signer *keypair.Full, params txnbuild.TransactionParams) (string, error) {
	tx, err := txnbuild.NewTransaction(params)
	if err != nil {
		return "", err
	}
	tx, err = tx.Sign(test.IntegrationNetworkPassphrase, signer)
	if err != nil {
		return "", err
	}
	txb64, err := tx.Base64()
	if err != nil {
		return "", err
	}
	return txb64, nil
}

func transact(account txnbuild.Account, ops ...txnbuild.Operation) txnbuild.TransactionParams {
	return txnbuild.TransactionParams{
		SourceAccount:        account,
		Operations:           ops,
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		IncrementSequenceNum: true,
	}
}

func submitOrLog(t *testing.T, client *sdk.Client, xdr string) (response proto.Transaction, err error) {
	response, err = client.SubmitTransactionXDR(xdr)
	assert.NoError(t, err)
	if err != nil {
		prob := sdk.GetError(err)
		t.Logf("Problem: %s\n", prob.Problem.Extras["result_codes"])
		return
	}

	return
}
