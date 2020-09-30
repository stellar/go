package integration

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

func sponsorOperations(account string, ops ...txnbuild.Operation) []txnbuild.Operation {
	return append(append(
		[]txnbuild.Operation{
			&txnbuild.BeginSponsoringFutureReserves{SponsoredID: account},
		},
		ops...),
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: &txnbuild.SimpleAccount{AccountID: account},
		},
	)
}

func findOperationByID(needle string, haystack []operations.Operation) func() bool {
	// usable by assert.Condition
	return func() bool {
		for _, o := range haystack {
			if o.GetID() == needle {
				return true
			}
		}
		return false
	}
}

func TestSponsorships(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	client := itest.Client()

	getOperationsByTx := func(txHash string) []operations.Operation {
		response, err := client.Operations(sdk.OperationRequest{ForTransaction: txHash})
		tt.NoError(err)
		return response.Embedded.Records
	}

	getEffectsByOp := func(opId string) []effects.Effect {
		response, err := client.Effects(sdk.EffectRequest{ForOperation: opId})
		tt.NoError(err)
		return response.Embedded.Records
	}

	getEffectsByTx := func(txId string) []effects.Effect {
		response, err := client.Effects(sdk.EffectRequest{ForTransaction: txId})
		tt.NoError(err)
		return response.Embedded.Records
	}

	//
	// Each test has its own sponsor and sponsoree (or is it sponsee?
	// :thinking:) so that we can do direct equality checks.
	//
	// Each sub-test follows a similar structure:
	//   - sponsor a particular operation
	//   - replace the sponsor with a new one
	//   - revoke the sponsorship
	//
	// Between each step, we validate /operations, /effects, etc.
	//

	// We will create the following operation structure:
	// BeginSponsoringFutureReserves A
	//   CreateAccount A
	// EndSponsoringFutureReserves (with A as a source)
	t.Run("CreateAccount", func(t *testing.T) {
		keys, accounts := itest.CreateAccounts(2, "1000")
		sponsor, sponsorPair := accounts[0], keys[0]
		newAccountKeys := keypair.MustRandom()
		newAccountID := newAccountKeys.Address()

		t.Logf("Testing sponsorship of CreateAccount operation")
		ops := sponsorOperations(newAccountID,
			&txnbuild.CreateAccount{
				Destination: newAccountID,
				Amount:      "100",
			})

		signers := []*keypair.Full{sponsorPair, newAccountKeys}
		txResp, err := itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Ensure that the operations are in fact the droids we're looking for
		opRecords := getOperationsByTx(txResp.Hash)
		tt.Len(opRecords, 3)
		tt.True(opRecords[0].IsTransactionSuccessful())

		startSponsoringOp := opRecords[0].(operations.BeginSponsoringFutureReserves)
		actualCreateAccount := opRecords[1].(operations.CreateAccount)
		endSponsoringOp := opRecords[2].(operations.EndSponsoringFutureReserves)

		tt.Equal(newAccountID, startSponsoringOp.SponsoredID)
		tt.Equal(sponsorPair.Address(), actualCreateAccount.Sponsor)
		tt.Equal(sponsorPair.Address(), endSponsoringOp.BeginSponsor)

		// Make sure that the sponsor is an (implicit) participant on the end
		// sponsorship operation
		response, err := client.Operations(sdk.OperationRequest{ForAccount: sponsorPair.Address()})
		tt.Condition(findOperationByID(endSponsoringOp.ID, response.Embedded.Records))
		t.Logf("  operations accurate")

		// Check that the num_sponsoring and num_sponsored fields are accurate
		tt.EqualValues(2, itest.MustGetAccount(sponsorPair).NumSponsoring)
		tt.EqualValues(2, itest.MustGetAccount(newAccountKeys).NumSponsored)
		t.Logf("  accounts accurate")

		// Check effects of CreateAccount Operation
		effectRecords := getEffectsByOp(actualCreateAccount.GetID())
		tt.Len(effectRecords, 4)
		tt.Equal(sponsorPair.Address(),
			effectRecords[3].(effects.AccountSponsorshipCreated).Sponsor)
		t.Logf("  effects accurate")

		// Update sponsor
		newSponsorPair, newSponsor := keys[1], accounts[1]

		t.Logf("Revoking & replacing sponsorship")
		ops = []txnbuild.Operation{
			&txnbuild.BeginSponsoringFutureReserves{
				SourceAccount: newSponsor,
				SponsoredID:   sponsorPair.Address(),
			},
			&txnbuild.RevokeSponsorship{
				SourceAccount:   sponsor,
				SponsorshipType: txnbuild.RevokeSponsorshipTypeAccount,
				Account:         &newAccountID,
			},
			&txnbuild.EndSponsoringFutureReserves{},
		}

		signers = []*keypair.Full{sponsorPair, newSponsorPair}
		txResp, err = itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify operation details
		response, err = client.Operations(sdk.OperationRequest{
			ForTransaction: txResp.Hash,
		})
		tt.NoError(err)
		opRecords = response.Embedded.Records
		tt.Len(opRecords, 3)
		tt.True(opRecords[1].IsTransactionSuccessful())

		revokeOp := opRecords[1].(operations.RevokeSponsorship)
		tt.Equal(newAccountID, *revokeOp.AccountID)
		t.Logf("  operations accurate")

		// Check effects
		effectRecords = getEffectsByOp(revokeOp.ID)
		tt.Len(effectRecords, 1)
		effect := effectRecords[0].(effects.AccountSponsorshipUpdated)
		tt.Equal(sponsorPair.Address(), effect.FormerSponsor)
		tt.Equal(newSponsorPair.Address(), effect.NewSponsor)
		t.Logf("  effects accurate")

		// Revoke sponsorship

		t.Logf("Revoking sponsorship entirely")
		op := &txnbuild.RevokeSponsorship{
			SponsorshipType: txnbuild.RevokeSponsorshipTypeAccount,
			Account:         &newAccountID,
		}
		txResp = itest.MustSubmitOperations(newSponsor, newSponsorPair, op)

		// Verify operation details
		opRecords = getOperationsByTx(txResp.Hash)
		tt.Len(opRecords, 1)
		tt.True(opRecords[0].IsTransactionSuccessful())
		revokeOp = opRecords[0].(operations.RevokeSponsorship)
		tt.Equal(newAccountID, *revokeOp.AccountID)

		// Make sure that the sponsoree is an (implicit) participant in the
		// revocation operation
		response, err = client.Operations(sdk.OperationRequest{ForAccount: newAccountID})
		tt.Condition(findOperationByID(revokeOp.ID, response.Embedded.Records))
		t.Logf("  operations accurate")

		// Check effects
		effectRecords = getEffectsByOp(revokeOp.ID)
		tt.Len(effectRecords, 1)
		tt.IsType(effects.AccountSponsorshipRemoved{}, effectRecords[0])
		desponsorOp := effectRecords[0].(effects.AccountSponsorshipRemoved)
		tt.Equal(newSponsorPair.Address(), desponsorOp.FormerSponsor)
		t.Logf("  effects accurate")
	})

	// Let's add a sponsored data entry
	// BeginSponsorship N (Source=sponsor)
	//   SetOptionsSigner (Source=N)
	// EndSponsorship (Source=N)
	t.Run("Signer", func(t *testing.T) {
		keys, accounts := itest.CreateAccounts(3, "1000")
		sponsorPair, sponsor := keys[0], accounts[0]
		newAccountPair, newAccount := keys[1], accounts[1]
		signerKey := keypair.MustRandom().Address() // unspecified signer

		ops := sponsorOperations(newAccountPair.Address(), &txnbuild.SetOptions{
			SourceAccount: newAccount,
			Signer: &txnbuild.Signer{
				Address: signerKey,
				Weight:  1,
			},
		})

		signers := []*keypair.Full{sponsorPair, newAccountPair}
		txResp, err := itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify that the signer was incorporated
		signerAdded := func() bool {
			signers := itest.MustGetAccount(newAccountPair).Signers
			for _, signer := range signers {
				if signer.Key == signerKey {
					tt.Equal(sponsorPair.Address(), signer.Sponsor)
					return true
				}
			}
			return false
		}
		tt.Condition(signerAdded)

		// Check effects and details of the SetOptions operation
		opRecords := getOperationsByTx(txResp.Hash)
		tt.Len(opRecords, 3)
		setOptionsOp := opRecords[1].(operations.SetOptions)
		tt.Equal(sponsorPair.Address(), setOptionsOp.Sponsor)

		effRecords := getEffectsByOp(setOptionsOp.GetID())
		tt.Len(effRecords, 2)
		signerSponsorshipEffect := effRecords[1].(effects.SignerSponsorshipCreated)
		tt.Equal(sponsorPair.Address(), signerSponsorshipEffect.Sponsor)
		tt.Equal(newAccountPair.Address(), signerSponsorshipEffect.Account)
		tt.Equal(signerKey, signerSponsorshipEffect.Signer)

		// Update sponsor
		newSponsorPair, newSponsor := keys[2], accounts[2]
		ops = []txnbuild.Operation{
			&txnbuild.BeginSponsoringFutureReserves{
				SourceAccount: newSponsor,
				SponsoredID:   sponsorPair.Address(),
			},
			&txnbuild.RevokeSponsorship{
				SponsorshipType: txnbuild.RevokeSponsorshipTypeSigner,
				Signer: &txnbuild.SignerID{
					AccountID:     newAccountPair.Address(),
					SignerAddress: signerKey,
				},
			},
			&txnbuild.EndSponsoringFutureReserves{},
		}

		signers = []*keypair.Full{sponsorPair, newSponsorPair}
		txResp, err = itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify operation details
		opRecords = getOperationsByTx(txResp.Hash)
		tt.NoError(err)
		tt.Len(opRecords, 3)
		tt.True(opRecords[1].IsTransactionSuccessful())

		revokeOp := opRecords[1].(operations.RevokeSponsorship)
		tt.Equal(newAccountPair.Address(), *revokeOp.SignerAccountID)
		tt.Equal(signerKey, *revokeOp.SignerKey)

		// Check effects
		effRecords = getEffectsByOp(revokeOp.ID)
		tt.Len(effRecords, 1)
		effect := effRecords[0].(effects.SignerSponsorshipUpdated)
		tt.Equal(sponsorPair.Address(), effect.FormerSponsor)
		tt.Equal(newSponsorPair.Address(), effect.NewSponsor)
		tt.Equal(signerKey, effect.Signer)

		// Revoke sponsorship

		revoke := txnbuild.RevokeSponsorship{
			SponsorshipType: txnbuild.RevokeSponsorshipTypeSigner,
			Signer: &txnbuild.SignerID{
				AccountID:     newAccountPair.Address(),
				SignerAddress: signerKey,
			},
		}
		txResp = itest.MustSubmitOperations(newSponsor, newSponsorPair, &revoke)

		effRecords = getEffectsByTx(txResp.ID)
		tt.Len(effRecords, 1)
		sponsorshipRemoved := effRecords[0].(effects.SignerSponsorshipRemoved)
		tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
		tt.Equal(signerKey, sponsorshipRemoved.Signer)
	})

	// Let's add a sponsored preauth signer with a transaction:
	//
	// BeginSponsorship N (Source=sponsor)
	//   SetOptionsSigner preAuthHash (Source=N)
	// EndSponsorship (Source=N)
	t.Run("PreAuthSigner", func(t *testing.T) {
		keys, accounts := itest.CreateAccounts(2, "1000")
		sponsorPair, sponsor := keys[0], accounts[0]
		newAccountPair, newAccount := keys[1], accounts[1]

		// unspecified signer
		randomSigner := keypair.MustRandom().Address()

		// Let's create a preauthorized transaction for the new account
		// to add a signer
		preAuthOp := &txnbuild.SetOptions{
			Signer: &txnbuild.Signer{
				Address: randomSigner,
				Weight:  1,
			},
		}
		txParams := txnbuild.TransactionParams{
			SourceAccount:        newAccount,
			Operations:           []txnbuild.Operation{preAuthOp},
			BaseFee:              txnbuild.MinBaseFee,
			Timebounds:           txnbuild.NewInfiniteTimeout(),
			IncrementSequenceNum: true,
		}
		preaAuthTx, err := txnbuild.NewTransaction(txParams)
		tt.NoError(err)
		preAuthHash, err := preaAuthTx.Hash(test.IntegrationNetworkPassphrase)
		tt.NoError(err)
		preAuthTxB64, err := preaAuthTx.Base64()
		tt.NoError(err)

		// Add a sponsored preauth signer with the above transaction.
		preAuthSignerKey := xdr.SignerKey{
			Type:      xdr.SignerKeyTypeSignerKeyTypePreAuthTx,
			PreAuthTx: (*xdr.Uint256)(&preAuthHash),
		}
		ops := sponsorOperations(newAccountPair.Address(),
			&txnbuild.SetOptions{
				SourceAccount: newAccount,
				Signer: &txnbuild.Signer{
					Address: preAuthSignerKey.Address(),
					Weight:  1,
				},
			})

		signers := []*keypair.Full{sponsorPair, newAccountPair}
		txResp, err := itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify that the preauth signer was incorporated
		preAuthSignerAdded := func() bool {
			for _, signer := range itest.MustGetAccount(newAccountPair).Signers {
				if preAuthSignerKey.Address() == signer.Key {
					return true
				}
			}
			return false
		}
		tt.Condition(preAuthSignerAdded)

		// Check effects and details of the SetOptions operation
		opRecords := getOperationsByTx(txResp.Hash)
		setOptionsOp := opRecords[1].(operations.SetOptions)
		tt.Equal(sponsorPair.Address(), setOptionsOp.Sponsor)

		effRecords := getEffectsByOp(setOptionsOp.GetID())
		signerSponsorshipEffect := effRecords[1].(effects.SignerSponsorshipCreated)
		tt.Equal(sponsorPair.Address(), signerSponsorshipEffect.Sponsor)
		tt.Equal(preAuthSignerKey.Address(), signerSponsorshipEffect.Signer)

		// Submit the preauthorized transaction
		var txResult xdr.TransactionResult
		tt.NoError(err)
		txResp, err = client.SubmitTransactionXDR(preAuthTxB64)
		tt.NoError(err)
		err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
		tt.NoError(err)
		tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

		// Verify that the new signer was incorporated and that the preauth signer was removed
		preAuthSignerAdded = func() bool {
			signers := itest.MustGetAccount(newAccountPair).Signers
			if len(signers) != 2 {
				return false
			}
			for _, signer := range signers {
				if signer.Key == randomSigner {
					return true
				}
			}
			return false
		}
		tt.Condition(preAuthSignerAdded)

		// We don't check effects because we don't process transaction-level changes
		// See https://github.com/stellar/go/pull/3050#discussion_r493651644
	})

	// Let's add a sponsored data entry
	//
	// BeginSponsorship N (Source=sponsor)
	//   ManageData "SponsoredData"="SponsoredValue" (Source=N)
	// EndSponsorship (Source=N)
	t.Run("Data", func(t *testing.T) {
		keys, accounts := itest.CreateAccounts(3, "1000")
		sponsorPair, sponsor := keys[0], accounts[0]
		newAccountPair, newAccount := keys[1], accounts[1]

		ops := sponsorOperations(newAccountPair.Address(),
			&txnbuild.ManageData{
				Name:          "SponsoredData",
				Value:         []byte("SponsoredValue"),
				SourceAccount: newAccount,
			})

		signers := []*keypair.Full{sponsorPair, newAccountPair}
		txResp, err := itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify that the data was incorporated
		dataAdded := func() bool {
			data := itest.MustGetAccount(newAccountPair).Data
			if value, ok := data["SponsoredData"]; ok {
				decoded, e := base64.StdEncoding.DecodeString(value)
				tt.NoError(e)
				if string(decoded) == "SponsoredValue" {
					return true
				}
			}
			return false
		}
		tt.Condition(dataAdded)

		// Check effects and details of the ManageData operation
		opRecords := getOperationsByTx(txResp.Hash)
		tt.Len(opRecords, 3)
		manageDataOp := opRecords[1].(operations.ManageData)
		tt.Equal(sponsorPair.Address(), manageDataOp.Sponsor)

		effRecords := getEffectsByOp(manageDataOp.GetID())
		tt.Len(effRecords, 2)
		dataSponsorshipEffect := effRecords[1].(effects.DataSponsorshipCreated)
		tt.Equal(sponsorPair.Address(), dataSponsorshipEffect.Sponsor)
		tt.Equal(newAccountPair.Address(), dataSponsorshipEffect.Account)
		tt.Equal("SponsoredData", dataSponsorshipEffect.DataName)

		// Update sponsor

		newSponsorPair, newSponsor := keys[2], accounts[2]
		ops = []txnbuild.Operation{
			&txnbuild.BeginSponsoringFutureReserves{
				SourceAccount: newSponsor,
				SponsoredID:   sponsorPair.Address(),
			},
			&txnbuild.RevokeSponsorship{
				SponsorshipType: txnbuild.RevokeSponsorshipTypeData,
				Data: &txnbuild.DataID{
					Account:  newAccountPair.Address(),
					DataName: "SponsoredData",
				},
			},
			&txnbuild.EndSponsoringFutureReserves{},
		}
		signers = []*keypair.Full{sponsorPair, newSponsorPair}
		txResp, err = itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify operation details
		opRecords = getOperationsByTx(txResp.Hash)
		tt.Len(opRecords, 3)
		tt.True(opRecords[1].IsTransactionSuccessful())

		revokeOp := opRecords[1].(operations.RevokeSponsorship)
		tt.Equal(newAccountPair.Address(), *revokeOp.DataAccountID)
		tt.Equal("SponsoredData", *revokeOp.DataName)

		// Check effects
		effRecords = getEffectsByOp(revokeOp.ID)
		tt.Len(effRecords, 1)
		effect := effRecords[0].(effects.DataSponsorshipUpdated)
		tt.Equal(sponsorPair.Address(), effect.FormerSponsor)
		tt.Equal(newSponsorPair.Address(), effect.NewSponsor)
		tt.Equal("SponsoredData", effect.DataName)

		// Revoke sponsorship

		revoke := txnbuild.RevokeSponsorship{
			SponsorshipType: txnbuild.RevokeSponsorshipTypeData,
			Data: &txnbuild.DataID{
				Account:  newAccountPair.Address(),
				DataName: "SponsoredData",
			},
		}
		txResp = itest.MustSubmitOperations(newSponsor, newSponsorPair, &revoke)

		effRecords = getEffectsByTx(txResp.ID)
		tt.Len(effRecords, 1)
		sponsorshipRemoved := effRecords[0].(effects.DataSponsorshipRemoved)
		tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
		tt.Equal("SponsoredData", sponsorshipRemoved.DataName)
	})

	// Let's add a sponsored trustline and offer
	//
	// BeginSponsorship N (Source=sponsor)
	//   Change Trust (ABC, sponsor) (Source=N)
	//   ManageSellOffer Buying (ABC, sponsor) (Source=N)
	// EndSponsorship (Source=N)
	t.Run("TrustlineAndOffer", func(t *testing.T) {
		keys, accounts := itest.CreateAccounts(3, "1000")
		sponsorPair, sponsor := keys[0], accounts[0]
		newAccountPair, newAccount := keys[1], accounts[1]

		asset := txnbuild.CreditAsset{Code: "ABCD", Issuer: sponsorPair.Address()}
		canonicalAsset := fmt.Sprintf("%s:%s", asset.Code, asset.Issuer)

		ops := sponsorOperations(newAccountPair.Address(),
			&txnbuild.ChangeTrust{
				SourceAccount: newAccount,
				Line:          asset,
				Limit:         txnbuild.MaxTrustlineLimit,
			},
			&txnbuild.ManageSellOffer{
				SourceAccount: newAccount,
				Selling:       txnbuild.NativeAsset{},
				Buying:        asset,
				Amount:        "3",
				Price:         "1",
			})

		signers := []*keypair.Full{sponsorPair, newAccountPair}
		txResp, err := itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify that the offer was incorporated correctly
		trustlineAdded := func() bool {
			for _, balance := range itest.MustGetAccount(newAccountPair).Balances {
				if balance.Issuer == sponsorPair.Address() {
					tt.Equal(asset.Code, balance.Code)
					tt.Equal(sponsorPair.Address(), balance.Sponsor)
					return true
				}
			}
			return false
		}
		tt.Condition(trustlineAdded)

		// Check the details of the ManageSellOffer operation
		// (there are no effects, which is intentional)
		opRecords := getOperationsByTx(txResp.Hash)
		tt.Len(opRecords, 4)
		changeTrust := opRecords[1].(operations.ChangeTrust)
		tt.Equal(sponsorPair.Address(), changeTrust.Sponsor)

		// Verify that the offer was incorporated correctly
		var offer protocol.Offer
		offerAdded := func() bool {
			offers, e := client.Offers(sdk.OfferRequest{
				ForAccount: newAccountPair.Address(),
			})
			tt.NoError(e)
			if tt.Len(offers.Embedded.Records, 1) {
				offer = offers.Embedded.Records[0]
				tt.Equal(sponsorPair.Address(), offer.Buying.Issuer)
				tt.Equal(asset.Code, offer.Buying.Code)
				tt.Equal(sponsorPair.Address(), offer.Sponsor)
				return true
			}
			return false
		}
		tt.Condition(offerAdded)

		// Check the details of the ManageSellOffer operation
		// (there are no effects, which is intentional)
		manageOffer := opRecords[2].(operations.ManageSellOffer)
		tt.Equal(sponsorPair.Address(), manageOffer.Sponsor)

		// Update sponsor

		newSponsorPair, newSponsor := keys[2], accounts[2]

		ops = []txnbuild.Operation{
			&txnbuild.BeginSponsoringFutureReserves{
				SourceAccount: newSponsor,
				SponsoredID:   sponsorPair.Address(),
			},
			&txnbuild.RevokeSponsorship{
				SponsorshipType: txnbuild.RevokeSponsorshipTypeOffer,
				Offer: &txnbuild.OfferID{
					SellerAccountAddress: offer.Seller,
					OfferID:              offer.ID,
				},
			},
			&txnbuild.RevokeSponsorship{
				SponsorshipType: txnbuild.RevokeSponsorshipTypeTrustLine,
				TrustLine: &txnbuild.TrustLineID{
					Account: newAccountPair.Address(),
					Asset:   asset,
				},
			},
			&txnbuild.EndSponsoringFutureReserves{},
		}
		signers = []*keypair.Full{sponsorPair, newSponsorPair}
		txResp, err = itest.SubmitMultiSigOperations(sponsor, signers, ops...)
		itest.LogFailedTx(txResp, err)

		// Verify operation details
		opRecords = getOperationsByTx(txResp.Hash)
		tt.Len(opRecords, 4)

		tt.True(opRecords[1].IsTransactionSuccessful())
		revokeOp := opRecords[1].(operations.RevokeSponsorship)
		tt.Equal(offer.ID, *revokeOp.OfferID)

		tt.True(opRecords[2].IsTransactionSuccessful())
		revokeOp = opRecords[2].(operations.RevokeSponsorship)
		tt.Equal(newAccountPair.Address(), *revokeOp.TrustlineAccountID)
		tt.Equal("ABCD:"+sponsorPair.Address(), *revokeOp.TrustlineAsset)

		// Check effects
		effRecords := getEffectsByOp(revokeOp.ID)
		tt.Len(effRecords, 1)
		effect := effRecords[0].(effects.TrustlineSponsorshipUpdated)
		tt.Equal(sponsorPair.Address(), effect.FormerSponsor)
		tt.Equal(newSponsorPair.Address(), effect.NewSponsor)

		// Revoke sponsorship
		ops = []txnbuild.Operation{
			&txnbuild.RevokeSponsorship{
				SponsorshipType: txnbuild.RevokeSponsorshipTypeOffer,
				Offer: &txnbuild.OfferID{
					SellerAccountAddress: offer.Seller,
					OfferID:              offer.ID,
				},
			},
			&txnbuild.RevokeSponsorship{
				SponsorshipType: txnbuild.RevokeSponsorshipTypeTrustLine,
				TrustLine: &txnbuild.TrustLineID{
					Account: newAccountPair.Address(),
					Asset:   asset,
				},
			},
		}
		txResp = itest.MustSubmitOperations(newSponsor, newSponsorPair, ops...)

		// There are intentionally no effects when revoking an Offer
		effRecords = getEffectsByTx(txResp.ID)
		tt.Len(effRecords, 1)
		sponsorshipRemoved := effRecords[0].(effects.TrustlineSponsorshipRemoved)
		tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
		tt.Equal(canonicalAsset, sponsorshipRemoved.Asset)
	})
}

/*
func TestSponsoredClaimableBalance(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	sponsorPair := itest.Master()
	sponsor := itest.MasterAccount

	pairs, _ := itest.CreateAccounts(1, "1000")
	newAccountPair := pairs[0]
	newAccount := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newAccountPair.Address()}
		account, err := client.AccountDetail(request)
		tt.NoError(err)
		return &account
	}
	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: sponsor(),
			SponsoredID:   newAccountPair.Address(),
		},
		&txnbuild.CreateClaimableBalance{
			SourceAccount: newAccount(),
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(sponsorPair.Address(), nil),
			},
			Amount: "20",
			Asset:  txnbuild.NativeAsset{},
		},
		&txnbuild.EndSponsoringFutureReserves{},
	}

	signers := []*keypair.Full{newAccountPair, sponsorPair}
	txResp, err := itest.SubmitMultiSigOperations(newAccount(), signers, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	balances, err := client.ClaimableBalances(sdk.ClaimableBalanceRequest{})
	tt.NoError(err)

	claims := balances.Embedded.Records
	tt.Len(claims, 1)
	balance := claims[0]
	tt.Equal(sponsorPair.Address(), balance.Sponsor)

	// Check effects and details of the CreateClaimableBalance operation
	operationsResponse, err := client.Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	tt.NoError(err)
	tt.Len(operationsResponse.Embedded.Records, 3)
	createClaimableBalance := operationsResponse.Embedded.Records[1].(operations.CreateClaimableBalance)
	tt.Equal(sponsorPair.Address(), createClaimableBalance.Sponsor)

	effectsResponse, err := client.Effects(sdk.EffectRequest{
		ForOperation: createClaimableBalance.GetID(),
	})
	tt.NoError(err)
	if tt.Len(effectsResponse.Embedded.Records, 4) {
		cbSponsorshipEffect := effectsResponse.Embedded.Records[3].(effects.ClaimableBalanceSponsorshipCreated)
		tt.Equal(sponsorPair.Address(), cbSponsorshipEffect.Sponsor)
		tt.Equal(newAccountPair.Address(), cbSponsorshipEffect.Account)
		tt.Equal(balance.BalanceID, cbSponsorshipEffect.BalanceID)
	}

	// Update sponsor

	pairs, _ = itest.CreateAccounts(1, "1000")
	newSponsorPair := pairs[0]
	newSponsor := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newSponsorPair.Address()}
		account, e := client.AccountDetail(request)
		tt.NoError(e)
		return &account
	}
	ops = []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: newSponsor(),
			SponsoredID:   sponsorPair.Address(),
		},
		&txnbuild.RevokeSponsorship{
			SponsorshipType:  txnbuild.RevokeSponsorshipTypeClaimableBalance,
			ClaimableBalance: &balance.BalanceID,
		},
		&txnbuild.EndSponsoringFutureReserves{},
	}
	signers = []*keypair.Full{sponsorPair, newSponsorPair}
	txResp, err = itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
	tt.NoError(err)

	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify operation details
	response, err := client.Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords := response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 3)
	tt.True(opRecords[1].IsTransactionSuccessful())

	revokeOp := opRecords[1].(operations.RevokeSponsorship)
	tt.Equal(balance.BalanceID, *revokeOp.ClaimableBalanceID)

	// Check effects
	effectsResponse, err = client.Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords := effectsResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.ClaimableBalanceSponsorshipUpdated{}, effectRecords[0])
	effect := effectRecords[0].(effects.ClaimableBalanceSponsorshipUpdated)
	tt.Equal(sponsorPair.Address(), effect.FormerSponsor)
	tt.Equal(newSponsorPair.Address(), effect.NewSponsor)

	// It's not possible to explicitly revoke the sponsorship of a claimable balance,
	// so we claim the balance instead
	claimOp := &txnbuild.ClaimClaimableBalance{
		BalanceID: balance.BalanceID,
	}
	txResp = itest.MustSubmitOperations(sponsor(), sponsorPair, claimOp)
	effectsResponse, err = client.Effects(sdk.EffectRequest{
		ForTransaction: txResp.ID,
	})
	tt.NoError(err)
	effectRecords = effectsResponse.Embedded.Records
	tt.Len(effectRecords, 3)
	sponsorshipRemoved := effectRecords[2].(effects.ClaimableBalanceSponsorshipRemoved)
	tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
	tt.Equal(balance.BalanceID, sponsorshipRemoved.BalanceID)
}
*/
