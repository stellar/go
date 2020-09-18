package integration

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/stellar/go/clients/horizonclient"
	proto "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/codes"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var protocol14Config = test.IntegrationConfig{ProtocolVersion: 14}

func TestProtocol14Basics(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	t.Run("SanityCheck", func(t *testing.T) {
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
	})

	t.Run("ClaimableBalanceCreation", func(t *testing.T) {
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
	})
}

func TestHappyClaimableBalances(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	master, client := itest.Master(), itest.Client()

	keys, accounts := itest.CreateAccounts(3, "1000")
	a, b, c := keys[0], keys[1], keys[2]
	accountA, accountB, accountC := accounts[0], accounts[1], accounts[2]

	/*
	 * Each sub-test is completely self-contained: at the end of the test, we
	 * start with a clean slate for each account. This lets us check with
	 * equality for things like "number of operations," etc.
	 */

	// We start simple: native asset, single destination, no predicate.
	t.Run("Simple/Native", func(t *testing.T) {
		// Note that we don't use the `itest.MustCreateClaimableBalance` helper
		// here because the whole point is to check that ^ generally works.
		t.Logf("Creating claimable balance.")
		_, err := itest.SubmitOperations(accountA, a,
			&txnbuild.CreateClaimableBalance{
				Destinations: []txnbuild.Claimant{
					txnbuild.NewClaimant(b.Address(), nil),
					txnbuild.NewClaimant(c.Address(), nil),
				},
				Asset:  txnbuild.NativeAsset{},
				Amount: "42",
			},
		)
		assert.NoError(t, err)

		//
		// Ensure it shows up with the various filters (and *doesn't* show up with
		// non-matching filters, of course).
		//
		// TODO: should we make these all subtests?
		//
		t.Log("Checking claimable balance filters")

		// Ensure it exists in the global list
		t.Log("  global")
		balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
		assert.NoError(t, err)

		claims := balances.Embedded.Records
		assert.Len(t, claims, 1)
		assert.Equal(t, a.Address(), claims[0].Sponsor)
		claim := claims[0]

		// Ensure we can look it up explicitly
		t.Log("  by ID")
		balance, err := client.ClaimableBalance(claim.BalanceID)
		assert.NoError(t, err)
		assert.Equal(t, claim, balance)

		checkFilters(itest, claim)

		for _, assetType := range []txnbuild.AssetType{
			txnbuild.AssetTypeCreditAlphanum4,
			txnbuild.AssetTypeCreditAlphanum12,
		} {
			t.Logf("  by non-native %+v", assetType)
			randomAsset := createAsset(assetType, a.Address())
			xdrAsset, innerErr := randomAsset.ToXDR()
			assert.NoError(t, innerErr)

			balances, innerErr = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Asset: xdrAsset.StringCanonical()})
			assert.NoError(t, innerErr)
			assert.Len(t, balances.Embedded.Records, 0)
		}

		//
		// Now, actually try to *claim* the CB to remove it from the global list.
		//

		// Claiming a balance when you aren't the recipient should fail...
		t.Logf("Stealing balance (ID=%s)...", claim.BalanceID)
		_, err = itest.SubmitOperations(accountA, a,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.Error(t, err)
		t.Log("  failed as expected")

		// ...but if you are it should succeed.
		t.Logf("Claiming balance (ID=%s)...", claim.BalanceID)
		_, err = itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.NoError(t, err)
		t.Log("  claimed")

		// Ensure the claimable balance is gone now.
		balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: a.Address()})
		assert.NoError(t, err)
		assert.Len(t, balances.Embedded.Records, 0)
		t.Log("  gone")

		// Ensure the actual account has a higher balance, now!
		request := sdk.AccountRequest{AccountID: b.Address()}
		details, err := client.AccountDetail(request)
		assert.NoError(t, err)

		foundBalance := false
		for _, balance := range details.Balances {
			if balance.Code != "" {
				continue
			}

			assert.Equal(t, "1041.9999900", balance.Balance) // 1000 + 42 - fee
			foundBalance = true
			break
		}
		assert.True(t, foundBalance)

		// Ensure that the other claimant can't do anything about it!
		t.Log("  other claimant can't claim")
		_, err = itest.SubmitOperations(accountC, c,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.Error(t, err)
	})

	// Now, confirm the same thing works for non-native assets.
	for _, assetType := range []txnbuild.AssetType{
		txnbuild.AssetTypeCreditAlphanum4,
		txnbuild.AssetTypeCreditAlphanum12,
	} {
		t.Run(fmt.Sprintf("Simple/%+v", assetType), func(t *testing.T) {
			asset := createAsset(assetType, a.Address())
			itest.MustEstablishTrustline(b, accountB, asset)

			t.Log("Creating claimable balance.")
			claim := itest.MustCreateClaimableBalance(a, asset, "42",
				txnbuild.NewClaimant(b.Address(), nil))

			//
			// Ensure it shows up with the various filters (and *doesn't* show up with
			// non-matching filters, of course).
			//
			t.Log("Checking claimable balance filters")

			// Ensure we can look it up explicitly
			t.Log("  by ID")
			balance, err := client.ClaimableBalance(claim.BalanceID)
			assert.NoError(t, err)
			assert.Equal(t, claim, balance)

			checkFilters(itest, claim)

			t.Logf("  by native")
			xdrAsset, err := txnbuild.NativeAsset{}.ToXDR()
			balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{Asset: xdrAsset.StringCanonical()})
			assert.NoError(t, err)
			assert.Len(t, balances.Embedded.Records, 0)

			// Even if the native asset filter doesn't match, we need to ensure
			// that a different credit asset also doesn't match.
			t.Logf("  by random asset")
			xdrAsset, err = txnbuild.CreditAsset{Code: "RAND", Issuer: master.Address()}.ToXDR()
			balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Asset: xdrAsset.StringCanonical()})
			assert.NoError(t, err)
			assert.Len(t, balances.Embedded.Records, 0)

			//
			// Now, actually try to *claim* the CB to remove it from the global list.
			//
			t.Logf("Claiming balance (ID=%s)...", claim.BalanceID)
			_, err = itest.SubmitOperations(accountB, b,
				&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
			assert.NoError(t, err)
			t.Log("  claimed")

			// Ensure the claimable balance is gone now.
			balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: a.Address()})
			assert.NoError(t, err)
			assert.Len(t, balances.Embedded.Records, 0)
			t.Log("  gone")

			// Ensure the actual account has a higher balance, now!
			account := itest.MustGetAccount(b)
			foundBalance := false
			for _, balance := range account.Balances {
				if balance.Code != asset.GetCode() || balance.Issuer != asset.GetIssuer() {
					continue
				}

				assert.Equal(t, "42.0000000", balance.Balance)
				foundBalance = true
				break
			}
			assert.True(t, foundBalance)
		})
	}

	t.Run("Predicates", func(t *testing.T) {
		now := time.Now().Unix()
		minute := int64(60 * 60)

		//
		// All of these predicates should succeed with no issues.
		//
		for description, predicate := range map[string]xdr.ClaimPredicate{
			"None":          txnbuild.UnconditionalPredicate,
			"BeforeAbsTime": txnbuild.BeforeAbsoluteTimePredicate(now + minute), // full minute to claim
			"BeforeRelTime": txnbuild.BeforeRelativeTimePredicate(minute),
			"BeforeBoth": txnbuild.AndPredicate(
				txnbuild.BeforeAbsoluteTimePredicate(now+minute),
				txnbuild.BeforeRelativeTimePredicate(minute),
			),
			"BeforeEither": txnbuild.OrPredicate(
				txnbuild.BeforeAbsoluteTimePredicate(now+minute),
				txnbuild.BeforeRelativeTimePredicate(minute),
			),
		} {
			t.Run(description, func(t *testing.T) {
				t.Logf("Creating claimable balance (asset=native).")
				t.Logf("  predicate: %+v", predicate.Type)

				claim := itest.MustCreateClaimableBalance(
					a, txnbuild.NativeAsset{}, "42",
					txnbuild.NewClaimant(b.Address(), &predicate))

				t.Logf("Claiming balance (ID=%s)...", claim.BalanceID)
				_, err := itest.SubmitOperations(accountB, b,
					&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
				assert.NoError(t, err)
				t.Log("  claimed")

				balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: a.Address()})
				assert.NoError(t, err)
				assert.Len(t, balances.Embedded.Records, 0)
				t.Log("  gone")
			})
		}
	})
}

// We want to ensure that users can't claim the same claimable balance twice.
func TestDoubleClaim(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	client := itest.Client()

	// Create a couple of accounts to test the interactions.
	keys, accounts := itest.CreateAccounts(2, "1000")
	a, b := keys[0], keys[1]
	_, accountB := accounts[0], accounts[1]

	notExistResult, _ := codes.String(xdr.ClaimClaimableBalanceResultCodeClaimClaimableBalanceDoesNotExist)

	// Two cases: claim in separate TXs, claim twice in same TX
	t.Run("TwoTx", func(t *testing.T) {
		claim := itest.MustCreateClaimableBalance(
			a, txnbuild.NativeAsset{}, "42",
			txnbuild.NewClaimant(b.Address(), nil))

		t.Logf("Claiming balance (ID=%s)...", claim.BalanceID)
		_, err := itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.NoError(t, err)
		t.Log("  claimed")

		_, err = itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.Error(t, err)
		t.Log("  couldn't claim twice")

		assert.Equal(t, notExistResult, getOperationsError(err))
	})

	t.Run("SameTx", func(t *testing.T) {
		claim := itest.MustCreateClaimableBalance(
			a, txnbuild.NativeAsset{}, "42",
			txnbuild.NewClaimant(b.Address(), nil))

		// One succeeds, other fails
		t.Logf("Claiming balance (ID=%s)...", claim.BalanceID)
		_, err := itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID},
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.Error(t, err)
		t.Log("  couldn't claim twice")

		assert.Equal(t, codes.OpSuccess, getOperationsErrorByIndex(err, 0))
		assert.Equal(t, notExistResult, getOperationsErrorByIndex(err, 1))

		// Both included in /operations
		response, err := client.Operations(sdk.OperationRequest{
			ForAccount:    b.Address(),
			Order:         "desc",
			Limit:         2,
			IncludeFailed: true,
		})
		ops := response.Embedded.Records
		assert.NoError(t, err)
		assert.Len(t, ops, 2)
	})
}

func TestComplexPredicates(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	_, client := itest.Master(), itest.Client()

	// Create a couple of accounts to test the interactions.
	keys, accounts := itest.CreateAccounts(3, "1000")
	a, b, _ := keys[0], keys[1], keys[2]
	_, accountB, _ := accounts[0], accounts[1], accounts[2]

	// reused a lot:
	cantClaimResult, _ := codes.String(xdr.ClaimClaimableBalanceResultCodeClaimClaimableBalanceCannotClaim)

	// This is an easy fail.
	predicate := txnbuild.NotPredicate(txnbuild.UnconditionalPredicate)
	t.Run("AlwaysFail", func(t *testing.T) {
		t.Logf("Creating claimable balance (asset=native).")
		t.Logf("  predicate: %+v", predicate.Type)

		claim := itest.MustCreateClaimableBalance(
			a, txnbuild.NativeAsset{}, "42",
			txnbuild.NewClaimant(b.Address(), &predicate))

		t.Logf("Claiming balance (ID=%s)...", claim.BalanceID)
		_, err := itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.Error(t, err)

		// Ensure it failed w/ the right error code:
		//  CLAIM_CLAIMABLE_BALANCE_CANNOT_CLAIM
		assert.Equal(t, cantClaimResult, getOperationsError(err))
		t.Logf("  tx did fail w/ %s", cantClaimResult)

		// check that /operations also has the claim as failed
		response, err := client.Operations(sdk.OperationRequest{
			Order:         "desc",
			Limit:         1,
			IncludeFailed: true,
		})
		ops := response.Embedded.Records
		assert.NoError(t, err)
		assert.Len(t, ops, 1)

		cb := ops[0].(operations.ClaimClaimableBalance)
		assert.False(t, cb.TransactionSuccessful)
		assert.Equal(t, claim.BalanceID, cb.BalanceID)
		assert.Equal(t, b.Address(), cb.Claimant)
		t.Log("  op did fail")
	})

	// This one fails because of an expiring claim.
	predicate = txnbuild.BeforeRelativeTimePredicate(1)
	t.Run("Expire", func(t *testing.T) {
		t.Log("Creating claimable balance (asset=native).")
		t.Logf("  predicate: %+v", predicate.Type)

		claim := itest.MustCreateClaimableBalance(
			a, txnbuild.NativeAsset{}, "42",
			txnbuild.NewClaimant(b.Address(), &predicate))

		oneSec, err := time.ParseDuration("1s")
		time.Sleep(oneSec)

		t.Logf("Claiming balance (ID=%s)...", claim.BalanceID)
		_, err = itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim.BalanceID})
		assert.Error(t, err)
	})
}

/* Utility functions below */

// Extracts the first error string in the "operations: [...]" of a Problem.
func getOperationsError(err error) string {
	return getOperationsErrorByIndex(err, 0)
}

func getOperationsErrorByIndex(err error, i int) string {
	resultCodes := sdk.GetError(err).Problem.Extras["result_codes"].(map[string]interface{})
	opResultCodes := resultCodes["operations"].([]interface{})
	return opResultCodes[i].(string)
}

// Checks that filtering works for a particular claim.
func checkFilters(i *test.IntegrationTest, claim proto.ClaimableBalance) {
	client := i.Client()
	t := i.CurrentTest()

	source := claim.Sponsor
	asset := claim.Asset

	t.Log("  by sponsor")
	balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: source})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 1)
	assert.Equal(t, claim, balances.Embedded.Records[0])

	dest := claim.Claimants[0].Destination
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Sponsor: dest})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)

	t.Log("  by claimant(s)")
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Claimant: source})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 0)

	for _, claimant := range claim.Claimants {
		dest := claimant.Destination
		balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Claimant: dest})
		assert.NoError(t, err)
		assert.Len(t, balances.Embedded.Records, 1)
		assert.Equal(t, claim, balances.Embedded.Records[0])
	}

	t.Log("  by exact asset")
	balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{Asset: asset})
	assert.NoError(t, err)
	assert.Len(t, balances.Embedded.Records, 1)
}

// Creates an asset object given a type and issuer.
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
