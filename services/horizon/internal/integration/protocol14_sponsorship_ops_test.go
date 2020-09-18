package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stellar/go/xdr"
)

func getSimpleAccountCreationSandwich(a *assert.Assertions) (*keypair.Full, []txnbuild.Operation) {
	// We will create the following operation structure:
	// BeginSponsoringFutureReserves A
	//   CreateAccount A
	// EndSponsoringFutureReserves (with A as a source)

	ops := make([]txnbuild.Operation, 3, 3)
	newAccountPair, err := keypair.Random()
	a.NoError(err)

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
	sponsor := itest.Master()
	newAccountPair, ops := getSimpleAccountCreationSandwich(tt)

	signers := []*keypair.Full{sponsor, newAccountPair}
	txResp, err := itest.SubmitMultiSigOperations(itest.MasterAccount(), signers, ops...)
	assert.NoError(t, err)

	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(txResp.ResultXdr, &txResult)
	assert.NoError(t, err)
	assert.Equal(t, xdr.TransactionResultCodeTxSuccess, txResult.Result.Code)

	response, err := itest.Client().Operations(sdk.OperationRequest{
		Order: "asc",
	})
	records := response.Embedded.Records
	tt.NoError(err)
	tt.Len(records, 3)
	tt.True(records[0].IsTransactionSuccessful())

	// Verify operation details
	tt.Equal(ops[0].(*txnbuild.BeginSponsoringFutureReserves).SponsoredID,
		records[0].(operations.BeginSponsoringFutureReserves).SponsoredID)

	actualCreateAccount := records[1].(operations.CreateAccount)
	tt.Equal(sponsor.Address(), actualCreateAccount.Sponsor)

	endSponsoringOp := records[2].(operations.EndSponsoringFutureReserves)
	tt.Equal(sponsor.Address(), endSponsoringOp.BeginSponsor)

	// Make sure that the sponsor is an (implicit) participant on the end sponsorship operation

	response, err = itest.Client().Operations(sdk.OperationRequest{
		ForAccount: sponsor.Address(),
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
		AccountID: sponsor.Address(),
	})
	tt.NoError(err)
	account.NumSponsoring = 1

	account, err = itest.Client().AccountDetail(sdk.AccountRequest{
		AccountID: newAccountPair.Address(),
	})
	tt.NoError(err)
	account.NumSponsored = 1

}
