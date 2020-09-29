package integration

import (
	"encoding/base64"
	"testing"
	"time"

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

func TestSponsoredAccount(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsor := itest.MasterAccount()
	sponsorPair := itest.Master()

	// We will create the following operation structure:
	// BeginSponsoringFutureReserves A
	//   CreateAccount A
	// EndSponsoringFutureReserves (with A as a source)

	ops := make([]txnbuild.Operation, 3, 3)
	newAccountPair, err := keypair.Random()
	tt.NoError(err)

	ops[0] = &txnbuild.BeginSponsoringFutureReserves{
		SponsoredID: newAccountPair.Address(),
	}
	ops[1] = &txnbuild.CreateAccount{
		Destination: newAccountPair.Address(),
		Amount:      "1000",
	}
	ops[2] = &txnbuild.EndSponsoringFutureReserves{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: newAccountPair.Address(),
		},
	}

	signers := []*keypair.Full{sponsorPair, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(sponsor, signers, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	response, err := itest.Client().Operations(sdk.OperationRequest{
		Order: "asc",
	})
	opRecords := response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 3)
	tt.True(opRecords[0].IsTransactionSuccessful())

	// Verify operation details
	tt.Equal(ops[0].(*txnbuild.BeginSponsoringFutureReserves).SponsoredID,
		opRecords[0].(operations.BeginSponsoringFutureReserves).SponsoredID)

	actualCreateAccount := opRecords[1].(operations.CreateAccount)
	tt.Equal(sponsorPair.Address(), actualCreateAccount.Sponsor)

	endSponsoringOp := opRecords[2].(operations.EndSponsoringFutureReserves)
	tt.Equal(sponsorPair.Address(), endSponsoringOp.BeginSponsor)

	// Make sure that the sponsor is an (implicit) participant on the end sponsorship operation

	response, err = itest.Client().Operations(sdk.OperationRequest{
		ForAccount: sponsorPair.Address(),
	})
	tt.NoError(err)

	endSponsorshipPresent := func() bool {
		for _, o := range response.Embedded.Records {
			if o.GetID() == endSponsoringOp.ID {
				return true
			}
		}
		return false
	}
	tt.Condition(endSponsorshipPresent)

	// Check numSponsoring and numSponsored
	account, err := itest.Client().AccountDetail(sdk.AccountRequest{
		AccountID: sponsorPair.Address(),
	})
	tt.NoError(err)
	account.NumSponsoring = 1

	account, err = itest.Client().AccountDetail(sdk.AccountRequest{
		AccountID: newAccountPair.Address(),
	})
	tt.NoError(err)
	account.NumSponsored = 1

	// Check effects of CreateAccount Operation
	eResponse, err := itest.Client().Effects(sdk.EffectRequest{ForOperation: opRecords[1].GetID()})
	tt.NoError(err)
	effectRecords := eResponse.Embedded.Records
	tt.Len(effectRecords, 4)
	tt.IsType(effects.AccountSponsorshipCreated{}, effectRecords[3])
	tt.Equal(sponsorPair.Address(), effectRecords[3].(effects.AccountSponsorshipCreated).Sponsor)

	// Submit sponsorship revocation in a separate transaction
	accountToRevoke := newAccountPair.Address()
	op := &txnbuild.RevokeSponsorship{
		SponsorshipType: txnbuild.RevokeSponsorshipTypeAccount,
		Account:         &accountToRevoke,
	}
	txResp, err = itest.SubmitOperations(sponsor, sponsorPair, op)
	tt.NoError(err)

	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify operation details
	response, err = itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords = response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 1)
	tt.True(opRecords[0].IsTransactionSuccessful())

	revokeOp := opRecords[0].(operations.RevokeSponsorship)
	tt.Equal(*op.Account, *revokeOp.AccountID)

	// Make sure that the sponsoree is an (implicit) participant in the revocation operation
	response, err = itest.Client().Operations(sdk.OperationRequest{
		ForAccount: newAccountPair.Address(),
	})
	tt.NoError(err)

	sponsorshipRevocationPresent := func() bool {
		for _, o := range response.Embedded.Records {
			if o.GetID() == revokeOp.ID {
				return true
			}
		}
		return false
	}
	tt.Condition(sponsorshipRevocationPresent)

	// Check effects
	eResponse, err = itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords = eResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.AccountSponsorshipRemoved{}, effectRecords[0])
	tt.Equal(sponsorPair.Address(), effectRecords[0].(effects.AccountSponsorshipRemoved).FormerSponsor)
}

func TestSponsorPreAuthSigner(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsorPair := itest.Master()
	sponsor := func() txnbuild.Account { return itest.MasterAccount() }

	// Let's create a new account
	pairs, _ := itest.CreateAccounts(1, "1000")
	newAccountPair := pairs[0]
	newAccount := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newAccountPair.Address()}
		account, err := itest.Client().AccountDetail(request)
		tt.NoError(err)
		return &account
	}

	// Let's create a preauthorized transaction for the new account
	// to add a signer
	preAuthOp := &txnbuild.SetOptions{
		Signer: &txnbuild.Signer{
			// unspecific signer
			Address: "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			Weight:  1,
		},
	}
	txParams := txnbuild.TransactionParams{
		SourceAccount:        newAccount(),
		Operations:           []txnbuild.Operation{preAuthOp},
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewInfiniteTimeout(),
		IncrementSequenceNum: true,
	}
	preaAuthTx, err := txnbuild.NewTransaction(txParams)
	tt.NoError(err)
	preAuthHash, err := preaAuthTx.Hash(test.IntegrationNetworkPassphrase)
	tt.NoError(err)

	// Let's add a sponsored preauth signer with the following transaction:
	//
	// BeginSponsorship N (Source=sponsor)
	//   SetOptionsSigner preAuthHash (Source=N)
	// EndSponsorship (Source=N)
	ops := make([]txnbuild.Operation, 3, 3)
	ops[0] = &txnbuild.BeginSponsoringFutureReserves{
		SponsoredID: newAccountPair.Address(),
	}
	preAuthSignerKey := xdr.SignerKey{
		Type:      xdr.SignerKeyTypeSignerKeyTypePreAuthTx,
		PreAuthTx: (*xdr.Uint256)(&preAuthHash),
	}
	ops[1] = &txnbuild.SetOptions{
		SourceAccount: newAccount(),
		Signer: &txnbuild.Signer{
			Address: preAuthSignerKey.Address(),
			Weight:  1,
		},
	}
	ops[2] = &txnbuild.EndSponsoringFutureReserves{
		SourceAccount: newAccount(),
	}

	signers := []*keypair.Full{sponsorPair, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify that the preauth signer was incorporated
	preAuthSignerAdded := func() bool {
		for _, signer := range newAccount().(*protocol.Account).Signers {
			if preAuthSignerKey.Address() == signer.Key {
				return true
			}
		}
		return false
	}
	tt.Eventually(preAuthSignerAdded, time.Second*10, time.Millisecond*100)

	// Check effects and details of the SetOptions operation
	operationsResponse, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	tt.NoError(err)
	tt.Len(operationsResponse.Embedded.Records, 3)
	setOptionsOp := operationsResponse.Embedded.Records[1].(operations.SetOptions)
	tt.Equal(sponsorPair.Address(), setOptionsOp.Sponsor)

	effectsResponse, err := itest.Client().Effects(sdk.EffectRequest{
		ForOperation: setOptionsOp.GetID(),
	})
	tt.NoError(err)
	if tt.Len(effectsResponse.Embedded.Records, 2) {
		signerSponsorshipEffect := effectsResponse.Embedded.Records[1].(effects.SignerSponsorshipCreated)
		tt.Equal(sponsorPair.Address(), signerSponsorshipEffect.Sponsor)
		tt.Equal(preAuthSignerKey.Address(), signerSponsorshipEffect.Signer)
	}

	// Submit the preauthorized transaction
	preAuthTxB64, err := preaAuthTx.Base64()
	tt.NoError(err)
	txResp, err = itest.Client().SubmitTransactionXDR(preAuthTxB64)
	tt.NoError(err)
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify that the new signer was incorporated and that the preauth signer was removed
	preAuthSignerAdded = func() bool {
		signers := newAccount().(*protocol.Account).Signers
		if len(signers) != 2 {
			return false
		}
		for _, signer := range signers {
			if "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML" == signer.Key {
				return true
			}
		}
		return false
	}
	tt.Eventually(preAuthSignerAdded, time.Second*10, time.Millisecond*100)

	// Check effects
	// Disabled since the effects processor doesn't process transaction-level changes
	// See https://github.com/stellar/go/pull/3050#discussion_r493651644
	/*
			operationsResponse, err = itest.Client().Operations(sdk.OperationRequest{
				ForTransaction: txResp.Hash,
			})
			tt.Len(operationsResponse.Embedded.Records, 1)
			setOptionsOp = operationsResponse.Embedded.Records[0].(operations.SetOptions)

		    effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{
				ForTransaction: txResp.Hash,
		    })
			tt.NoError(err)
			if tt.Len(effectsResponse.Embedded.Records, 2) {
				signerSponsorshipEffect := effectsResponse.Embedded.Records[1].(effects.SignerSponsorshipRemoved)
				tt.Equal(sponsorPair.Address(), signerSponsorshipEffect.FormerSponsor)
				tt.Equal("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", signerSponsorshipEffect.Signer)
			}
	*/

}

func TestSponsoredData(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsorPair := itest.Master()
	sponsor := func() txnbuild.Account { return itest.MasterAccount() }

	// Let's create a new account
	pairs, _ := itest.CreateAccounts(1, "1000")
	newAccountPair := pairs[0]
	newAccount := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newAccountPair.Address()}
		account, err := itest.Client().AccountDetail(request)
		tt.NoError(err)
		return &account
	}

	// Let's add a sponsored data entry
	//
	// BeginSponsorship N (Source=sponsor)
	//   ManageData "SponsoredData"="SponsoredValue" (Source=N)
	// EndSponsorship (Source=N)
	ops := make([]txnbuild.Operation, 3, 3)
	ops[0] = &txnbuild.BeginSponsoringFutureReserves{
		SponsoredID: newAccountPair.Address(),
	}
	ops[1] = &txnbuild.ManageData{
		Name:          "SponsoredData",
		Value:         []byte("SponsoredValue"),
		SourceAccount: newAccount(),
	}
	ops[2] = &txnbuild.EndSponsoringFutureReserves{
		SourceAccount: newAccount(),
	}

	signers := []*keypair.Full{sponsorPair, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify that the data was incorporated
	dataAdded := func() bool {
		data := newAccount().(*protocol.Account).Data
		if value, ok := data["SponsoredData"]; ok {
			decoded, e := base64.StdEncoding.DecodeString(value)
			tt.NoError(e)
			if string(decoded) == "SponsoredValue" {
				return true
			}
		}
		return false
	}
	tt.Eventually(dataAdded, time.Second*10, time.Millisecond*100)

	// Check effects and details of the ManageData operation
	operationsResponse, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	tt.NoError(err)
	tt.Len(operationsResponse.Embedded.Records, 3)
	manageDataOp := operationsResponse.Embedded.Records[1].(operations.ManageData)
	tt.Equal(sponsorPair.Address(), manageDataOp.Sponsor)

	effectsResponse, err := itest.Client().Effects(sdk.EffectRequest{
		ForOperation: manageDataOp.GetID(),
	})
	tt.NoError(err)
	if tt.Len(effectsResponse.Embedded.Records, 2) {
		dataSponsorshipEffect := effectsResponse.Embedded.Records[1].(effects.DataSponsorshipCreated)
		tt.Equal(sponsorPair.Address(), dataSponsorshipEffect.Sponsor)
		tt.Equal(newAccountPair.Address(), dataSponsorshipEffect.Account)
		tt.Equal("SponsoredData", dataSponsorshipEffect.DataName)
	}

	// Test revocation of sponsorship
	revoke := txnbuild.RevokeSponsorship{
		SponsorshipType: txnbuild.RevokeSponsorshipTypeData,
		Data: &txnbuild.DataID{
			Account:  newAccountPair.Address(),
			DataName: "SponsoredData",
		},
	}
	tx := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(), &revoke)

	effectRecords, err := itest.Client().Effects(sdk.EffectRequest{
		ForTransaction: tx.ID,
	})
	tt.NoError(err)
	tt.Len(effectRecords.Embedded.Records, 1)
	sponsorshipRemoved := effectRecords.Embedded.Records[0].(effects.DataSponsorshipRemoved)
	tt.Equal(sponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
	tt.Equal("SponsoredData", sponsorshipRemoved.DataName)

}

func TestSponsoredTrustlineAndOffer(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsorPair := itest.Master()
	sponsor := func() txnbuild.Account { return itest.MasterAccount() }

	// Let's create a new account
	pairs, _ := itest.CreateAccounts(1, "1000")
	newAccountPair := pairs[0]
	newAccount := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newAccountPair.Address()}
		account, err := itest.Client().AccountDetail(request)
		tt.NoError(err)
		return &account
	}

	// Let's add a sponsored trustline and offer
	//
	// BeginSponsorship N (Source=sponsor)
	//   Change Trust (ABC, sponsor) (Source=N)
	//   ManageSellOffer Buying (ABC, sponsor) (Source=N)
	// EndSponsorship (Source=N)
	ops := make([]txnbuild.Operation, 4, 4)
	ops[0] = &txnbuild.BeginSponsoringFutureReserves{
		SponsoredID: newAccountPair.Address(),
	}
	ops[1] = &txnbuild.ChangeTrust{
		SourceAccount: newAccount(),
		Line:          txnbuild.CreditAsset{"ABCD", sponsorPair.Address()},
		Limit:         txnbuild.MaxTrustlineLimit,
	}
	ops[2] = &txnbuild.ManageSellOffer{
		SourceAccount: newAccount(),
		Selling:       txnbuild.NativeAsset{},
		Buying:        txnbuild.CreditAsset{"ABCD", sponsorPair.Address()},
		Amount:        "3",
		Price:         "1",
	}
	ops[3] = &txnbuild.EndSponsoringFutureReserves{
		SourceAccount: newAccount(),
	}

	signers := []*keypair.Full{sponsorPair, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify that the offer was incorporated correctly
	trustlineAdded := func() bool {
		for _, balance := range newAccount().(*protocol.Account).Balances {
			if balance.Issuer == sponsorPair.Address() {
				tt.Equal("ABCD", balance.Code)
				tt.Equal(sponsorPair.Address(), balance.Sponsor)
				return true
			}
		}
		return false
	}
	tt.Eventually(trustlineAdded, time.Second*10, time.Millisecond*100)

	// Check the details of the ManageSellOffer operation
	// (there are no effects, which is intentional)
	operationsResponse, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	tt.NoError(err)
	tt.Len(operationsResponse.Embedded.Records, 4)
	changeTrust := operationsResponse.Embedded.Records[1].(operations.ChangeTrust)
	tt.Equal(sponsorPair.Address(), changeTrust.Sponsor)

	// Verify that the offer was incorporated correctly
	var offer protocol.Offer
	offerAdded := func() bool {
		offers, e := itest.Client().Offers(sdk.OfferRequest{
			ForAccount: newAccountPair.Address(),
		})
		tt.NoError(e)
		if len(offers.Embedded.Records) == 1 {
			offer = offers.Embedded.Records[0]
			tt.Equal(sponsorPair.Address(), offer.Buying.Issuer)
			tt.Equal("ABCD", offer.Buying.Code)
			tt.Equal(sponsorPair.Address(), offer.Sponsor)
			return true
		}
		return false
	}
	tt.Eventually(offerAdded, time.Second*10, time.Millisecond*100)

	// Check the details of the ManageSellOffer operation
	// (there are no effects, which is intentional)
	manageOffer := operationsResponse.Embedded.Records[2].(operations.ManageSellOffer)
	tt.Equal(sponsorPair.Address(), manageOffer.Sponsor)

	// Test revocation of sponsorships
	revoke1 := txnbuild.RevokeSponsorship{
		SponsorshipType: txnbuild.RevokeSponsorshipTypeOffer,
		Offer: &txnbuild.OfferID{
			SellerAccountAddress: offer.Seller,
			OfferID:              offer.ID,
		},
	}
	revoke2 := txnbuild.RevokeSponsorship{
		SponsorshipType: txnbuild.RevokeSponsorshipTypeTrustLine,
		TrustLine: &txnbuild.TrustLineID{
			Account: newAccountPair.Address(),
			Asset: txnbuild.CreditAsset{
				Code:   "ABCD",
				Issuer: sponsorPair.Address(),
			},
		},
	}
	tx := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(), &revoke1, &revoke2)

	effectRecords, err := itest.Client().Effects(sdk.EffectRequest{
		ForTransaction: tx.ID,
	})
	tt.NoError(err)
	tt.Len(effectRecords.Embedded.Records, 1)
	sponsorshipRemoved := effectRecords.Embedded.Records[0].(effects.TrustlineSponsorshipRemoved)
	tt.Equal(sponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
	tt.Equal("ABCD:"+sponsorPair.Address(), sponsorshipRemoved.Asset)
}

func TestSponsoredClaimableBalance(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	keys, accounts := itest.CreateAccounts(1, "50")
	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: itest.MasterAccount(),
			SponsoredID:   keys[0].Address(),
		},
		&txnbuild.CreateClaimableBalance{
			SourceAccount: accounts[0],
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(master.Address(), nil),
			},
			Amount: "20",
			Asset:  txnbuild.NativeAsset{},
		},
		&txnbuild.EndSponsoringFutureReserves{},
	}

	txResp, err := itest.SubmitMultiSigOperations(accounts[0], []*keypair.Full{keys[0], master}, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	balances, err := itest.Client().ClaimableBalances(sdk.ClaimableBalanceRequest{})
	tt.NoError(err)

	claims := balances.Embedded.Records
	tt.Len(claims, 1)
	balance := claims[0]
	tt.Equal(master.Address(), balance.Sponsor)

}
