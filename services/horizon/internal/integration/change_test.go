package integration

import (
	"context"
	"encoding/json"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"io"
	"sort"
	"testing"
	"time"
)

func TestProtocolUpgradeChanges(t *testing.T) {
	tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{SkipHorizonStart: true})

	upgradedLedgerAppx, _ := itest.GetUpgradedLedgerSeqAppx()
	waitForLedgerInArchive(t, 15*time.Second, upgradedLedgerAppx)

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
	waitForLedgerInArchive(t, 15*time.Second, ledgerSeq)

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

func TestSomething2(t *testing.T) {
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()
	keys, accounts := itest.CreateAccounts(3, "1000")
	keyA, keyB := keys[0], keys[1]
	accountA, accountB := accounts[0], accounts[1]

	t.Logf("Key A: %v, Key B: %v", keyA.Address(), keyB.Address())
	t.Logf("Acc A: %v, Acc B: %v", accountA, accountB)

	// Some random asset
	xyzAsset := txnbuild.CreditAsset{Code: "XYZ", Issuer: master.Address()}
	itest.MustEstablishTrustline(keyA, accountA, xyzAsset)
	itest.MustEstablishTrustline(keyB, accountB, xyzAsset)

	// Make it so that A has some amount of xyzAsset
	paymentOperation := txnbuild.Payment{
		Destination: keyA.Address(),
		Asset:       xyzAsset,
		Amount:      "2000",
	}
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &paymentOperation)

	t.Logf("Payment operation response: %v", get_json_string(txResp))

	txResp2 := itest.MustGetAccount(keyA)
	t.Logf("Balances of acount A - %v,  %v", keyA.Address(), get_json_string(txResp2.Balances))

	txResp2 = itest.MustGetAccount(keyB)
	t.Logf("Balances of acount B - %v,  %v", keyB.Address(), get_json_string(txResp2.Balances))

	t.Logf("Get Account for account A - %v response before offer generation: %v", keyA.Address(), get_json_string(txResp))

	claim := itest.MustCreateClaimableBalance(
		keyA, xyzAsset, "42",
		txnbuild.NewClaimant(keyB.Address(), nil))
	t.Logf("Details about claim: %v", get_json_string(claim))

	txResp3 := itest.MustGetAccount(keyA)
	t.Logf("Balance of account A -  %v, after creating claimabale balacne: %v", keyA.Address(), get_json_string(txResp3.Balances))
	txResp4 := itest.MustGetAccount(keyB)
	t.Logf("Balance of account B -  %v, after creating claimabale balacne: %v", keyA.Address(), get_json_string(txResp4.Balances))

}

func get_json_string(input interface{}) string {
	data, _ := json.MarshalIndent(input, "", "  ")
	return string(data)
}

func TestSomething(t *testing.T) {
	//tt := assert.New(t)
	itest := integration.NewTest(t, integration.Config{})
	master := itest.Master()
	keys, accounts := itest.CreateAccounts(3, "1000")
	keyA, keyB := keys[0], keys[1]
	accountA, accountB := accounts[0], accounts[1]

	// Some random asset
	xyzAsset := txnbuild.CreditAsset{Code: "XYZ", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(keyA, accountA, xyzAsset)

	itest.MustEstablishTrustline(keyB, accountB, xyzAsset)

	t.Logf("*****")
	paymentOperation := txnbuild.Payment{
		Destination: keyA.Address(),
		Asset:       xyzAsset,
		Amount:      "2000",
	}
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(), &paymentOperation)
	data, _ := json.MarshalIndent(txResp, "", "  ")

	t.Logf("Acc A: %v, Acc B: %v", keyA.Address(), keyB.Address())

	sellOfferOperationFromA := txnbuild.ManageSellOffer{
		Selling:       xyzAsset,
		Buying:        txnbuild.NativeAsset{},
		Amount:        "50",
		Price:         xdr.Price{N: 1, D: 1},
		SourceAccount: keyA.Address(),
	}
	txResp = itest.MustSubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, keyA}, &sellOfferOperationFromA)
	data, _ = json.MarshalIndent(txResp, "", "  ")
	t.Logf("Transaction Sell Offer: %v", string(data))
	t.Logf("Tx response meta xdr Sell Offer: %v", txResp.ResultMetaXdr)
	t.Logf("*****")

	sellOfferLedgerSeq := uint32(txResp.Ledger)

	buyOfferOperationFromB := txnbuild.ManageBuyOffer{
		Buying:        xyzAsset,
		Selling:       txnbuild.NativeAsset{},
		Amount:        "66",
		Price:         xdr.Price{N: 1, D: 1},
		SourceAccount: keyB.Address(),
	}

	txResp = itest.MustSubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, keyB}, &buyOfferOperationFromB)
	data, _ = json.MarshalIndent(txResp, "", "  ")
	t.Logf("Transaction Buy Offer: %v", string(data))
	t.Logf("Tx response meta xdr Buy Offer: %v", txResp.ResultMetaXdr)
	t.Logf("*****")

	buyOfferLedgerSeq := uint32(txResp.Ledger)

	t.Logf("Sell Offer Ledger:%v, Buy Offer Ledger: %v", sellOfferLedgerSeq, buyOfferLedgerSeq)

	waitForLedgerInArchive(t, 15*time.Second, buyOfferLedgerSeq)

	ledgerMap := getLedgers(itest, sellOfferLedgerSeq, buyOfferLedgerSeq)
	for ledgerSeq, ledger := range ledgerMap {
		t.Logf("LedgerSeq:::::::::::::::::::::: %v", ledgerSeq)
		changes := getChangesFromLedger(itest, ledger)
		for _, change := range changes {
			if change.Reason != ingest.LedgerEntryChangeReasonOperation {
				continue
			}
			typ := change.Type.String()
			pre, _ := change.Pre.MarshalBinaryBase64()
			post, _ := change.Post.MarshalBinaryBase64()
			t.Logf("ledger: %v, Change - type - %v, pre: %v, post: %v", ledgerSeq, typ, pre, post)
		}
	}
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
	if err != nil {
		t.Fatalf("unable to create captive core config: %v", err)
	}

	captiveCore, err := ledgerbackend.NewCaptive(*ccConfig)
	if err != nil {
		t.Fatalf("unable to create captive core: %v", err)
	}
	defer captiveCore.Close()

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

func waitForLedgerInArchive(t *testing.T, waitTime time.Duration, ledgerSeq uint32) {
	archive, err := integration.GetHistoryArchive()
	if err != nil {
		t.Fatalf("could not get history archive: %v", err)
	}

	var latestCheckpoint uint32

	assert.Eventually(t,
		func() bool {
			has, requestErr := archive.GetRootHAS()
			if requestErr != nil {
				t.Logf("Request to fetch checkpoint failed: %v", requestErr)
				return false
			}
			latestCheckpoint = has.CurrentLedger
			return latestCheckpoint >= ledgerSeq

		},
		waitTime,
		1*time.Second)
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
