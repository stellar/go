package integration

import (
	"fmt"
	"testing"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var protocol14Config = test.IntegrationConfig{ProtocolVersion: 14}

func TestProtocol14Sample(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	root, err := itest.Client().Root()
	assert.NoError(t, err)
	assert.Equal(t, int32(14), root.CoreSupportedProtocolVersion)
	assert.Equal(t, int32(14), root.CurrentProtocolVersion)

	// Submit a simple tx
	master := itest.Master()
	op := txnbuild.Payment{
		Destination: master.Address(),
		Amount:      "10",
		Asset:       txnbuild.NativeAsset{},
	}
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: master.Address(),
				Sequence:  1,
			},
			Operations: []txnbuild.Operation{&op},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	tx, err = tx.Sign(test.IntegrationNetworkPassphrase, itest.Master().(*keypair.Full))
	assert.NoError(t, err)

	txb64, err := tx.Base64()
	assert.NoError(t, err)

	txResp, err := itest.Client().SubmitTransactionXDR(txb64)
	assert.NoError(t, err)
	assert.Equal(t, itest.Master().Address(), txResp.Account)
	assert.Equal(t, "1", txResp.AccountSequence)
}

func TestCreateClaimableBalance(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	// Submit a simple tx
	master := itest.Master()
	op := txnbuild.CreateClaimableBalance{
		Destinations: []string{master.Address()},
		Amount:       "10",
		Asset:        txnbuild.NativeAsset{},
	}
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: master.Address(),
				Sequence:  1,
			},
			Operations: []txnbuild.Operation{&op},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	tx, err = tx.Sign(test.IntegrationNetworkPassphrase, itest.Master().(*keypair.Full))
	assert.NoError(t, err)

	txb64, err := tx.Base64()
	assert.NoError(t, err)

	txResp, err := itest.Client().SubmitTransactionXDR(txb64)
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

func TestNonNativeClaimableBalance(t *testing.T) {
	protocol14Config.SkipContainerCreation = true
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client := itest.Client()
	master := itest.Master().(*keypair.Full)

	accounts, tx, err := createAccounts(master, 2, int64(1))
	assert.NoError(t, err)

	if err = submitOrLog(t, client, tx); err == nil {
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
	if err = submitOrLog(t, client, tx); err == nil {
		t.Log("Created asset trustline.")
	}

	RunClaimableBalanceTest(t, client, accounts[0], accounts[1], asset)
}

func TestClaimableBalances(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client := itest.Client()

	master := itest.Master().(*keypair.Full)

	// Create a couple of accounts to test the interactions.
	accounts, tx, err := createAccounts(master, 2, int64(1))
	assert.NoError(t, err)

	if err = submitOrLog(t, client, tx); err == nil {
		for _, account := range accounts {
			t.Logf("Funded %s (%s).\n", account.Seed(), account.Address())
		}
	}

	RunClaimableBalanceTest(t, client, accounts[0], accounts[1], txnbuild.NativeAsset{})
}

func RunClaimableBalanceTest(t *testing.T, client *sdk.Client,
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
	tx, err := makeAndSign(
		source,
		txnbuild.TransactionParams{
			SourceAccount:        &sourceAccount,
			Operations:           []txnbuild.Operation{&op},
			BaseFee:              txnbuild.MinBaseFee,
			Timebounds:           txnbuild.NewInfiniteTimeout(),
			IncrementSequenceNum: true,
		},
	)
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

	// filter by sponsor
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: source.Address()})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 1)
	assert.Equal(t, claims[0], balances.Embedded.Records[0])

	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: dest.Address()})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)

	// filter by claimant
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Claimant: source.Address()})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)

	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Claimant: dest.Address()})
	assert.NoError(t, err)
	assert.Equal(t, claims[0], balances.Embedded.Records[0])

	t.Log("Done.")
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

func submitOrLog(t *testing.T, client *sdk.Client, xdr string) (err error) {
	_, err = client.SubmitTransactionXDR(xdr)
	assert.NoError(t, err)
	if err != nil {
		prob := sdk.GetError(err)
		t.Logf("Problem (if any): %s\n", prob.Problem.Extras["result_codes"])
		return err
	}

	return nil
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
	tx, err := makeAndSign(
		master,
		txnbuild.TransactionParams{
			SourceAccount: &masterAccount,
			BaseFee:       txnbuild.MinBaseFee,
			Timebounds:    txnbuild.NewInfiniteTimeout(),
			Operations:    ops,
		},
	)

	return pairs, tx, err
}

func createValueFromThinAir(client *sdk.Client, truster *keypair.Full, asset txnbuild.Asset) (tx string, err error) {
	// Load the source account
	request := sdk.AccountRequest{AccountID: truster.Address()}
	sourceAccount, err := client.AccountDetail(request)
	if err != nil {
		return
	}

	txp := txnbuild.TransactionParams{
		SourceAccount:        &sourceAccount,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		Operations: []txnbuild.Operation{
			&txnbuild.ChangeTrust{
				Line:  asset,
				Limit: "2000",
			},
		},
		Memo: txnbuild.MemoText(fmt.Sprintf("i trust u %s", asset.GetCode())),
	}

	// The usual song & dance w/ signing, submitting, etc.
	tx, err = makeAndSign(truster, txp)
	return
}
