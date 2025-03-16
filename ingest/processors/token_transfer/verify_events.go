package token_transfer

import (
	"fmt"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
	"sort"
)

type balanceKey struct {
	holder string
	asset  string
}

type keyAndAmount struct {
	key balanceKey
	amt int64
}

func fetchAccountDeltaFromChange(change ingest.Change) *keyAndAmount {
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
	return &keyAndAmount{key: balanceKey{holder: accountKey, asset: xlmAsset.StringCanonical()}, amt: delta}
}

func fetchTrustlineDeltaFromChange(change ingest.Change) *keyAndAmount {
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
	return &keyAndAmount{key: balanceKey{holder: trustlineKey, asset: asset}, amt: delta}
}

func fetchClaimableDeltaFromChange(change ingest.Change) *keyAndAmount {
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
	return &keyAndAmount{key: balanceKey{holder: cbKey, asset: asset}, amt: delta}
}

func fetchLiquidityPoolDeltaFromChange(change ingest.Change) []keyAndAmount {
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
	var entries []keyAndAmount
	if deltaA != 0 {
		entries = append(entries, keyAndAmount{key: balanceKey{holder: lpKey, asset: assetA}, amt: deltaA})
	}
	if deltaB != 0 {
		entries = append(entries, keyAndAmount{key: balanceKey{holder: lpKey, asset: assetB}, amt: deltaB})
	}
	return entries
}

func findBalanceDeltasFromChanges(changes []ingest.Change) map[balanceKey]int64 {
	var hashmap = make(map[balanceKey]int64)
	for _, change := range changes {
		switch change.Type {

		case xdr.LedgerEntryTypeAccount:
			entry := fetchAccountDeltaFromChange(change)
			if entry != nil {
				hashmap[entry.key] += entry.amt
				if hashmap[entry.key] == 0 {
					delete(hashmap, entry.key)
				}
			}

		case xdr.LedgerEntryTypeTrustline:
			entry := fetchTrustlineDeltaFromChange(change)
			if entry != nil {
				hashmap[entry.key] += entry.amt
				if hashmap[entry.key] == 0 {
					delete(hashmap, entry.key)
				}
			}

		case xdr.LedgerEntryTypeClaimableBalance:
			entry := fetchClaimableDeltaFromChange(change)
			if entry != nil {
				hashmap[entry.key] += entry.amt
				if hashmap[entry.key] == 0 {
					delete(hashmap, entry.key)
				}
			}

		case xdr.LedgerEntryTypeLiquidityPool:
			entries := fetchLiquidityPoolDeltaFromChange(change)
			for _, entry := range entries {
				hashmap[entry.key] += entry.amt
				if hashmap[entry.key] == 0 {
					delete(hashmap, entry.key)
				}
			}
		}
	}
	return hashmap
}

func findBalanceDeltasFromEvents(events []*TokenTransferEvent) map[balanceKey]int64 {
	var hashmap = make(map[balanceKey]int64)

	for _, event := range events {

		if event.Asset == nil { // needed toKey check for custom token events which wont have an asset
			continue
		}

		eventType := event.GetEventType()
		switch eventType {
		case "Fee":
			ev := event.GetFee()
			address := ev.From.StrKey
			asset := xlmAsset.StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			// Address' balance reduces by amt in FEE
			entry := balanceKey{holder: address, asset: asset}
			hashmap[entry] += -amt
			if hashmap[entry] == 0 {
				delete(hashmap, entry)
			}

		case "Transfer":
			ev := event.GetTransfer()
			fromAddress := ev.From.StrKey
			toAddress := ev.To.StrKey
			amt := int64(amount.MustParse(ev.Amount))
			asset := event.Asset.ToXdrAsset().StringCanonical()
			// FromAddress' balance reduces by amt in TRANSFER
			fromEntry := balanceKey{holder: fromAddress, asset: asset}
			// ToAddress' balance increases by amt in TRANSFER
			toEntry := balanceKey{holder: toAddress, asset: asset}
			hashmap[fromEntry] += -amt
			hashmap[toEntry] += amt
			if hashmap[toEntry] == 0 {
				delete(hashmap, toEntry)
			}
			if hashmap[fromEntry] == 0 {
				delete(hashmap, fromEntry)
			}

		case "Mint":
			ev := event.GetMint()
			toAddress := ev.To.StrKey
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			// ToAddress' balance increases by amt in MINT
			entry := balanceKey{holder: toAddress, asset: asset}
			hashmap[entry] += amt
			if hashmap[entry] == 0 {
				delete(hashmap, entry)
			}

		case "Burn":
			ev := event.GetBurn()
			fromAddress := ev.From.StrKey
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			// FromAddress' balance reduces by amt in BURN
			entry := balanceKey{holder: fromAddress, asset: asset}
			hashmap[entry] += -amt
			if hashmap[entry] == 0 {
				delete(hashmap, entry)
			}

		case "Clawback":
			ev := event.GetClawback()
			fromAddress := ev.From.StrKey
			asset := event.Asset.ToXdrAsset().StringCanonical()
			amt := int64(amount.MustParse(ev.Amount))
			// FromAddress' balance reduces by amt in CLAWBACK
			entry := balanceKey{holder: fromAddress, asset: asset}
			hashmap[entry] += -amt
			if hashmap[entry] == 0 {
				delete(hashmap, entry)
			}

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
	events, err := ProcessTokenTransferEventsFromLedger(ledger, passphrase)
	if err != nil {
		panic(errors.Wrapf(err, "unable to process token transfer events from ledger"))
	}

	changesMap := findBalanceDeltasFromChanges(changes)
	eventsMap := findBalanceDeltasFromEvents(events)

	fmt.Println("----ChangeMap-----")
	printMap(changesMap)
	fmt.Println("------")
	fmt.Println("----EventsMap-----")
	printMap(eventsMap)
	fmt.Println("------")

	return mapsEqual(eventsMap, changesMap)
}

// A custom type to implement sorting by balanceKey
type balanceKeySlice []balanceKey

func (s balanceKeySlice) Len() int {
	return len(s)
}

func (s balanceKeySlice) Less(i, j int) bool {
	// Sort first by holder, then by asset
	if s[i].holder == s[j].holder {
		return s[i].asset < s[j].asset
	}
	return s[i].holder < s[j].holder
}

func (s balanceKeySlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Function to print map in sorted order of keys
func printMap(m map[balanceKey]int64) {
	// Extract the keys from the map and sort them
	var keys []balanceKey
	for key := range m {
		keys = append(keys, key)
	}

	// Sort the keys
	sort.Sort(balanceKeySlice(keys))

	// Iterate over sorted keys and print the map in that order
	for _, key := range keys {
		fmt.Printf("Holder: %s, Asset: %s, Delta: %d\n", key.holder, key.asset, m[key])
	}
}
