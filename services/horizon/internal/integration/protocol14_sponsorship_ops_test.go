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
	sponsor := itest.MasterAccount
	sponsorPair := itest.Master()

	// We will create the following operation structure:
	// BeginSponsoringFutureReserves A
	//   CreateAccount A
	// EndSponsoringFutureReserves (with A as a source)

	newAccountPair, err := keypair.Random()
	tt.NoError(err)

	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SponsoredID: newAccountPair.Address(),
		},

		&txnbuild.CreateAccount{
			Destination: newAccountPair.Address(),
			Amount:      "1000",
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: newAccountPair.Address(),
			},
		},
	}

	signers := []*keypair.Full{sponsorPair, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
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
	effectsResponse, err := itest.Client().Effects(sdk.EffectRequest{ForOperation: opRecords[1].GetID()})
	tt.NoError(err)
	effectRecords := effectsResponse.Embedded.Records
	tt.Len(effectRecords, 4)
	tt.IsType(effects.AccountSponsorshipCreated{}, effectRecords[3])
	tt.Equal(sponsorPair.Address(), effectRecords[3].(effects.AccountSponsorshipCreated).Sponsor)

	// Update sponsor

	accountToRevoke := newAccountPair.Address()
	pairs, _ := itest.CreateAccounts(1, "1000")
	newSponsorPair := pairs[0]
	newSponsor := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newSponsorPair.Address()}
		account, e := itest.Client().AccountDetail(request)
		tt.NoError(e)
		return &account
	}
	ops = []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: newSponsor(),
			SponsoredID:   sponsorPair.Address(),
		},
		&txnbuild.RevokeSponsorship{
			SourceAccount:   sponsor(),
			SponsorshipType: txnbuild.RevokeSponsorshipTypeAccount,
			Account:         &accountToRevoke,
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
	response, err = itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords = response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 3)
	tt.True(opRecords[1].IsTransactionSuccessful())

	revokeOp := opRecords[1].(operations.RevokeSponsorship)
	tt.Equal(accountToRevoke, *revokeOp.AccountID)

	// Check effects
	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords = effectsResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.AccountSponsorshipUpdated{}, effectRecords[0])
	effect := effectRecords[0].(effects.AccountSponsorshipUpdated)
	tt.Equal(sponsorPair.Address(), effect.FormerSponsor)
	tt.Equal(newSponsorPair.Address(), effect.NewSponsor)

	// Revoke sponsorship

	op := &txnbuild.RevokeSponsorship{
		SponsorshipType: txnbuild.RevokeSponsorshipTypeAccount,
		Account:         &accountToRevoke,
	}
	txResp = itest.MustSubmitOperations(newSponsor(), newSponsorPair, op)

	// Verify operation details
	response, err = itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords = response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 1)
	tt.True(opRecords[0].IsTransactionSuccessful())

	revokeOp = opRecords[0].(operations.RevokeSponsorship)
	tt.Equal(accountToRevoke, *revokeOp.AccountID)

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
	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords = effectsResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.AccountSponsorshipRemoved{}, effectRecords[0])
	tt.Equal(newSponsorPair.Address(), effectRecords[0].(effects.AccountSponsorshipRemoved).FormerSponsor)
}

func TestSponsoredSigner(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsorPair := itest.Master()
	sponsor := itest.MasterAccount

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
	//   SetOptionsSigner (Source=N)
	// EndSponsorship (Source=N)

	// unspecific signer
	signerKey := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"

	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SponsoredID: newAccountPair.Address(),
		},
		&txnbuild.SetOptions{
			SourceAccount: newAccount(),
			Signer: &txnbuild.Signer{
				Address: signerKey,
				Weight:  1,
			},
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: newAccount(),
		},
	}

	signers := []*keypair.Full{sponsorPair, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify that the signer was incorporated
	signerAdded := func() bool {
		signers := newAccount().(*protocol.Account).Signers
		for _, signer := range signers {
			if signer.Key == signerKey {
				tt.Equal(sponsorPair.Address(), signer.Sponsor)
				return true
			}
		}
		return false
	}
	tt.Eventually(signerAdded, time.Second*10, time.Millisecond*100)

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
		tt.Equal(newAccountPair.Address(), signerSponsorshipEffect.Account)
		tt.Equal(signerKey, signerSponsorshipEffect.Signer)
	}

	// Update sponsor

	pairs, _ = itest.CreateAccounts(1, "1000")
	newSponsorPair := pairs[0]
	newSponsor := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newSponsorPair.Address()}
		account, e := itest.Client().AccountDetail(request)
		tt.NoError(e)
		return &account
	}
	ops = []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: newSponsor(),
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
	txResp, err = itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
	tt.NoError(err)

	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify operation details
	response, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords := response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 3)
	tt.True(opRecords[1].IsTransactionSuccessful())

	revokeOp := opRecords[1].(operations.RevokeSponsorship)
	tt.Equal(newAccountPair.Address(), *revokeOp.SignerAccountID)
	tt.Equal(signerKey, *revokeOp.SignerKey)

	// Check effects
	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords := effectsResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.SignerSponsorshipUpdated{}, effectRecords[0])
	effect := effectRecords[0].(effects.SignerSponsorshipUpdated)
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
	txResp = itest.MustSubmitOperations(newSponsor(), newSponsorPair, &revoke)

	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{
		ForTransaction: txResp.ID,
	})
	tt.NoError(err)
	tt.Len(effectsResponse.Embedded.Records, 1)
	sponsorshipRemoved := effectsResponse.Embedded.Records[0].(effects.SignerSponsorshipRemoved)
	tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
	tt.Equal(signerKey, sponsorshipRemoved.Signer)

}

func TestSponsoredPreAuthSigner(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsorPair := itest.Master()
	sponsor := itest.MasterAccount

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
	preAuthSignerKey := xdr.SignerKey{
		Type:      xdr.SignerKeyTypeSignerKeyTypePreAuthTx,
		PreAuthTx: (*xdr.Uint256)(&preAuthHash),
	}
	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SponsoredID: newAccountPair.Address(),
		},

		&txnbuild.SetOptions{
			SourceAccount: newAccount(),
			Signer: &txnbuild.Signer{
				Address: preAuthSignerKey.Address(),
				Weight:  1,
			},
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: newAccount(),
		},
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
	sponsor := itest.MasterAccount

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
	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SponsoredID: newAccountPair.Address(),
		},
		&txnbuild.ManageData{
			Name:          "SponsoredData",
			Value:         []byte("SponsoredValue"),
			SourceAccount: newAccount(),
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: newAccount(),
		},
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

	// Update sponsor

	pairs, _ = itest.CreateAccounts(1, "1000")
	newSponsorPair := pairs[0]
	newSponsor := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newSponsorPair.Address()}
		account, e := itest.Client().AccountDetail(request)
		tt.NoError(e)
		return &account
	}
	ops = []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: newSponsor(),
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
	txResp, err = itest.SubmitMultiSigOperations(sponsor(), signers, ops...)
	tt.NoError(err)

	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	// Verify operation details
	response, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords := response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 3)
	tt.True(opRecords[1].IsTransactionSuccessful())

	revokeOp := opRecords[1].(operations.RevokeSponsorship)
	tt.Equal(newAccountPair.Address(), *revokeOp.DataAccountID)
	tt.Equal("SponsoredData", *revokeOp.DataName)

	// Check effects
	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords := effectsResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.DataSponsorshipUpdated{}, effectRecords[0])
	effect := effectRecords[0].(effects.DataSponsorshipUpdated)
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
	txResp = itest.MustSubmitOperations(newSponsor(), newSponsorPair, &revoke)

	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{
		ForTransaction: txResp.ID,
	})
	tt.NoError(err)
	tt.Len(effectsResponse.Embedded.Records, 1)
	sponsorshipRemoved := effectsResponse.Embedded.Records[0].(effects.DataSponsorshipRemoved)
	tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
	tt.Equal("SponsoredData", sponsorshipRemoved.DataName)

}

func TestSponsoredTrustlineAndOffer(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsorPair := itest.Master()
	sponsor := itest.MasterAccount

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
	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SponsoredID: newAccountPair.Address(),
		},
		&txnbuild.ChangeTrust{
			SourceAccount: newAccount(),
			Line:          txnbuild.CreditAsset{"ABCD", sponsorPair.Address()},
			Limit:         txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.ManageSellOffer{
			SourceAccount: newAccount(),
			Selling:       txnbuild.NativeAsset{},
			Buying:        txnbuild.CreditAsset{"ABCD", sponsorPair.Address()},
			Amount:        "3",
			Price:         "1",
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: newAccount(),
		},
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

	// Update sponsor

	pairs, _ = itest.CreateAccounts(1, "1000")
	newSponsorPair := pairs[0]
	newSponsor := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newSponsorPair.Address()}
		account, e := itest.Client().AccountDetail(request)
		tt.NoError(e)
		return &account
	}
	ops = []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: newSponsor(),
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
				Asset: txnbuild.CreditAsset{
					Code:   "ABCD",
					Issuer: sponsorPair.Address(),
				},
			},
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
	response, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords := response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 4)

	tt.True(opRecords[1].IsTransactionSuccessful())
	revokeOp := opRecords[1].(operations.RevokeSponsorship)
	tt.Equal(offer.ID, *revokeOp.OfferID)

	tt.True(opRecords[2].IsTransactionSuccessful())
	revokeOp = opRecords[2].(operations.RevokeSponsorship)
	tt.Equal(newAccountPair.Address(), *revokeOp.TrustlineAccountID)
	tt.Equal("ABCD:"+sponsorPair.Address(), *revokeOp.TrustlineAsset)

	// Check effects
	effectsResponse, err := itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords := effectsResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.TrustlineSponsorshipUpdated{}, effectRecords[0])
	effect := effectRecords[0].(effects.TrustlineSponsorshipUpdated)
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
				Asset: txnbuild.CreditAsset{
					Code:   "ABCD",
					Issuer: sponsorPair.Address(),
				},
			},
		},
	}
	txResp = itest.MustSubmitOperations(newSponsor(), newSponsorPair, ops...)

	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{
		ForTransaction: txResp.ID,
	})
	tt.NoError(err)
	// There are intentionally no effects when revoking an Offer
	tt.Len(effectsResponse.Embedded.Records, 1)
	sponsorshipRemoved := effectsResponse.Embedded.Records[0].(effects.TrustlineSponsorshipRemoved)
	tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
	tt.Equal("ABCD:"+sponsorPair.Address(), sponsorshipRemoved.Asset)
}

func TestSponsoredClaimableBalance(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsorPair := itest.Master()
	sponsor := itest.MasterAccount

	pairs, _ := itest.CreateAccounts(1, "1000")
	newAccountPair := pairs[0]
	newAccount := func() txnbuild.Account {
		request := sdk.AccountRequest{AccountID: newAccountPair.Address()}
		account, err := itest.Client().AccountDetail(request)
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

	balances, err := itest.Client().ClaimableBalances(sdk.ClaimableBalanceRequest{})
	tt.NoError(err)

	claims := balances.Embedded.Records
	tt.Len(claims, 1)
	balance := claims[0]
	tt.Equal(sponsorPair.Address(), balance.Sponsor)

	// Check effects and details of the CreateClaimableBalance operation
	operationsResponse, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	tt.NoError(err)
	tt.Len(operationsResponse.Embedded.Records, 3)
	createClaimableBalance := operationsResponse.Embedded.Records[1].(operations.CreateClaimableBalance)
	tt.Equal(sponsorPair.Address(), createClaimableBalance.Sponsor)

	effectsResponse, err := itest.Client().Effects(sdk.EffectRequest{
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
		account, e := itest.Client().AccountDetail(request)
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
	response, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords := response.Embedded.Records
	tt.NoError(err)
	tt.Len(opRecords, 3)
	tt.True(opRecords[1].IsTransactionSuccessful())

	revokeOp := opRecords[1].(operations.RevokeSponsorship)
	tt.Equal(balance.BalanceID, *revokeOp.ClaimableBalanceID)

	// Check effects
	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
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
	effectsResponse, err = itest.Client().Effects(sdk.EffectRequest{
		ForTransaction: txResp.ID,
	})
	tt.NoError(err)
	effectRecords = effectsResponse.Embedded.Records
	tt.Len(effectRecords, 3)
	sponsorshipRemoved := effectRecords[2].(effects.ClaimableBalanceSponsorshipRemoved)
	tt.Equal(newSponsorPair.Address(), sponsorshipRemoved.FormerSponsor)
	tt.Equal(balance.BalanceID, sponsorshipRemoved.BalanceID)
}
