package integration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/codes"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

func NewProtocol16Test(t *testing.T) *integration.Test {
	// TODO, this should be removed once a core version with CAP 35 is released
	if os.Getenv("HORIZON_INTEGRATION_ENABLE_CAP_35") != "true" {
		t.Skip("skipping CAP35 test, set HORIZON_INTEGRATION_ENABLE_CAP_35=true if you want to run it")
	}
	config := integration.Config{
		ProtocolVersion: 16,
		CoreDockerImage: "2opremio/stellar-core:cap35",
	}
	return integration.NewTest(t, config)
}

func TestProtocol16Basics(t *testing.T) {
	tt := assert.New(t)
	itest := NewProtocol16Test(t)
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
	itest := NewProtocol16Test(t)
	master := itest.Master()

	// Give the master account the revocable flag (needed to set the clawback flag)
	setRevocableFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
		},
	}

	itest.MustSubmitOperations(itest.MasterAccount(), master, &setRevocableFlag)

	// Give the master account the clawback flag
	setClawBackFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthClawbackEnabled,
		},
	}
	itest.MustSubmitOperations(itest.MasterAccount(), master, &setClawBackFlag)

	// Make sure the clawback flag was set

	accountDetails := itest.MustGetAccount(master)
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

	accountDetails = itest.MustGetAccount(accountKeyPair)
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
	submissionResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &pesetasClawback)

	// Check that the balance was clawed back (the account's balance should be at 0)
	accountDetails = itest.MustGetAccount(accountKeyPair)
	if tt.Len(accountDetails.Balances, 2) {
		pts := accountDetails.Balances[0]
		tt.Equal("PTS", pts.Code)
		if tt.NotNil(pts.IsClawbackEnabled) {
			tt.True(*pts.IsClawbackEnabled)
		}
		tt.Equal("0.0000000", pts.Balance)
	}

	// Check the operation details
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

	// Check the operation effects
	effectsResponse, err := itest.Client().Effects(horizonclient.EffectRequest{
		ForTransaction: submissionResp.Hash,
	})
	tt.NoError(err)

	if tt.Len(effectsResponse.Embedded.Records, 2) {
		accountCredited := effectsResponse.Embedded.Records[0].(effects.AccountCredited)
		tt.Equal(master.Address(), accountCredited.Account)
		tt.Equal("10.0000000", accountCredited.Amount)
		tt.Equal(master.Address(), accountCredited.Issuer)
		tt.Equal("PTS", accountCredited.Code)
		accountDebited := effectsResponse.Embedded.Records[1].(effects.AccountDebited)
		tt.Equal(accountKeyPair.Address(), accountDebited.Account)
		tt.Equal("10.0000000", accountDebited.Amount)
		tt.Equal(master.Address(), accountDebited.Issuer)
		tt.Equal("PTS", accountDebited.Code)
	}

}

func TestHappyClawbackClaimableBalance(t *testing.T) {
	tt := assert.New(t)
	itest := NewProtocol16Test(t)
	master := itest.Master()

	// Give the master account the revocable flag (needed to set the clawback flag)
	setRevocableFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
		},
	}

	itest.MustSubmitOperations(itest.MasterAccount(), master, &setRevocableFlag)

	// Give the master account the clawback flag
	setClawBackFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthClawbackEnabled,
		},
	}
	itest.MustSubmitOperations(itest.MasterAccount(), master, &setClawBackFlag)

	// Make sure the clawback flag was set
	accountDetails := itest.MustGetAccount(master)
	tt.True(accountDetails.Flags.AuthClawbackEnabled)

	// Create another account as a claimable balance claimant
	keyPairs, accounts := itest.CreateAccounts(1, "100")
	accountKeyPair := keyPairs[0]
	account := accounts[0]

	// Time machine to Spain before Euros were a thing
	pesetasAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: master.Address()}
	itest.MustEstablishTrustline(accountKeyPair, account, pesetasAsset)

	// Make a claimable balance from the master account (and asset issuer) to the account with an asset which allows clawback
	pesetasCreateCB := txnbuild.CreateClaimableBalance{
		Amount: "10",
		Asset:  pesetasAsset,
		Destinations: []txnbuild.Claimant{
			txnbuild.NewClaimant(accountKeyPair.Address(), nil),
		},
	}
	itest.MustSubmitOperations(itest.MasterAccount(), master, &pesetasCreateCB)

	// Check that the claimable balance was created, clawback is enabled and obtain the id to claw it back later on
	listCBResp, err := itest.Client().ClaimableBalances(horizonclient.ClaimableBalanceRequest{
		Claimant: accountKeyPair.Address(),
	})
	tt.NoError(err)
	cbID := ""
	if tt.Len(listCBResp.Embedded.Records, 1) {
		cb := listCBResp.Embedded.Records[0]
		tt.True(cb.Flags.ClawbackEnabled)
		cbID = cb.BalanceID
		tt.Equal(master.Address(), cb.Sponsor)
	}

	// Clawback the claimable balance
	pesetasClawbackCB := txnbuild.ClawbackClaimableBalance{
		BalanceID: cbID,
	}
	clawbackCBResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &pesetasClawbackCB)

	// Make sure the claimable balance is clawed back (gone)
	_, err = itest.Client().ClaimableBalance(cbID)
	// Not found
	tt.Error(err)

	// Check the operation details
	opDetailsResponse, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: clawbackCBResp.Hash,
	})
	tt.NoError(err)
	if tt.Len(opDetailsResponse.Embedded.Records, 1) {
		clawbackOp := opDetailsResponse.Embedded.Records[0].(operations.ClawbackClaimableBalance)
		tt.Equal(cbID, *clawbackOp.ClaimableBalanceID)
	}

	// Check the operation effects
	effectsResponse, err := itest.Client().Effects(horizonclient.EffectRequest{
		ForTransaction: clawbackCBResp.Hash,
	})
	tt.NoError(err)

	if tt.Len(effectsResponse.Embedded.Records, 3) {
		claimableBalanceClawedBack := effectsResponse.Embedded.Records[0].(effects.ClaimableBalanceClawedBack)
		tt.Equal(master.Address(), claimableBalanceClawedBack.Account)
		tt.Equal(cbID, claimableBalanceClawedBack.BalanceID)
		accountCredited := effectsResponse.Embedded.Records[1].(effects.AccountCredited)
		tt.Equal(master.Address(), accountCredited.Account)
		tt.Equal("10.0000000", accountCredited.Amount)
		tt.Equal(accountCredited.Issuer, master.Address())
		tt.Equal(accountCredited.Code, "PTS")
		cbSponsorshipRemoved := effectsResponse.Embedded.Records[2].(effects.ClaimableBalanceSponsorshipRemoved)
		tt.Equal(master.Address(), cbSponsorshipRemoved.Account)
		tt.Equal(cbID, cbSponsorshipRemoved.BalanceID)
		tt.Equal(master.Address(), cbSponsorshipRemoved.FormerSponsor)
	}
}

func TestHappySetTrustLineFlags(t *testing.T) {
	tt := assert.New(t)
	itest := NewProtocol16Test(t)
	master := itest.Master()

	// Give the master account the revocable flag (needed to set the clawback flag)
	setRevocableFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
		},
	}

	itest.MustSubmitOperations(itest.MasterAccount(), master, &setRevocableFlag)

	// Give the master account the clawback flag
	setClawBackFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthClawbackEnabled,
		},
	}
	itest.MustSubmitOperations(itest.MasterAccount(), master, &setClawBackFlag)

	// Make sure the clawback flag was set
	accountDetails := itest.MustGetAccount(master)
	tt.True(accountDetails.Flags.AuthClawbackEnabled)

	// Create another account fot the Trustline
	keyPairs, accounts := itest.CreateAccounts(1, "100")
	accountKeyPair := keyPairs[0]
	account := accounts[0]

	// Time machine to Spain before Euros were a thing
	pesetasAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: master.Address()}
	itest.MustEstablishTrustline(accountKeyPair, account, pesetasAsset)
	// Confirm that the Trustline has the clawback flag
	accountDetails = itest.MustGetAccount(accountKeyPair)
	if tt.Len(accountDetails.Balances, 2) {
		pts := accountDetails.Balances[0]
		tt.Equal("PTS", pts.Code)
		if tt.NotNil(pts.IsClawbackEnabled) {
			tt.True(*pts.IsClawbackEnabled)
		}
	}

	// Clear the clawback flag
	setTrustlineFlags := txnbuild.SetTrustLineFlags{
		Trustor: accountKeyPair.Address(),
		Asset:   pesetasAsset,
		ClearFlags: []txnbuild.TrustLineFlag{
			txnbuild.TrustLineClawbackEnabled,
		},
	}
	submissionResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &setTrustlineFlags)

	// make sure it was cleared
	accountDetails = itest.MustGetAccount(accountKeyPair)
	if tt.Len(accountDetails.Balances, 2) {
		pts := accountDetails.Balances[0]
		tt.Equal("PTS", pts.Code)
		tt.Nil(pts.IsClawbackEnabled)
	}

	// Check the operation details
	opDetailsResponse, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: submissionResp.Hash,
	})
	tt.NoError(err)
	if tt.Len(opDetailsResponse.Embedded.Records, 1) {
		setTrustlineFlagsDetails := opDetailsResponse.Embedded.Records[0].(operations.SetTrustLineFlags)
		tt.Equal("PTS", setTrustlineFlagsDetails.Code)
		tt.Equal(master.Address(), setTrustlineFlagsDetails.Issuer)
		tt.Equal(accountKeyPair.Address(), setTrustlineFlagsDetails.Trustor)
		if tt.Len(setTrustlineFlagsDetails.ClearFlags, 1) {
			tt.True(xdr.TrustLineFlags(setTrustlineFlagsDetails.ClearFlags[0]).IsClawbackEnabledFlag())
		}
		if tt.Len(setTrustlineFlagsDetails.ClearFlagsS, 1) {
			tt.Equal(setTrustlineFlagsDetails.ClearFlagsS[0], "clawback_enabled")
		}
		tt.Len(setTrustlineFlagsDetails.SetFlags, 0)
		tt.Len(setTrustlineFlagsDetails.SetFlagsS, 0)
	}

	// Check the operation effects
	effectsResponse, err := itest.Client().Effects(horizonclient.EffectRequest{
		ForTransaction: submissionResp.Hash,
	})
	tt.NoError(err)

	if tt.Len(effectsResponse.Embedded.Records, 1) {
		trustlineFlagsUpdated := effectsResponse.Embedded.Records[0].(effects.TrustlineFlagsUpdated)
		tt.Equal(master.Address(), trustlineFlagsUpdated.Account)
		tt.Equal(master.Address(), trustlineFlagsUpdated.Issuer)
		tt.Equal("PTS", trustlineFlagsUpdated.Code)
		tt.Nil(trustlineFlagsUpdated.Authorized)
		tt.Nil(trustlineFlagsUpdated.AuthorizedToMaintainLiabilities)
		if tt.NotNil(trustlineFlagsUpdated.ClawbackEnabled) {
			tt.False(*trustlineFlagsUpdated.ClawbackEnabled)
		}
	}

	// Try to set the clawback flag (we shouldn't be able to)
	setTrustlineFlags = txnbuild.SetTrustLineFlags{
		Trustor: accountKeyPair.Address(),
		Asset:   pesetasAsset,
		SetFlags: []txnbuild.TrustLineFlag{
			txnbuild.TrustLineClawbackEnabled,
		},
	}
	_, err = itest.SubmitOperations(itest.MasterAccount(), master, &setTrustlineFlags)
	if tt.Error(err) {
		clientErr, ok := err.(*horizonclient.Error)
		if tt.True(ok) {
			tt.Equal(400, clientErr.Problem.Status)
			resCodes, err := clientErr.ResultCodes()
			tt.NoError(err)
			tt.Equal(codes.OpMalformed, resCodes.OperationCodes[0])
		}

	}

}
