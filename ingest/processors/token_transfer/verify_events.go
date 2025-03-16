package token_transfer

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
	"sort"
)

type Strkey string

type balanceKey struct {
	key   string
	asset string
	delta int64
}

func fetchAccountDeltaFromChange(change ingest.Change) *balanceKey {
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
	if delta == 0 {
		return nil
	}
	return &balanceKey{key: accountKey, asset: xlmAsset.StringCanonical(), delta: delta}
}

func fetchTrustlineDeltaFromChange(change ingest.Change) *balanceKey {
	var trustlineKey string
	var asset string
	var pre, post xdr.Int64

	if change.Pre != nil {
		entry := change.Pre.Data.MustTrustLine()
		if entry.Asset.Type == xdr.AssetTypeAssetTypePoolShare {
			return nil
		}
		trustlineKey = entry.AccountId.Address()
		pre = entry.Balance
		asset = entry.Asset.ToAsset().StringCanonical()
	}
	if change.Post != nil {
		entry := change.Post.Data.MustTrustLine()
		if entry.Asset.Type == xdr.AssetTypeAssetTypePoolShare {
			return nil
		}
		trustlineKey = entry.AccountId.Address()
		post = entry.Balance
		asset = entry.Asset.ToAsset().StringCanonical()
	}

	delta := int64(post - pre)
	if delta == 0 {
		return nil
	}
	return &balanceKey{key: trustlineKey, asset: asset, delta: delta}
}

func fetchLiquidityPoolDeltaFromChange(change ingest.Change) []balanceKey {
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
	var entries []balanceKey
	if deltaA != 0 {
		entries = append(entries, balanceKey{key: lpKey, asset: assetA, delta: deltaA})
	}
	if deltaB != 0 {
		entries = append(entries, balanceKey{key: lpKey, asset: assetB, delta: deltaB})
	}
	return entries
}

func fetchClaimableDeltaFromChange(change ingest.Change) *balanceKey {
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
	if delta == 0 {
		return nil
	}
	return &balanceKey{key: cbKey, asset: asset, delta: delta}
}

func findBalanceDeltaFromChanges(changes []ingest.Change) []balanceKey {
	var entries []balanceKey
	for _, change := range changes {
		switch change.Type {
		case xdr.LedgerEntryTypeAccount:
			entry := fetchAccountDeltaFromChange(change)
			if entry != nil {
				entries = append(entries, *entry)
			}
		case xdr.LedgerEntryTypeTrustline:

			entry := fetchTrustlineDeltaFromChange(change)
			if entry != nil {
				entries = append(entries, *entry)
			}
		case xdr.LedgerEntryTypeClaimableBalance:
			entry := fetchClaimableDeltaFromChange(change)
			if entry != nil {
				entries = append(entries, *entry)
			}
		case xdr.LedgerEntryTypeLiquidityPool:
			entries = append(entries, fetchLiquidityPoolDeltaFromChange(change)...)
		}
	}
	return entries
}

func balanceKeysToMap(entries []balanceKey) map[Strkey]mapset.Set[balanceKey] {
	var strkeysToBalancesMap = make(map[Strkey]mapset.Set[balanceKey])
	for _, entry := range entries {
		key := Strkey(entry.key)
		// Check if the set for the key is nil, if so, initialize it
		if strkeysToBalancesMap[key] == nil {
			strkeysToBalancesMap[key] = mapset.NewSet[balanceKey]()
		}
		strkeysToBalancesMap[key].Add(entry)
	}
	return strkeysToBalancesMap
}

// Main function - part 1
func findBalanceDeltasFromChanges(changes []ingest.Change) map[Strkey]mapset.Set[balanceKey] {
	entries := findBalanceDeltaFromChanges(changes)
	return balanceKeysToMap(entries)
}

func findBalanceDeltasFromEvents(events []*TokenTransferEvent) map[Strkey]mapset.Set[balanceKey] {
	var entries []balanceKey

	for _, event := range events {

		if event.Asset == nil { // needed toKey check for custom token events which wont have an asset
			continue
		}

		eventType := event.GetEventType()
		switch eventType {
		case "Fee":
			ev := event.GetFee()
			key := ev.From.ToStrkey()
			asset := xlmAsset.StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			// Key's balance reduces by amt in FEE
			entry := balanceKey{key: key, asset: asset, delta: -amt}
			entries = append(entries, entry)

		case "Transfer":
			ev := event.GetTransfer()
			fromKey, toKey := ev.From.ToStrkey(), ev.To.ToStrkey()
			amt := int64(amount.MustParse(ev.Amount))
			asset := event.Asset.ToXdrAsset().StringCanonical()
			// FromKey's balance reduces by amt in TRANSFER
			fromEntry := balanceKey{key: fromKey, asset: asset, delta: -amt}
			//ToKey's balance increases by amt in TRANSFER
			toEntry := balanceKey{key: toKey, asset: asset, delta: amt}
			entries = append(entries, fromEntry, toEntry)

		case "Mint":
			ev := event.GetMint()
			key := ev.To.ToStrkey()
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			// Key's balance incrases by amt in MINT
			entry := balanceKey{key: key, asset: asset, delta: amt}
			entries = append(entries, entry)

		case "Burn":
			ev := event.GetBurn()
			key := ev.From.ToStrkey()
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			// Key's balance reduces by amt in BURN
			entry := balanceKey{key: key, asset: asset, delta: -amt}
			entries = append(entries, entry)

		case "Clawback":
			ev := event.GetClawback()
			key := ev.From.ToStrkey()
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			entry := balanceKey{key: key, asset: asset, delta: -amt}
			entries = append(entries, entry)

		default:
			panic(errors.Errorf("unknown event type %s", eventType))
		}
	}
	return balanceKeysToMap(entries)
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

func VerifyTtpOnLedger(ledger xdr.LedgerCloseMeta, passphrase string) bool {
	changes := getChangesFromLedger(ledger, passphrase)

	events, err := ProcessTokenTransferEventsFromLedger(ledger, passphrase)
	if err != nil {
		panic(errors.Wrapf(err, "unable to process token transfer events from ledger"))
	}
	changesMap := findBalanceDeltasFromChanges(changes)
	eventsMap := findBalanceDeltasFromEvents(events)
	fmt.Println("******")
	fmt.Println("ChangesMap")
	PrintMap(changesMap)
	fmt.Println("******")

	fmt.Println("******")
	fmt.Println("ChangesMap")
	PrintMap(eventsMap)
	fmt.Println("******")

	return mapsEqual(eventsMap, changesMap)
}

func mapsEqual(a, b map[Strkey]mapset.Set[balanceKey]) bool {
	if len(a) != len(b) {
		// If the maps have different lengths, they are not equal
		return false
	}

	// Compare each key and the associated sets
	for key, setA := range a {
		setB, ok := b[key]
		if !ok {
			// If key is missing in map b, they are not equal
			return false
		}

		// Compare the sets associated with the current key
		if !setA.Equal(setB) {
			// If the sets are not equal, the maps are not equal
			return false
		}
	}

	// If all keys and sets are equal, the maps are equal
	return true
}

// PrintMap prints the map in a sorted, deconstructed format
func PrintMap(m map[Strkey]mapset.Set[balanceKey]) {
	// Extract keys from the map and sort them
	var keys []Strkey
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j] // Sort by string value (Strkey is a string)
	})

	// Iterate over sorted keys and print the corresponding sets in sorted order
	for _, key := range keys {
		// Print the key
		fmt.Printf("Key: %s\n", key)

		// Convert the set to a slice and sort it
		set := m[key]
		var entries []balanceKey

		set.Each(
			func(entry balanceKey) bool {
				entries = append(entries, entry)
				return true
			})

		// Sort the entries by `asset` and then `delta`
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].asset == entries[j].asset {
				return entries[i].delta < entries[j].delta // Secondary sort by delta
			}
			return entries[i].asset < entries[j].asset // Primary sort by asset
		})

		// Print the sorted entries in the set
		for _, entry := range entries {
			fmt.Printf("  - Asset: %s, Delta: %d\n", entry.asset, entry.delta)
		}
	}
}
