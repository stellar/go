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

			// We should be able to always[^1] create & claim a balance even if
			// there's a relative time predicate.
			//
			// [^1]: Almost* always is more accurate, since if it's a *really*
			//       short timeline (see the TestComplexPredicates/Expire test)
			//       there's not enough time to form & submit the request into a
			//       new ledger before the previous one expires. Basically, all
			//       bets are off for anything < $LEDGER_CLOSE_TIME.
			"BeforeFast": txnbuild.BeforeRelativeTimePredicate(10),
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
	keys, accounts := itest.CreateAccounts(3, "10000")
	a, b, c := keys[0], keys[1], keys[2]
	_, accountB, accountC := accounts[0], accounts[1], accounts[2]

	// reused a lot:
	cantClaimResult, _ := codes.String(xdr.ClaimClaimableBalanceResultCodeClaimClaimableBalanceCannotClaim)

	// This one can be claimed within X seconds or after Y seconds but NOT
	// in-between; we check to make sure all conditions are upheld.
	asset := txnbuild.NativeAsset{}
	t.Run("BeforeOrAfter/Rel", func(t *testing.T) {
		accountA := itest.MustGetAccount(a)

		predicate := txnbuild.OrPredicate(
			txnbuild.BeforeRelativeTimePredicate(30),
			txnbuild.NotPredicate(txnbuild.BeforeRelativeTimePredicate(60)),
		)
		t.Log("Creating claimable balances...")
		_ = itest.MustSubmitOperations(&accountA, a,
			&txnbuild.CreateClaimableBalance{
				Destinations: []txnbuild.Claimant{
					txnbuild.NewClaimant(b.Address(), &predicate),
				},
				Asset:  asset,
				Amount: "123",
			},
			&txnbuild.CreateClaimableBalance{
				Destinations: []txnbuild.Claimant{
					txnbuild.NewClaimant(c.Address(), &predicate),
				},
				Asset:  asset,
				Amount: "456",
			},
		)

		// Ensure it exists in the global list
		balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
		assert.NoError(t, err)

		claims := balances.Embedded.Records
		assert.Len(t, claims, 2)
		claim1, claim2 := claims[0], claims[1]
		t.Logf("   %s", claim1.BalanceID)
		t.Logf("   %s", claim2.BalanceID)

		// First try claiming immediately.
		t.Logf("Claiming balance before expiration (ID=%s)...", claim1.BalanceID)
		t.Logf("  %s", time.Now().String())
		_, err = itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim1.BalanceID})
		assert.NoError(t, err)
		t.Log("  claimed")

		// Should fail after the first predicate expires
		time.Sleep(40 * time.Second)

		t.Logf("Claiming balance DURING lull (ID=%s)...", claim2.BalanceID)
		t.Logf("  %s", time.Now().String())
		_, err = itest.SubmitOperations(accountC, c,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim2.BalanceID})
		assert.Error(t, err)

		assert.Equal(t, cantClaimResult, getOperationsError(err))
		t.Log("  failed as expected")

		// Shouldn't fail after the second predicate kicks in
		time.Sleep(40 * time.Second)

		t.Logf("Claiming balance AFTER lull (ID=%s)...", claim2.BalanceID)
		t.Log(time.Now().String())
		_, err = itest.SubmitOperations(accountC, c,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim2.BalanceID})
		assert.NoError(t, err)
		t.Log("  claimed")

		// Ensure both are gone now
		balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
		assert.NoError(t, err)
		assert.Len(t, balances.Embedded.Records, 0)
		t.Log("All claimables gone.")
	})

	// In this one, we use absolute time to refine our conditions, instead. The
	// reason why the times are still high is because the "preparation" ops
	// themselves take time (creating CBs), so we need to pad for that.
	t.Run("BeforeOrAfter/Abs", func(t *testing.T) {
		accountA := itest.MustGetAccount(a)

		twentySecFromNow := time.Now().Add(20 * time.Second)
		thirtySecFromNow := twentySecFromNow.Add(10 * time.Second)
		predicate := txnbuild.OrPredicate(
			txnbuild.BeforeAbsoluteTimePredicate(twentySecFromNow.Unix()),
			txnbuild.NotPredicate(
				txnbuild.BeforeAbsoluteTimePredicate(thirtySecFromNow.Unix()),
			),
		)

		t.Log("Creating claimable balances...")
		_ = itest.MustSubmitOperations(&accountA, a,
			&txnbuild.CreateClaimableBalance{
				Destinations: []txnbuild.Claimant{
					txnbuild.NewClaimant(b.Address(), &predicate),
				},
				Asset:  asset,
				Amount: "123",
			},
			&txnbuild.CreateClaimableBalance{
				Destinations: []txnbuild.Claimant{
					txnbuild.NewClaimant(c.Address(), &predicate),
				},
				Asset:  asset,
				Amount: "456",
			},
		)

		// Ensure both exist in the global list
		balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
		assert.NoError(t, err)

		claims := balances.Embedded.Records
		assert.Len(t, claims, 1)
		claim1, claim2 := claims[0], claims[1]
		t.Logf("   %s", claim1.BalanceID)
		t.Logf("   %s", claim2.BalanceID)

		// First try claiming immediately.
		t.Logf("Claiming balance before expiration (ID=%s)...", claim1.BalanceID)
		t.Log(time.Now().String())
		_, err = itest.SubmitOperations(accountB, b,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim1.BalanceID})
		assert.NoError(t, err)
		t.Log("  claimed")

		// Should fail after ~15-20s
		time.Sleep(20 * time.Second)

		t.Logf("Claiming balance DURING lull (ID=%s)...", claim2.BalanceID)
		t.Log(time.Now().String())
		_, err = itest.SubmitOperations(accountC, c,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim2.BalanceID})
		assert.Error(t, err)

		assert.Equal(t, cantClaimResult, getOperationsError(err))
		t.Log("  failed as expected")

		// Shouldn't fail after another ~10s
		time.Sleep(10 * time.Second)

		t.Logf("Claiming balance AFTER lull (ID=%s)...", claim2.BalanceID)
		t.Log(time.Now().String())
		_, err = itest.SubmitOperations(accountC, c,
			&txnbuild.ClaimClaimableBalance{BalanceID: claim2.BalanceID})
		assert.NoError(t, err)
		t.Log("  success")

		// Ensure both are gone now
		balances, err = client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
		assert.NoError(t, err)
		assert.Len(t, balances.Embedded.Records, 0)
		t.Log("All claimables gone.")
	})

	//
	// These two are done last because they don't clean up after themselves,
	// making it harder to do exact length checks in a test.
	//

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

		assert.Equal(t, cantClaimResult, getOperationsError(err))
		t.Logf("  tx did fail w/ %s", cantClaimResult)
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
