package integration

import (
	"crypto/sha256"
	"fmt"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/ingest"
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

func generateCBIdFromLpId(lpId xdr.PoolId, accountSeq int64, txAccount xdr.AccountId, opIndex uint32, asset xdr.Asset) xdr.Hash {
	preImageId := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypePoolRevokeOpId,
		RevokeId: &xdr.HashIdPreimageRevokeId{
			SourceAccount:   txAccount,
			SeqNum:          xdr.SequenceNumber(accountSeq),
			OpNum:           xdr.Uint32(opIndex),
			LiquidityPoolId: lpId,
			Asset:           asset,
		},
	}
	binaryDump, _ := preImageId.MarshalBinary()
	sha256hash := xdr.Hash(sha256.Sum256(binaryDump))
	return sha256hash
}
func printProtoEvents(events []*token_transfer.TokenTransferEvent) {
	for _, event := range events {
		jsonBytes, _ := protojson.MarshalOptions{
			Multiline: true, // Enable pretty printing with newlines
			Indent:    "  ", // Specify indentation string (e.g., two spaces)
		}.Marshal(event)
		fmt.Println("###")
		fmt.Println(string(jsonBytes))
		fmt.Println("###")
	}
}

func getLpIdsFromChanges(changes []ingest.Change) []xdr.PoolId {

	var entries []xdr.PoolId
	for _, c := range changes {
		if c.Type != xdr.LedgerEntryTypeLiquidityPool {
			continue
		}
		var lpId xdr.PoolId

		if c.Pre != nil {
			lpId = c.Pre.Data.LiquidityPool.LiquidityPoolId
		}

		if c.Post != nil {
			lpId = c.Post.Data.LiquidityPool.LiquidityPoolId
		}

		entries = append(entries, lpId)
	}

	return entries
}

func getCbEntriesFromChanges(changes []ingest.Change) []xdr.ClaimableBalanceEntry {

	var entries []xdr.ClaimableBalanceEntry
	/*
		This function is expected to be called only to get details of newly created claimable balance
		(for e.g as a result of allowTrust or setTrustlineFlags  operations)
		or claimable balances that are be deleted
		(for e.g due to clawback claimable balance operation)
	*/
	var cb xdr.ClaimableBalanceEntry
	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeClaimableBalance {
			continue
		}
		// Check if claimable balance entry is deleted
		if change.Pre != nil && change.Post == nil {
			cb = change.Pre.Data.MustClaimableBalance()
			entries = append(entries, cb)
		} else if change.Post != nil && change.Pre == nil { // check if claimable balance entry is created
			cb = change.Post.Data.MustClaimableBalance()
			entries = append(entries, cb)
		}
	}

	return entries
}

func TestTrustlineRevocationEvents(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()

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

	// Pre-setup
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

	_, err := itest.Client().LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: usdcXlmPoolIDHexString,
	})
	tt.NoError(err)
	_, err = itest.Client().LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: usdcEthPoolIdHexString,
	})
	tt.NoError(err)

	revokeTrustlineTxResp := itest.MustSubmitMultiSigOperations(
		itest.MasterAccount(),
		[]*keypair.Full{master, ethAccountKeys},
		revokeTrustline(master.Address(), lpParticipantAccount.GetAccountID(), usdcAsset),
		revokeTrustline(master.Address(), ethAccount.GetAccountID(), usdcAsset),
		revokeTrustline(ethAccount.GetAccountID(), master.Address(), ethAsset),
	)

	ledgerSeq := uint32(revokeTrustlineTxResp.Ledger)
	itest.WaitForLedgerInArchive(30*time.Second, ledgerSeq)
	ledger := getLedgers(itest, ledgerSeq, ledgerSeq)[ledgerSeq]

	events, _ := token_transfer.ProcessTokenTransferEventsFromLedger(ledger, itest.GetPassPhrase())
	fmt.Println("Printing all token transfer events from ledger:")

	// Assertions to follow soon, for now just printing
	printProtoEvents(events)

}
