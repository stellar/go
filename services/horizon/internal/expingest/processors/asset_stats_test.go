package processors

import (
	"math"
	"sort"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func TestEmptyAssetStatSet(t *testing.T) {
	set := assetStatSet{}
	if all := set.all(); len(all) != 0 {
		t.Fatalf("expected empty list but got %v", all)
	}
}

func TestAddAssetStatSet(t *testing.T) {
	set := assetStatSet{}
	eur := "EUR"
	err := set.add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all := set.all()
	if len(all) != 1 {
		t.Fatalf("expected list of 1 asset stat but got %v", all)
	}

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   eur,
		AssetIssuer: trustLineIssuer.Address(),
		Amount:      "1",
		NumAccounts: 1,
	}
	if all[0] != eurAssetStat {
		t.Fatalf("expected asset stat to be %v but got %v", eurAssetStat, all[0])
	}

	err = set.add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   24,
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all = set.all()
	if len(all) != 1 {
		t.Fatalf("expected list of 1 asset stat but got %v", all)
	}

	eurAssetStat.Amount = "25"
	eurAssetStat.NumAccounts++
	if all[0] != eurAssetStat {
		t.Fatalf("expected asset stat to be %v but got %v", eurAssetStat, all[0])
	}

	usd := "USD"
	err = set.add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset(usd, trustLineIssuer.Address()),
		Balance:   10,
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	ether := "ETHER"
	err = set.add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset(ether, trustLineIssuer.Address()),
		Balance:   3,
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	all = set.all()
	if len(all) != 3 {
		t.Fatalf("expected list of 1 asset stat but got %v", all)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].AssetCode < all[j].AssetCode
	})
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

	for i, got := range all {
		if expected[i] != got {
			t.Fatalf("expected asset stat to be %v but got %v", expected[i], got)
		}
	}
}

func TestOverflowAssetStatSet(t *testing.T) {
	set := assetStatSet{}
	eur := "EUR"
	err := set.add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   math.MaxInt64,
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all := set.all()
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

	err = set.add(xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()),
		Balance:   math.MaxInt64,
	})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all = set.all()
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
