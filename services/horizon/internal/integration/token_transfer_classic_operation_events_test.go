package integration

import (
	"fmt"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/ingest/processors/token_transfer"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
	"time"
)

var (
	revokeTrustline = func(issuer, trustor string, asset txnbuild.Asset) *txnbuild.SetTrustLineFlags {
		return &txnbuild.SetTrustLineFlags{
			SourceAccount: issuer,
			Trustor:       trustor,
			Asset:         asset,
			ClearFlags:    []txnbuild.TrustLineFlag{txnbuild.TrustLineAuthorized},
			//SetFlags:      []txnbuild.TrustLineFlag{0},
		}
	}
	// Give the master account the revocable flag (needed to set the clawback flag)
	setRevocableFlag = txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
		},
	}

	addTrustlineForAssetOp = func(forAccount string, asset txnbuild.Asset) *txnbuild.ChangeTrust {
		return &txnbuild.ChangeTrust{
			Line: txnbuild.ChangeTrustAssetWrapper{
				Asset: asset,
			},
			Limit:         txnbuild.MaxTrustlineLimit,
			SourceAccount: forAccount,
		}

	}

	addTrustlineForLiquidityPoolOp = func(forAccount string, assetA txnbuild.Asset, assetB txnbuild.Asset) *txnbuild.ChangeTrust {
		return &txnbuild.ChangeTrust{
			SourceAccount: forAccount,
			Line: txnbuild.LiquidityPoolShareChangeTrustAsset{
				LiquidityPoolParameters: txnbuild.LiquidityPoolParameters{
					AssetA: assetA,
					AssetB: assetB,
					Fee:    30,
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		}
	}

	paymentOp = func(src string, dest string, asset txnbuild.Asset, amount string) *txnbuild.Payment {
		return &txnbuild.Payment{
			SourceAccount: src,
			Destination:   dest,
			Asset:         asset,
			Amount:        amount,
		}
	}
)

func printProtoEvents(events []*token_transfer.TokenTransferEvent) {
	for _, event := range events {
		jsonBytes, _ := protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}.Marshal(event)
		fmt.Printf("### Event Type : %v\n", event.GetEventType())
		fmt.Println(string(jsonBytes))
		fmt.Println("###")
	}
}

func TestTrustlineRevocationEvents(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()

	//Setup
	keys, accounts := itest.CreateAccounts(2, "1000000000")
	// One simple account that participates in LP-USDC-XLM
	lpParticipantAccountKeys, lpParticipantAccount := keys[0], accounts[0]
	// Eth Issuer account that participates in LP-USDC-ETH
	ethAccountKeys, ethAccount := keys[1], accounts[1]

	itest.MustSubmitOperations(itest.MasterAccount(), master, &setRevocableFlag)
	itest.MustSubmitOperations(ethAccount, ethAccountKeys, &setRevocableFlag)

	usdcAsset := txnbuild.CreditAsset{
		Code:   "USDC",
		Issuer: master.Address(),
	}

	ethAsset := txnbuild.CreditAsset{
		Code:   "ETH",
		Issuer: ethAccount.GetAccountID(),
	}
	xlmAsset := txnbuild.NativeAsset{}

	itest.MustSubmitMultiSigOperations(itest.MasterAccount(),
		[]*keypair.Full{lpParticipantAccountKeys, master, ethAccountKeys},

		addTrustlineForAssetOp(lpParticipantAccount.GetAccountID(), usdcAsset),
		addTrustlineForAssetOp(ethAccount.GetAccountID(), usdcAsset),
		addTrustlineForAssetOp(master.Address(), ethAsset),

		addTrustlineForLiquidityPoolOp(lpParticipantAccount.GetAccountID(), xlmAsset, usdcAsset),
		addTrustlineForLiquidityPoolOp(ethAccount.GetAccountID(), ethAsset, usdcAsset),

		paymentOp(master.Address(), lpParticipantAccount.GetAccountID(), usdcAsset, "1000"),
		paymentOp(master.Address(), ethAccount.GetAccountID(), usdcAsset, "1000"),
	)

	usdcXlmPoolId, _ := xdr.NewPoolId(
		xdr.MustNewNativeAsset(),
		xdr.MustNewCreditAsset(usdcAsset.Code, usdcAsset.Issuer),
		30,
	)

	usdcEthPoolId, _ := xdr.NewPoolId(
		xdr.MustNewCreditAsset(ethAsset.Code, ethAsset.Issuer),
		xdr.MustNewCreditAsset(usdcAsset.Code, usdcAsset.Issuer),
		30,
	)

	usdcXlmPoolIDHexString := xdr.Hash(usdcXlmPoolId).HexString()
	usdcEthPoolIdHexString := xdr.Hash(usdcEthPoolId).HexString()

	itest.MustSubmitMultiSigOperations(itest.MasterAccount(),
		[]*keypair.Full{master, lpParticipantAccountKeys, ethAccountKeys},
		&txnbuild.LiquidityPoolDeposit{
			SourceAccount:   lpParticipantAccount.GetAccountID(),
			LiquidityPoolID: [32]byte(usdcXlmPoolId),
			MaxAmountA:      "400",
			MaxAmountB:      "777",
			MinPrice:        xdr.Price{N: 1, D: 2},
			MaxPrice:        xdr.Price{N: 2, D: 1},
		},
		&txnbuild.LiquidityPoolDeposit{
			SourceAccount:   ethAccount.GetAccountID(),
			LiquidityPoolID: [32]byte(usdcEthPoolId),
			MaxAmountA:      "400",
			MaxAmountB:      "777",
			MinPrice:        xdr.Price{N: 1, D: 2},
			MaxPrice:        xdr.Price{N: 2, D: 1},
		},
	)

	// Some sanity asserts
	_, err := itest.Client().LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: usdcXlmPoolIDHexString,
	})
	tt.NoError(err)
	_, err = itest.Client().LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: usdcEthPoolIdHexString,
	})
	tt.NoError(err)

	// Actual test fixture
	revokeTrustlineTxResp := itest.MustSubmitMultiSigOperations(
		itest.MasterAccount(),
		[]*keypair.Full{master},
		// This operation shud generate 2 transfer events
		revokeTrustline(master.Address(), lpParticipantAccount.GetAccountID(), usdcAsset),
		// Revoking USDC trustline for EthIssuer.. BURN
		revokeTrustline(master.Address(), ethAccount.GetAccountID(), usdcAsset),
	)

	ledgerSeq := uint32(revokeTrustlineTxResp.Ledger)
	itest.WaitForLedgerInArchive(30*time.Second, ledgerSeq)
	ledger := getLedgers(itest, ledgerSeq, ledgerSeq)[ledgerSeq]

	events, _ := token_transfer.ProcessTokenTransferEventsFromLedger(ledger, itest.GetPassPhrase())
	fmt.Println("Printing all token transfer events from ledger:")

	// Assertions to follow soon, for now just printing
	printProtoEvents(events)

}
