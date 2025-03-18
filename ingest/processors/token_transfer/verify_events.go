package token_transfer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/support/collections/maps"
	"github.com/stellar/go/xdr"
	"io"
	"slices"
)

// balanceKey represents a unique holder-asset pair for tracking balance changes
type balanceKey struct {
	holder string
	asset  string
}

// updateBalanceMap updates the map and removes the entry if the value becomes 0
func updateBalanceMap(m map[balanceKey]int64, key balanceKey, delta int64) {
	m[key] += delta
	if m[key] == 0 {
		delete(m, key)
	}
}

func fetchAccountDeltaFromChange(change ingest.Change, m map[balanceKey]int64) {
	var accountKey string
	var pre, post xdr.Int64

	if change.Pre != nil {
		entry := change.Pre.Data.MustAccount()
		accountKey = entry.AccountId.Address()
		pre = entry.Balance
	}
	if change.Post != nil {
		entry := change.Post.Data.MustAccount()
		accountKey = entry.AccountId.Address()
		post = entry.Balance
	}

	delta := int64(post - pre)
	updateBalanceMap(m, balanceKey{holder: accountKey, asset: xlmAsset.StringCanonical()}, delta)
}

func fetchTrustlineDeltaFromChange(change ingest.Change, m map[balanceKey]int64) {
	var trustlineKey string
	var asset string
	var pre, post xdr.Int64

	if change.Pre != nil {
		entry := change.Pre.Data.MustTrustLine()
		if entry.Asset.Type == xdr.AssetTypeAssetTypePoolShare {
			return // Skip pool share assets
		}
		trustlineKey = entry.AccountId.Address()
		pre = entry.Balance
		asset = entry.Asset.ToAsset().StringCanonical()
	}
	if change.Post != nil {
		entry := change.Post.Data.MustTrustLine()
		if entry.Asset.Type == xdr.AssetTypeAssetTypePoolShare {
			return // Skip pool share assets
		}
		trustlineKey = entry.AccountId.Address()
		post = entry.Balance
		asset = entry.Asset.ToAsset().StringCanonical()
	}

	delta := int64(post - pre)
	updateBalanceMap(m, balanceKey{holder: trustlineKey, asset: asset}, delta)
}

func fetchClaimableDeltaFromChange(change ingest.Change, m map[balanceKey]int64) {
	var cbKey string
	var asset string
	var pre, post xdr.Int64

	if change.Pre != nil {
		entry := change.Pre.Data.MustClaimableBalance()
		cbKey = cbIdToStrkey(entry.BalanceId)
		asset = entry.Asset.StringCanonical()
		pre = entry.Amount
	}
	if change.Post != nil {
		entry := change.Post.Data.MustClaimableBalance()
		cbKey = cbIdToStrkey(entry.BalanceId)
		asset = entry.Asset.StringCanonical()
		post = entry.Amount
	}

	delta := int64(post - pre)
	updateBalanceMap(m, balanceKey{holder: cbKey, asset: asset}, delta)
}

func fetchLiquidityPoolDeltaFromChange(change ingest.Change, m map[balanceKey]int64) {
	var lpKey string
	var assetA, assetB string
	var preA, preB, postA, postB xdr.Int64

	if change.Pre != nil {
		entry := change.Pre.Data.MustLiquidityPool()
		lpKey = lpIdToStrkey(entry.LiquidityPoolId)
		cp := entry.Body.ConstantProduct
		assetA, assetB = cp.Params.AssetA.StringCanonical(), cp.Params.AssetB.StringCanonical()
		preA, preB = cp.ReserveA, cp.ReserveB
	}

	if change.Post != nil {
		entry := change.Post.Data.MustLiquidityPool()
		lpKey = lpIdToStrkey(entry.LiquidityPoolId)
		cp := entry.Body.ConstantProduct
		assetA, assetB = cp.Params.AssetA.StringCanonical(), cp.Params.AssetB.StringCanonical()
		postA, postB = cp.ReserveA, cp.ReserveB
	}

	deltaA := int64(postA - preA)
	deltaB := int64(postB - preB)

	updateBalanceMap(m, balanceKey{holder: lpKey, asset: assetA}, deltaA)
	updateBalanceMap(m, balanceKey{holder: lpKey, asset: assetB}, deltaB)
}

// findBalanceDeltasFromChanges aggregates all balance changes from ledger entry changes
func findBalanceDeltasFromChanges(changes []ingest.Change) map[balanceKey]int64 {
	hashmap := make(map[balanceKey]int64)
	for _, change := range changes {
		switch change.Type {
		case xdr.LedgerEntryTypeAccount:
			fetchAccountDeltaFromChange(change, hashmap)
		case xdr.LedgerEntryTypeTrustline:
			fetchTrustlineDeltaFromChange(change, hashmap)
		case xdr.LedgerEntryTypeClaimableBalance:
			fetchClaimableDeltaFromChange(change, hashmap)
		case xdr.LedgerEntryTypeLiquidityPool:
			fetchLiquidityPoolDeltaFromChange(change, hashmap)
		}
	}
	return hashmap
}

// findBalanceDeltasFromEvents aggregates all balance changes from token transfer events
func findBalanceDeltasFromEvents(events []*TokenTransferEvent) map[balanceKey]int64 {
	hashmap := make(map[balanceKey]int64)

	for _, event := range events {
		if event.Asset == nil { // needed check for custom token events which won't have an asset
			continue
		}

		eventType := event.GetEventType()
		switch eventType {
		case "Fee":
			ev := event.GetFee()
			address := ev.From
			asset := xlmAsset.StringCanonical()
			amt := amount.MustParseInt64Raw(ev.Amount)
			// Address' balance reduces by amt in FEE
			updateBalanceMap(hashmap, balanceKey{holder: address, asset: asset}, -amt)

		case "Transfer":
			ev := event.GetTransfer()
			fromAddress := ev.From
			toAddress := ev.To
			amt := amount.MustParseInt64Raw(ev.Amount)
			asset := event.Asset.ToXdrAsset().StringCanonical()
			// FromAddress' balance reduces by amt in TRANSFER
			updateBalanceMap(hashmap, balanceKey{holder: fromAddress, asset: asset}, -amt)
			// ToAddress' balance increases by amt in TRANSFER
			updateBalanceMap(hashmap, balanceKey{holder: toAddress, asset: asset}, amt)

		case "Mint":
			ev := event.GetMint()
			toAddress := ev.To
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := amount.MustParseInt64Raw(ev.Amount)
			// ToAddress' balance increases by amt in MINT
			updateBalanceMap(hashmap, balanceKey{holder: toAddress, asset: asset}, amt)

		case "Burn":
			ev := event.GetBurn()
			fromAddress := ev.From
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := amount.MustParseInt64Raw(ev.Amount)
			// FromAddress' balance reduces by amt in BURN
			updateBalanceMap(hashmap, balanceKey{holder: fromAddress, asset: asset}, -amt)

		case "Clawback":
			ev := event.GetClawback()
			fromAddress := ev.From
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := amount.MustParseInt64Raw(ev.Amount)
			// FromAddress' balance reduces by amt in CLAWBACK
			updateBalanceMap(hashmap, balanceKey{holder: fromAddress, asset: asset}, -amt)

		default:
			panic(errors.Errorf("unknown event type %s", eventType))
		}
	}
	return hashmap
}

func getChangesFromLedger(ledger xdr.LedgerCloseMeta, passphrase string) []ingest.Change {
	changeReader, err := ingest.NewLedgerChangeReaderFromLedgerCloseMeta(passphrase, ledger)
	changes := make([]ingest.Change, 0)
	defer changeReader.Close()
	if err != nil {
		panic(errors.Wrapf(err, "unable to create ledger change reader"))
	}
	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(errors.Wrapf(err, "unable to read from ledger"))
		}
		changes = append(changes, change)
	}
	return changes
}

func mapsEqual(map1, map2 map[balanceKey]int64) bool {
	// First, check if the maps have the same length
	if len(map1) != len(map2) {
		return false
	}

	// Then, compare the keys and values in both maps
	for key, value := range map1 {
		if val, exists := map2[key]; !exists || val != value {
			return false
		}
	}

	return true
}

func VerifyTtpOnLedger(ledger xdr.LedgerCloseMeta, passphrase string) bool {
	changes := getChangesFromLedger(ledger, passphrase)
	ttp := NewEventsProcessor(passphrase)
	events, err := ttp.EventsFromLedger(ledger)
	if err != nil {
		panic(errors.Wrapf(err, "unable to process token transfer events from ledger"))
	}

	changesMap := findBalanceDeltasFromChanges(changes)
	eventsMap := findBalanceDeltasFromEvents(events)

	isSuccess := mapsEqual(eventsMap, changesMap)
	if !isSuccess {
		fmt.Println("----ChangeMap-----")
		printMap(changesMap)
		fmt.Println("------")
		fmt.Println("----EventsMap-----")
		printMap(eventsMap)
		fmt.Println("------")
	}
	return isSuccess
}

// Function to print map in sorted order of keys
func printMap(m map[balanceKey]int64) {
	keys := maps.Keys(m)

	// Stable sort
	slices.SortStableFunc(keys, func(a, b balanceKey) int {
		// Sort first by holder, then by asset
		if a.holder != b.holder {
			if a.holder < b.holder {
				return -1
			}
			return 1
		}
		if a.asset < b.asset {
			return -1
		}
		if a.asset > b.asset {
			return 1
		}
		return 0
	})

	// Iterate over sorted keys and print the map in that order
	for _, key := range keys {
		fmt.Printf("Holder: %s, Asset: %s, Delta: %d\n", key.holder, key.asset, m[key])
	}
}
