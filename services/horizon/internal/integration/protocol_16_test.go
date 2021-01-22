package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
)

var protocol16Config = integration.Config{ProtocolVersion: 16}

func TestProtocol16Basics(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, protocol16Config)
	master := itest.Master()

	t.Run("Sanity", func(t *testing.T) {
		root, err := itest.Client().Root()
		tt.NoError(err)
		tt.LessOrEqual(int32(16), root.CoreSupportedProtocolVersion)
		tt.Equal(int32(16), root.CurrentProtocolVersion)

		// Submit a simple tx
		op := txnbuild.Payment{
			Destination: master.Address(),
			Amount:      "10",
			Asset:       txnbuild.NativeAsset{},
		}

		txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &op)
		tt.Equal(master.Address(), txResp.Account)
		tt.Equal("1", txResp.AccountSequence)
	})
}

func TestHappyClawback(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, protocol16Config)
	master := itest.Master()

	// Give the master account the revocable flag (needed to set the clawback flag)
	setRevocableFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
		},
	}
	_, err := itest.SubmitOperations(itest.MasterAccount(), master, &setRevocableFlag)
	tt.NoError(err)

	// Give the master account the clawback flag
	setClawBackFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthClawbackEnabled,
		},
	}
	_, err = itest.SubmitOperations(itest.MasterAccount(), master, &setClawBackFlag)
	tt.NoError(err)

	// Make sure the clawback flag was set

	accountDetails, err := itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: master.Address(),
	})
	tt.True(accountDetails.Flags.AuthClawbackEnabled)

	// Create another account from which to claw an asset back

	keyPairs, accounts := itest.CreateAccounts(1, "100")
	accountKeyPair := keyPairs[0]
	account := accounts[0]

	// Make a payment to the account with asset which allows clawback

	// Time machine to Spain before Euros were a thing
	pesetasAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: master.Address()}
	itest.MustEstablishTrustline(accountKeyPair, account, pesetasAsset)
	pesetasPayment := txnbuild.Payment{
		Destination: accountKeyPair.Address(),
		Amount:      "10",
		Asset:       pesetasAsset,
	}
	itest.MustSubmitOperations(itest.MasterAccount(), master, &pesetasPayment)

	accountDetails, err = itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: accountKeyPair.Address(),
	})
	tt.NoError(err)
	if tt.Len(accountDetails.Balances, 2) {
		pts := accountDetails.Balances[0]
		tt.Equal("PTS", pts.Code)
		if tt.NotNil(pts.IsClawbackEnabled) {
			tt.True(*pts.IsClawbackEnabled)
		}
		tt.Equal("10.0000000", pts.Balance)
	}

	// Finally, clawback the asset
	pesetasClawback := txnbuild.Clawback{
		From:   accountKeyPair.Address(),
		Amount: "10",
		Asset:  pesetasAsset,
	}
	submissionResp, err := itest.SubmitOperations(itest.MasterAccount(), master, &pesetasClawback)
	tt.NoError(err)

	// Check that the balance was clawed back (the account's balance should be at 0)
	accountDetails, err = itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: accountKeyPair.Address(),
	})
	tt.NoError(err)
	if tt.Len(accountDetails.Balances, 2) {
		pts := accountDetails.Balances[0]
		tt.Equal("PTS", pts.Code)
		if tt.NotNil(pts.IsClawbackEnabled) {
			tt.True(*pts.IsClawbackEnabled)
		}
		tt.Equal("0.0000000", pts.Balance)
	}

	// Get the operation details
	opDetailsResponse, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: submissionResp.Hash,
	})
	tt.NoError(err)
	if tt.Len(opDetailsResponse.Embedded.Records, 1) {
		clawbackOp := opDetailsResponse.Embedded.Records[0].(operations.Clawback)
		tt.Equal("PTS", clawbackOp.Code)
		tt.Equal(accountKeyPair.Address(), clawbackOp.From)
		tt.Equal("10.0000000", clawbackOp.Amount)
	}

	// Get the operation details
	effectsResponse, err := itest.Client().Effects(horizonclient.EffectRequest{
		ForTransaction: submissionResp.Hash,
	})
	tt.NoError(err)

	if tt.Len(effectsResponse.Embedded.Records, 2) {
		accountCredited := effectsResponse.Embedded.Records[0].(effects.AccountCredited)
		tt.Equal(accountCredited.Account, master.Address())
		tt.Equal(accountCredited.Amount, "10.0000000")
		tt.Equal(accountCredited.Issuer, master.Address())
		tt.Equal(accountCredited.Code, "PTS")
		accountDebited := effectsResponse.Embedded.Records[1].(effects.AccountDebited)
		tt.Equal(accountDebited.Account, accountKeyPair.Address())
		tt.Equal(accountDebited.Amount, "10.0000000")
		tt.Equal(accountDebited.Issuer, master.Address())
		tt.Equal(accountDebited.Code, "PTS")
	}

}
