package integration

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/ingest/processors/token_transfer"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

func TestLiquidityPoolHappyPath2(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()

	// Give the master account the revocable flag (needed to set the clawback flag)
	setRevocableFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
		},
	}
	itest.MustSubmitOperations(itest.MasterAccount(), master, &setRevocableFlag)

	keys, accounts := itest.CreateAccounts(2, "1000000")
	shareKeys, shareAccount := keys[0], accounts[0]

	itest.MustSubmitMultiSigOperations(shareAccount, []*keypair.Full{shareKeys, master},
		&txnbuild.ChangeTrust{
			Line: txnbuild.ChangeTrustAssetWrapper{
				Asset: txnbuild.CreditAsset{
					Code:   "USD",
					Issuer: master.Address(),
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.ChangeTrust{
			Line: txnbuild.LiquidityPoolShareChangeTrustAsset{
				LiquidityPoolParameters: txnbuild.LiquidityPoolParameters{
					AssetA: txnbuild.NativeAsset{},
					AssetB: txnbuild.CreditAsset{
						Code:   "USD",
						Issuer: master.Address(),
					},
					Fee: 30,
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.Payment{
			SourceAccount: master.Address(),
			Destination:   shareAccount.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   "USD",
				Issuer: master.Address(),
			},
			Amount: "1000",
		},
	)

	poolID, err := xdr.NewPoolId(
		xdr.MustNewNativeAsset(),
		xdr.MustNewCreditAsset("USD", master.Address()),
		30,
	)
	tt.NoError(err)
	poolIDHexString := xdr.Hash(poolID).HexString()

	pools, err := itest.Client().LiquidityPools(horizonclient.LiquidityPoolsRequest{})
	tt.NoError(err)
	tt.Len(pools.Embedded.Records, 1)

	pool := pools.Embedded.Records[0]
	tt.Equal(poolIDHexString, pool.ID)
	tt.Equal(uint32(30), pool.FeeBP)
	tt.Equal("constant_product", pool.Type)
	tt.Equal("0.0000000", pool.TotalShares)
	tt.Equal(uint64(1), pool.TotalTrustlines)

	tt.Equal("0.0000000", pool.Reserves[0].Amount)
	tt.Equal("native", pool.Reserves[0].Asset)
	tt.Equal("0.0000000", pool.Reserves[1].Amount)
	tt.Equal(fmt.Sprintf("USD:%s", master.Address()), pool.Reserves[1].Asset)

	itest.MustSubmitOperations(shareAccount, shareKeys,
		&txnbuild.LiquidityPoolDeposit{
			LiquidityPoolID: [32]byte(poolID),
			MaxAmountA:      "400",
			MaxAmountB:      "777",
			MinPrice:        xdr.Price{N: 1, D: 2},
			MaxPrice:        xdr.Price{N: 2, D: 1},
		},
	)

	pool, err = itest.Client().LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: poolIDHexString,
	})
	tt.NoError(err)

	tt.Equal(poolIDHexString, pool.ID)
	tt.Equal(uint64(1), pool.TotalTrustlines)

	tt.Equal("400.0000000", pool.Reserves[0].Amount)
	tt.Equal("native", pool.Reserves[0].Asset)
	tt.Equal("777.0000000", pool.Reserves[1].Amount)
	tt.Equal(fmt.Sprintf("USD:%s", master.Address()), pool.Reserves[1].Asset)

	// Full clawback of the asset, with a deauthorize/reauthorize sandwich
	//revokeTrustlineOp := txnbuild.SetTrustLineFlags{
	//	Trustor: shareAccount.GetAccountID(),
	//	Asset: txnbuild.CreditAsset{
	//		Code:   "USD",
	//		Issuer: master.Address(),
	//	},
	//	ClearFlags: []txnbuild.TrustLineFlag{txnbuild.TrustLineAuthorized},
	//	SetFlags:   []txnbuild.TrustLineFlag{0},
	//}

	randomKeys, randomAccount := keys[1], accounts[1]
	signerKeys := []*keypair.Full{randomKeys, master}

	submissionResp := itest.MustSubmitMultiSigOperations(
		//itest.MasterAccount(),
		//master,
		randomAccount, // some other account pays the fees
		signerKeys,
		//randomKeys,
		&txnbuild.SetTrustLineFlags{
			SourceAccount: master.Address(),
			Trustor:       shareAccount.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   "USD",
				Issuer: master.Address(),
			},
			ClearFlags: []txnbuild.TrustLineFlag{txnbuild.TrustLineAuthorized},
			SetFlags:   []txnbuild.TrustLineFlag{0},
		})

	//fmt.Println("**********")
	//fmt.Println("submissionResp := ")
	// Marshal the struct to JSON with indentation
	//jsonString, _ := json.MarshalIndent(submissionResp, "", "    ") // Use 4 spaces for indentation
	// Print the indented JSON string
	//fmt.Println(string(jsonString))
	//fmt.Println("**********")

	if !submissionResp.Successful {
		return
	}
	fmt.Println("***** Transaction submission successful")
	ledgerSeq := uint32(submissionResp.Ledger)
	itest.WaitForLedgerInArchive(30*time.Second, ledgerSeq)
	ledger := getLedgers(itest, ledgerSeq, ledgerSeq)[ledgerSeq]
	changes := getChangesFromLedger(itest, ledger)

	lpIds := getLpIdsFromChanges(changes)
	cbEntries := getCbEntriesFromChanges(changes)
	lpMap := make(map[string]xdr.PoolId)
	cbMap := make(map[string]xdr.ClaimableBalanceEntry)

	for _, entry := range cbEntries {
		cbMap[entry.BalanceId.MustV0().HexString()] = entry
	}
	for _, entry := range lpIds {
		lpMap[xdr.Hash(entry).HexString()] = entry
	}

	asset := xdr.MustNewCreditAsset("USD", master.Address())

	//masterAccountId := xdr.MustAddress(itest.Master().Address())
	randomAccountId := xdr.MustAddress(randomAccount.GetAccountID())
	for _, entry := range lpIds {
		genCbId := generateCBIdFromLpId(entry, submissionResp.AccountSequence, randomAccountId, 0, asset)
		fmt.Printf("Constructed Claimable Balance Id from LP ----- %v\n", genCbId.HexString())
	}
	for _, entry := range cbEntries {
		fmt.Printf("CB Entry from changes CBId: %v\n", entry.BalanceId.MustV0().HexString())
	}

	events, _ := token_transfer.ProcessTokenTransferEventsFromLedger(ledger, itest.GetPassPhrase())
	fmt.Println("Printing all token transfer events from ledger:")
	printProtoEvents(events)

}

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

func TestProtocolUpgradeChanges(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{SkipHorizonStart: true})

	upgradedLedgerAppx, _ := itest.GetUpgradedLedgerSeqAppx()
	itest.WaitForLedgerInArchive(6*time.Minute, upgradedLedgerAppx)

	ledgerSeqToLedgers := getLedgers(itest, 2, upgradedLedgerAppx)

	// It is important to find the "exact" ledger which is representative of protocol upgrade
	// and the one before it, to check for upgrade related changes
	upgradedLedgerSeq := getExactUpgradedLedgerSeq(ledgerSeqToLedgers, itest.Config().ProtocolVersion)
	prevLedgerToUpgrade := upgradedLedgerSeq - 1

	prevLedgerChanges := getChangesFromLedger(itest, ledgerSeqToLedgers[prevLedgerToUpgrade])
	prevLedgerChangeMap := changeReasonCountMap(prevLedgerChanges)
	upgradedLedgerChanges := getChangesFromLedger(itest, ledgerSeqToLedgers[upgradedLedgerSeq])
	upgradedLedgerChangeMap := changeReasonCountMap(upgradedLedgerChanges)

	tt.Zero(prevLedgerChangeMap[ingest.LedgerEntryChangeReasonUpgrade])
	tt.NotZero(upgradedLedgerChangeMap[ingest.LedgerEntryChangeReasonUpgrade])
	for _, change := range upgradedLedgerChanges {
		tt.Equal(change.Ledger.LedgerSequence(), upgradedLedgerSeq)
		tt.Empty(change.Transaction)
		tt.NotEmpty(change.LedgerUpgrade)
	}
}

func TestOneTxOneOperationChanges(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})

	master := itest.Master()
	keys, _ := itest.CreateAccounts(2, "1000")
	srcAcc, destAcc := keys[0], keys[1]

	operation := txnbuild.Payment{
		SourceAccount: srcAcc.Address(),
		Destination:   destAcc.Address(),
		Asset:         txnbuild.NativeAsset{},
		Amount:        "900",
	}
	txResp, err := itest.SubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, srcAcc}, &operation)
	tt.NoError(err)

	ledgerSeq := uint32(txResp.Ledger)
	itest.WaitForLedgerInArchive(6*time.Minute, ledgerSeq)

	ledger := getLedgers(itest, ledgerSeq, ledgerSeq)[ledgerSeq]
	changes := getChangesFromLedger(itest, ledger)

	reasonCntMap := changeReasonCountMap(changes)
	tt.Equal(2, reasonCntMap[ingest.LedgerEntryChangeReasonOperation])
	tt.Equal(1, reasonCntMap[ingest.LedgerEntryChangeReasonTransaction])
	tt.Equal(1, reasonCntMap[ingest.LedgerEntryChangeReasonFee])

	reasonToChangeMap := changeReasonToChangeMap(changes)
	// Assert Transaction Hash and Ledger Sequence within Transaction are accurate in all changes
	for _, change := range changes {
		tt.Equal(change.Transaction.Hash.HexString(), txResp.Hash)
		tt.Equal(change.Transaction.Ledger.LedgerSequence(), ledgerSeq)
		tt.Empty(change.Ledger)
		tt.Empty(change.LedgerUpgrade)
	}

	feeRelatedChange := reasonToChangeMap[ingest.LedgerEntryChangeReasonFee][0]
	txRelatedChange := reasonToChangeMap[ingest.LedgerEntryChangeReasonTransaction][0]
	operationChanges := reasonToChangeMap[ingest.LedgerEntryChangeReasonOperation]

	accountFromEntry := func(e *xdr.LedgerEntry) xdr.AccountEntry {
		return e.Data.MustAccount()
	}

	changeForAccount := func(changes []ingest.Change, target string) ingest.Change {
		for _, change := range changes {
			acc := change.Pre.Data.MustAccount()
			if acc.AccountId.Address() == target {
				return change
			}
		}
		return ingest.Change{}
	}

	containsAccount := func(changes []ingest.Change, target string) bool {
		for _, change := range changes {
			addr := change.Pre.Data.MustAccount().AccountId.Address()
			if addr == target {
				return true
			}
		}
		return false
	}

	tt.Equal(accountFromEntry(feeRelatedChange.Pre).AccountId.Address(), master.Address())
	tt.Equal(accountFromEntry(txRelatedChange.Pre).AccountId.Address(), master.Address())
	tt.True(containsAccount(operationChanges, srcAcc.Address()))
	tt.True(containsAccount(operationChanges, destAcc.Address()))
	// MasterAccount shouldn't show up in operation level changes
	tt.False(containsAccount(operationChanges, master.Address()))
	tt.True(accountFromEntry(feeRelatedChange.Pre).Balance > accountFromEntry(feeRelatedChange.Post).Balance)
	tt.True(accountFromEntry(txRelatedChange.Post).SeqNum == accountFromEntry(txRelatedChange.Pre).SeqNum+1)

	srcAccChange := changeForAccount(operationChanges, srcAcc.Address())
	destAccChange := changeForAccount(operationChanges, destAcc.Address())
	tt.True(accountFromEntry(srcAccChange.Pre).Balance > accountFromEntry(srcAccChange.Post).Balance)
	tt.True(accountFromEntry(destAccChange.Pre).Balance < accountFromEntry(destAccChange.Post).Balance)
}

func getChangesFromLedger(itest *integration.Test, ledger xdr.LedgerCloseMeta) []ingest.Change {
	t := itest.CurrentTest()
	changeReader, err := ingest.NewLedgerChangeReaderFromLedgerCloseMeta(itest.GetPassPhrase(), ledger)
	changes := make([]ingest.Change, 0)
	defer changeReader.Close()
	if err != nil {
		t.Fatalf("unable to create ledger change reader: %v", err)
	}
	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unable to read ledger change: %v", err)
		}
		changes = append(changes, change)
	}
	return changes
}

func getLedgers(itest *integration.Test, startingLedger uint32, endLedger uint32) map[uint32]xdr.LedgerCloseMeta {
	t := itest.CurrentTest()

	ccConfig, err := itest.CreateCaptiveCoreConfig()
	require.NoError(t, err)

	captiveCore, err := ledgerbackend.NewCaptive(ccConfig)
	require.NoError(t, err)

	ctx := context.Background()
	err = captiveCore.PrepareRange(ctx, ledgerbackend.BoundedRange(startingLedger, endLedger))
	if err != nil {
		t.Fatalf("failed to prepare range: %v", err)
	}

	var seqToLedgersMap = make(map[uint32]xdr.LedgerCloseMeta)
	for ledgerSeq := startingLedger; ledgerSeq <= endLedger; ledgerSeq++ {
		ledger, err := captiveCore.GetLedger(ctx, ledgerSeq)
		if err != nil {
			t.Fatalf("failed to get ledgerNum: %v, error: %v", ledgerSeq, err)
		}
		seqToLedgersMap[ledgerSeq] = ledger
	}

	require.NoError(t, captiveCore.Close())
	return seqToLedgersMap
}

func changeReasonCountMap(changes []ingest.Change) map[ingest.LedgerEntryChangeReason]int {
	changeMap := make(map[ingest.LedgerEntryChangeReason]int)
	for _, change := range changes {
		changeMap[change.Reason]++
	}
	return changeMap
}

func changeReasonToChangeMap(changes []ingest.Change) map[ingest.LedgerEntryChangeReason][]ingest.Change {
	changeMap := make(map[ingest.LedgerEntryChangeReason][]ingest.Change)
	for _, change := range changes {
		changeMap[change.Reason] = append(changeMap[change.Reason], change)
	}
	return changeMap
}

func getExactUpgradedLedgerSeq(ledgerMap map[uint32]xdr.LedgerCloseMeta, version uint32) uint32 {
	keys := make([]int, 0, len(ledgerMap))
	for key := range ledgerMap {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)

	for _, key := range keys {
		if ledgerMap[uint32(key)].ProtocolVersion() == version {
			return uint32(key)
		}
	}
	return 0
}
