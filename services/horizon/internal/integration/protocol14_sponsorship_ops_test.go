package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/xdr"
)

func getSimpleAccountCreationSandwich(tt *assert.Assertions) (*keypair.Full, []txnbuild.Operation) {
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
	return newAccountPair, ops
}

func TestSimpleSandwichHappyPath(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsor := itest.MasterAccount()
	sponsorPair := itest.Master()
	newAccountPair, ops := getSimpleAccountCreationSandwich(tt)

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
}

func TestSimpleSandwichRevocation(t *testing.T) {
	tt := assert.New(t)
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	sponsor := itest.MasterAccount()
	sponsorPair := itest.Master()
	newAccountPair, ops := getSimpleAccountCreationSandwich(tt)

	signers := []*keypair.Full{sponsorPair, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(sponsor, signers, ops...)
	tt.NoError(err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	tt.NoError(err)
	tt.Equal(xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

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
	response, err := itest.Client().Operations(sdk.OperationRequest{
		ForTransaction: txResp.Hash,
	})
	opRecords := response.Embedded.Records
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
	eResponse, err := itest.Client().Effects(sdk.EffectRequest{ForOperation: revokeOp.ID})
	tt.NoError(err)
	effectRecords := eResponse.Embedded.Records
	tt.Len(effectRecords, 1)
	tt.IsType(effects.AccountSponsorshipRemoved{}, effectRecords[0])
	tt.Equal(sponsorPair.Address(), effectRecords[0].(effects.AccountSponsorshipRemoved).FormerSponsor)
}
