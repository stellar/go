package integration

import (
	"testing"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	// "fmt"
	// "strconv"
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

func CreateFundedAccount(master *keypair.Full, seq int64) (*keypair.Full, string, error) {
	// client := sdk.DefaultTestNetClient
	pair, _ := keypair.Random()
	amount := "1000"

	masterAccount := txnbuild.SimpleAccount{
		AccountID: master.Address(),
		Sequence:  seq,
	}

	// Build transaction:
	tx, err := MakeAndSign(
		master,
		txnbuild.TransactionParams{
			SourceAccount: &masterAccount,
			BaseFee:       txnbuild.MinBaseFee,
			Timebounds:    txnbuild.NewInfiniteTimeout(),
			Operations: []txnbuild.Operation{
				&txnbuild.CreateAccount{
					Destination:   pair.Address(),
					Amount:        amount,
					SourceAccount: &masterAccount,
				},
			},
		},
	)

	return pair, tx, err
}

func TestClaimableBalances(t *testing.T) {
	// client := sdk.DefaultTestNetClient

	// The use case is straightforward:
	//
	// > It should be easy to send a payment to an account that is not
	// > necessarily prepared to receive the payment.
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	seq := int64(1)
	master := itest.Master().(*keypair.Full)

	// Create a couple of accounts to test the interactions.
	a, tx, err := CreateFundedAccount(master, seq)
	assert.NoError(t, err)
	seq++

	_, err = itest.Client().SubmitTransactionXDR(tx)
	assert.NoError(t, err)
	if err != nil {
		prob := sdk.GetError(err)
		t.Logf("Problem (if any): %s\n", prob.Problem.Extras["result_codes"])
	} else {
		t.Logf("Funded %s.\n", a.Seed())
	}

	b, tx, err := CreateFundedAccount(master, seq)
	assert.NoError(t, err)
	seq++

	_, err = itest.Client().SubmitTransactionXDR(tx)
	assert.NoError(t, err)
	if err != nil {
		prob := sdk.GetError(err)
		t.Logf("Problem (if any): %s\n", prob.Problem.Extras["result_codes"])
	} else {
		t.Logf("Funded %s.\n", b.Seed())
	}

	masterAccount := txnbuild.SimpleAccount{
		AccountID: master.Address(),
		Sequence:  seq,
	}

	// Submit a simple tx
	op := txnbuild.CreateClaimableBalance{
		Destinations: []string{masterAccount.AccountID},
		Amount:       "10",
		Asset:        txnbuild.NativeAsset{},
	}

	tx, err = MakeAndSign(
		master,
		txnbuild.TransactionParams{
			SourceAccount: &masterAccount,
			Operations:    []txnbuild.Operation{&op},
			BaseFee:       txnbuild.MinBaseFee,
			Timebounds:    txnbuild.NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	_, err = itest.Client().SubmitTransactionXDR(tx)
	assert.NoError(t, err)
	if err != nil {
		prob := sdk.GetError(err)
		t.Logf("Problem: %s\n", prob.Problem.Extras["result_codes"])
	}
}
