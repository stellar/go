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

func MakeAndSign(signer *keypair.Full, params txnbuild.TransactionParams) (string, error) {
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

func CreateAccounts(master *keypair.Full, count int, seq int64) ([]*keypair.Full, string, error) {
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
	tx, err := MakeAndSign(
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

func TestClaimableBalances(t *testing.T) {
	// The use case is straightforward:
	//
	// > It should be easy to send a payment to an account that is not
	// > necessarily prepared to receive the payment.
	protocol14Config.SkipContainerCreation = true
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client := itest.Client()

	seq := int64(1)
	master := itest.Master().(*keypair.Full)

	// Create a couple of accounts to test the interactions.
	accounts, tx, err := CreateAccounts(master, 2, seq)
	assert.NoError(t, err)
	seq++

	_, err = client.SubmitTransactionXDR(tx)
	assert.NoError(t, err)
	if err != nil {
		prob := sdk.GetError(err)
		t.Logf("Problem (if any): %s\n", prob.Problem.Extras["result_codes"])
	} else {
		for _, account := range accounts {
			t.Logf("Funded %s (%s).\n", account.Seed(), account.Address())
		}
	}

	a, b := accounts[0], accounts[1]

	request := sdk.AccountRequest{AccountID: a.Address()}
	aAccount, err := client.AccountDetail(request)
	assert.NoError(t, err)

	// Submit a simple claimable balance from A -> B.
	tx, err = MakeAndSign(
		a,
		txnbuild.TransactionParams{
			SourceAccount: &aAccount,
			Operations: []txnbuild.Operation{
				&txnbuild.CreateClaimableBalance{
					Destinations: []string{b.Address()},
					Amount:       "10",
					Asset:        txnbuild.NativeAsset{},
				},
			},
			BaseFee:              txnbuild.MinBaseFee,
			Timebounds:           txnbuild.NewInfiniteTimeout(),
			IncrementSequenceNum: true,
		},
	)
	assert.NoError(t, err)

	_, err = client.SubmitTransactionXDR(tx)
	assert.NoError(t, err)
	if err != nil {
		prob := sdk.GetError(err)
		t.Logf("Problem: %s\n", prob.Problem.Extras["result_codes"])
	}
}
