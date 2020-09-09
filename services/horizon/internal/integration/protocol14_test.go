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

var protocol14Config = test.IntegrationConfig{
	ProtocolVersion: 14,
	// SkipContainerCreation: true,
}

func TestProtocol14SanityCheck(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	client, master := itest.Client(), itest.Master().(*keypair.Full)

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

	tx, err := makeAndSign(master,
		transact(&txnbuild.SimpleAccount{
			AccountID: master.Address(),
			Sequence:  0,
		}, &op),
	)
	assert.NoError(t, err)

	txResp, err := submitOrLog(t, client, tx)
	assert.NoError(t, err)
	assert.Equal(t, itest.Master().Address(), txResp.Account)
	assert.Equal(t, "1", txResp.AccountSequence)
}

func TestCreateClaimableBalance(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	client, master := itest.Client(), itest.Master().(*keypair.Full)

	// Submit a simple tx
	op := txnbuild.CreateClaimableBalance{
		Destinations: []string{master.Address()},
		Amount:       "10",
		Asset:        txnbuild.NativeAsset{},
	}

	tx, err := makeAndSign(
		master,
		transact(&txnbuild.SimpleAccount{
			AccountID: master.Address(),
			Sequence:  0,
		}, &op),
	)
	assert.NoError(t, err)

	txResp, err := submitOrLog(t, client, tx)
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

func TestFilteringNonNativeClaimableBalances(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client, master := itest.Client(), itest.Master().(*keypair.Full)

	accounts, tx, err := createAccounts(master, 2)
	assert.NoError(t, err)

	if _, err = submitOrLog(t, client, tx); err == nil {
		for _, account := range accounts {
			t.Logf("Funded %s (%s).\n", account.Seed(), account.Address())
		}
	}

	/*
	 * Flow:
	 *	- A creates an asset via a trustline from B
	 *	- A issues a claimable balance to B
	 *	- B validates filtering, etc.
	 *	- B claims the asset A
	 *	- A validates that the asset is gone
	 */
	a, b := accounts[0], accounts[1]

	asset := txnbuild.CreditAsset{Code: "HELLO", Issuer: a.Address()}
	tx, err = createValueFromThinAir(client, b, asset)
	if _, err = submitOrLog(t, client, tx); err == nil {
		t.Log("Created asset trustline.")
	}

	runFilteringTest(t, client, accounts[0], accounts[1], asset)
}

func TestFilteringClaimableBalances(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client, master := itest.Client(), itest.Master().(*keypair.Full)

	// Create a couple of accounts to test the interactions.
	accounts, tx, err := createAccounts(master, 2)
	assert.NoError(t, err)

	if _, err = submitOrLog(t, client, tx); err == nil {
		for _, account := range accounts {
			t.Logf("Funded %s (%s).\n", account.Seed(), account.Address())
		}
	}

	runFilteringTest(t, client, accounts[0], accounts[1], txnbuild.NativeAsset{})
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

func createAccounts(master *keypair.Full, count int, seq int64) (
	[]*keypair.Full, string, error,
) {
	pairs := make([]*keypair.Full, count)
	ops := make([]txnbuild.Operation, count)
	amount := "1000"

	masterAccount := txnbuild.SimpleAccount{
		AccountID: master.Address(),
		Sequence:  seq,
	}

	for i := 0; i < count; i++ {
		pair, _ := keypair.Random()
		pairs[i] = pair

		ops[i] = &txnbuild.CreateAccount{
			SourceAccount: &masterAccount,
			Destination:   pair.Address(),
			Amount:        amount,
		}
	}

	// Build transaction:
	tx, err := makeAndSign(master, transact(&masterAccount, ops...))

	return pairs, tx, err
}

func createValueFromThinAir(client *sdk.Client, truster *keypair.Full, asset txnbuild.Asset) (tx string, err error) {
	// Load the source account
	request := sdk.AccountRequest{AccountID: truster.Address()}
	sourceAccount, err := client.AccountDetail(request)
	if err != nil {
		return
	}

	txp := transact(&sourceAccount, &txnbuild.ChangeTrust{
		Line:  asset,
		Limit: "2000",
	})

	// The usual song & dance w/ signing, submitting, etc.
	tx, err = makeAndSign(truster, txp)
	return
}
