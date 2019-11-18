package processors

import (
	"math"
	"sort"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func TestEmptyAssetStatSet(t *testing.T) {
	set := AssetStatSet{}
	if all := set.All(); len(all) != 0 {
		t.Fatalf("expected empty list but got %v", all)
	}

	_, ok := set.Remove(
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
	)
	if ok {
		t.Fatal("expected remove to return false")
	}
}

func assertAllEquals(t *testing.T, set AssetStatSet, expected []history.ExpAssetStat) {
	all := set.All()
	if len(all) != len(expected) {
		t.Fatalf("expected list of %v asset stats but got %v", len(expected), all)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].AssetCode < all[j].AssetCode
	})
	for i, got := range all {
		if expected[i] != got {
			t.Fatalf("expected asset stat to be %v but got %v", expected[i], got)
		}
	}
}

func TestAssetStatSetIgnoresUnauthorizedTrustlines(t *testing.T) {
	set := AssetStatSet{}
	err := set.Add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if all := set.All(); len(all) != 0 {
		t.Fatalf("expected empty list but got %v", all)
	}
}

func TestAddAndRemoveAssetStats(t *testing.T) {
	set := AssetStatSet{}
	eur := "EUR"
	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   eur,
		AssetIssuer: trustLineIssuer.Address(),
		Amount:      "1",
		NumAccounts: 1,
	}

	err := set.Add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   1,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	assertAllEquals(t, set, []history.ExpAssetStat{eurAssetStat})

	err = set.Add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   24,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	eurAssetStat.Amount = "25"
	eurAssetStat.NumAccounts++
	assertAllEquals(t, set, []history.ExpAssetStat{eurAssetStat})

	usd := "USD"
	err = set.Add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset(usd, trustLineIssuer.Address()),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	ether := "ETHER"
	err = set.Add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset(ether, trustLineIssuer.Address()),
		Balance:   3,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	expected := []history.ExpAssetStat{
		history.ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetCode:   ether,
			AssetIssuer: trustLineIssuer.Address(),
			Amount:      "3",
			NumAccounts: 1,
		},
		eurAssetStat,
		history.ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetCode:   usd,
			AssetIssuer: trustLineIssuer.Address(),
			Amount:      "10",
			NumAccounts: 1,
		},
	}
	assertAllEquals(t, set, expected)

	for i, assetStat := range expected {
		removed, ok := set.Remove(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
		if !ok {
			t.Fatal("expected remove to return true")
		}
		if removed != assetStat {
			t.Fatalf("expected removed asset stat to be %v but got %v", assetStat, removed)
		}

		assertAllEquals(t, set, expected[i+1:])
	}
}

func TestOverflowAssetStatSet(t *testing.T) {
	set := AssetStatSet{}
	eur := "EUR"
	err := set.Add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   math.MaxInt64,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all := set.All()
	if len(all) != 1 {
		t.Fatalf("expected list of 1 asset stat but got %v", all)
	}

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   eur,
		AssetIssuer: trustLineIssuer.Address(),
		Amount:      "9223372036854775807",
		NumAccounts: 1,
	}
	if all[0] != eurAssetStat {
		t.Fatalf("expected asset stat to be %v but got %v", eurAssetStat, all[0])
	}

	err = set.Add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   math.MaxInt64,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all = set.All()
	if len(all) != 1 {
		t.Fatalf("expected list of 1 asset stat but got %v", all)
	}

	eurAssetStat = history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   eur,
		AssetIssuer: trustLineIssuer.Address(),
		Amount:      "18446744073709551614",
		NumAccounts: 2,
	}
	if all[0] != eurAssetStat {
		t.Fatalf("expected asset stat to be %v but got %v", eurAssetStat, all[0])
	}
}
