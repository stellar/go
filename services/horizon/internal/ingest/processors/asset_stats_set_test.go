package processors

import (
	"github.com/stellar/go/keypair"
	"math"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func TestEmptyAssetStatSet(t *testing.T) {
	set := NewAssetStatSet("")
	all, m, cs := set.All()
	assert.Empty(t, all)
	assert.Empty(t, cs)
	assert.Empty(t, m)

	all, err := set.AllFromSnapshot()
	assert.Empty(t, all)
	assert.NoError(t, err)
}

func assertAllEquals(t *testing.T, set AssetStatSet, expected []history.ExpAssetStat) {
	all, m, cs := set.All()
	assert.Empty(t, m)
	assert.Empty(t, cs)
	assertAssetStatsAreEqual(t, all, expected)
}

func assertAssetStatsAreEqual(t *testing.T, all []history.ExpAssetStat, expected []history.ExpAssetStat) {
	assert.Len(t, all, len(expected))
	sort.Slice(all, func(i, j int) bool {
		return all[i].AssetCode < all[j].AssetCode
	})
	for i, got := range all {
		assert.True(t, expected[i].Equals(got))
	}
}

func assertAllFromSnapshotEquals(t *testing.T, set AssetStatSet, expected []history.ExpAssetStat) {
	all, err := set.AllFromSnapshot()
	assert.NoError(t, err)
	assertAssetStatsAreEqual(t, all, expected)
}

func TestAddContractData(t *testing.T) {
	xlmID, err := xdr.MustNewNativeAsset().ContractID("passphrase")
	assert.NoError(t, err)
	usdcIssuer := keypair.MustRandom().Address()
	usdcAsset := xdr.MustNewCreditAsset("USDC", usdcIssuer)
	usdcID, err := usdcAsset.ContractID("passphrase")
	assert.NoError(t, err)
	etherIssuer := keypair.MustRandom().Address()
	etherAsset := xdr.MustNewCreditAsset("ETHER", etherIssuer)
	etherID, err := etherAsset.ContractID("passphrase")
	assert.NoError(t, err)
	uniAsset := xdr.MustNewCreditAsset("UNI", etherIssuer)
	uniID, err := uniAsset.ContractID("passphrase")
	assert.NoError(t, err)

	set := NewAssetStatSet("passphrase")

	xlmContractData, err := AssetToContractData(true, "", "", xlmID)
	assert.NoError(t, err)
	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: xlmContractData,
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(xlmID, [32]byte{}, 100),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(uniID, [32]byte{}, 0),
		},
	})
	assert.NoError(t, err)

	usdcContractData, err := AssetToContractData(false, "USDC", usdcIssuer, usdcID)
	assert.NoError(t, err)
	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: usdcContractData,
		},
	})
	assert.NoError(t, err)

	etherContractData, err := AssetToContractData(false, "ETHER", etherIssuer, etherID)
	assert.NoError(t, err)
	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: etherContractData,
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(etherID, [32]byte{}, 50),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(etherID, [32]byte{1}, 150),
		},
	})
	assert.NoError(t, err)

	// negative balances will be ignored
	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: balanceToContractData(etherID, [32]byte{1}, xdr.Int128Parts{Hi: 1 << 63, Lo: 0}),
		},
	})
	assert.NoError(t, err)

	btcAsset := xdr.MustNewCreditAsset("BTC", etherIssuer)
	btcID, err := btcAsset.ContractID("passphrase")
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(btcID, [32]byte{2}, 300),
		},
	})
	assert.NoError(t, err)

	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset("ETHER", etherIssuer).ToTrustLineAsset(),
			Balance:   1,
			Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
		})),
	)

	all, m, cs := set.All()
	assert.Len(t, all, 1)
	etherAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
		AssetCode:   "ETHER",
		AssetIssuer: etherIssuer,
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 1,
	}
	assert.True(t, all[0].Equals(etherAssetStat))
	assert.Len(t, m, 2)
	assert.True(t, m[usdcID].Equals(usdcAsset))
	assert.True(t, m[etherID].Equals(etherAsset))
	assert.Len(t, cs, 2)
	assert.Equal(t, cs[etherID].numHolders, int32(2))
	assert.Zero(t, cs[etherID].balance.Cmp(big.NewInt(200)))
	assert.Equal(t, cs[btcID].numHolders, int32(1))
	assert.Zero(t, cs[btcID].balance.Cmp(big.NewInt(300)))

	usdcAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   "USDC",
		AssetIssuer: usdcIssuer,
		Accounts:    history.ExpAssetStatAccounts{},
		Balances:    newAssetStatBalance().ConvertToHistoryObject(),
		Amount:      "0",
		NumAccounts: 0,
		ContractID:  nil,
	}

	etherAssetStat.SetContractID(etherID)
	etherAssetStat.Balances.Contracts = "200"
	etherAssetStat.Accounts.Contracts = 2
	usdcAssetStat.SetContractID(usdcID)

	assertAllFromSnapshotEquals(t, set, []history.ExpAssetStat{
		etherAssetStat,
		usdcAssetStat,
	})
}

func TestUpdateContractBalance(t *testing.T) {
	usdcIssuer := keypair.MustRandom().Address()
	usdcAsset := xdr.MustNewCreditAsset("USDC", usdcIssuer)
	usdcID, err := usdcAsset.ContractID("passphrase")
	assert.NoError(t, err)
	etherIssuer := keypair.MustRandom().Address()
	etherAsset := xdr.MustNewCreditAsset("ETHER", etherIssuer)
	etherID, err := etherAsset.ContractID("passphrase")
	assert.NoError(t, err)
	btcAsset := xdr.MustNewCreditAsset("BTC", etherIssuer)
	btcID, err := btcAsset.ContractID("passphrase")
	assert.NoError(t, err)
	uniAsset := xdr.MustNewCreditAsset("UNI", etherIssuer)
	uniID, err := uniAsset.ContractID("passphrase")
	assert.NoError(t, err)

	set := NewAssetStatSet("passphrase")

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(usdcID, [32]byte{}, 50),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(usdcID, [32]byte{}, 100),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(usdcID, [32]byte{2}, 30),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(usdcID, [32]byte{2}, 100),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(usdcID, [32]byte{4}, 0),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(usdcID, [32]byte{4}, 100),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(etherID, [32]byte{}, 200),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(etherID, [32]byte{}, 50),
		},
	})
	assert.NoError(t, err)

	// negative balances will be ignored
	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(etherID, [32]byte{}, 200),
		},
		Post: &xdr.LedgerEntry{
			Data: balanceToContractData(etherID, [32]byte{1}, xdr.Int128Parts{Hi: 1 << 63, Lo: 0}),
		},
	})
	assert.NoError(t, err)

	// negative balances will be ignored
	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: balanceToContractData(etherID, [32]byte{1}, xdr.Int128Parts{Hi: 1 << 63, Lo: 0}),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(etherID, [32]byte{}, 200),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(btcID, [32]byte{2}, 300),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(btcID, [32]byte{2}, 300),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(btcID, [32]byte{2}, 0),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(btcID, [32]byte{2}, 0),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(btcID, [32]byte{2}, 0),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(btcID, [32]byte{2}, 0),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(uniID, [32]byte{2}, 300),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(uniID, [32]byte{3}, 100),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(uniID, [32]byte{3}, 0),
		},
	})
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: BalanceToContractData(uniID, [32]byte{4}, 100),
		},
		Post: &xdr.LedgerEntry{
			Data: BalanceToContractData(uniID, [32]byte{4}, 50),
		},
	})
	assert.NoError(t, err)

	all, m, cs := set.All()
	assert.Empty(t, all)
	assert.Empty(t, m)

	assert.Len(t, cs, 3)
	assert.Equal(t, cs[usdcID].numHolders, int32(1))
	assert.Zero(t, cs[usdcID].balance.Cmp(big.NewInt(220)))
	assert.Equal(t, cs[etherID].numHolders, int32(0))
	assert.Zero(t, cs[etherID].balance.Cmp(big.NewInt(-150)))
	assert.Equal(t, cs[uniID].numHolders, int32(-2))
	assert.Zero(t, cs[uniID].balance.Cmp(big.NewInt(-450)))

	all, err = set.AllFromSnapshot()
	assert.NoError(t, err)
	assert.Empty(t, all)
}

func TestRemoveContractData(t *testing.T) {
	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("passphrase")
	assert.NoError(t, err)
	set := NewAssetStatSet("passphrase")

	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	assert.NoError(t, err)
	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: eurContractData,
		},
	})
	assert.NoError(t, err)

	all, m, cs := set.All()
	assert.Empty(t, all)
	assert.Empty(t, cs)
	assert.Len(t, m, 1)
	asset, ok := m[eurID]
	assert.True(t, ok)
	assert.Nil(t, asset)
}

func TestChangeContractData(t *testing.T) {
	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("passphrase")
	assert.NoError(t, err)

	usdcIssuer := keypair.MustRandom().Address()
	usdcID, err := xdr.MustNewCreditAsset("USDC", usdcIssuer).ContractID("passphrase")
	assert.NoError(t, err)

	set := NewAssetStatSet("passphrase")

	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	assert.NoError(t, err)
	usdcContractData, err := AssetToContractData(false, "USDC", usdcIssuer, usdcID)
	assert.NoError(t, err)

	err = set.AddContractData(ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			Data: eurContractData,
		},
		Post: &xdr.LedgerEntry{
			Data: usdcContractData,
		},
	})
	assert.EqualError(t, err, "asset contract changed asset")
}

func TestAddNativeClaimableBalance(t *testing.T) {
	set := NewAssetStatSet("")
	claimableBalance := xdr.ClaimableBalanceEntry{
		BalanceId: xdr.ClaimableBalanceId{},
		Claimants: nil,
		Asset:     xdr.MustNewNativeAsset(),
		Amount:    100,
	}
	assert.NoError(t, set.AddClaimableBalance(
		ingest.Change{
			Post: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					ClaimableBalance: &claimableBalance,
				},
			},
		},
	))
	all, m, cs := set.All()
	assert.Empty(t, all)
	assert.Empty(t, m)
	assert.Empty(t, cs)
}

func trustlineChange(pre, post *xdr.TrustLineEntry) ingest.Change {
	c := ingest.Change{}
	if pre != nil {
		c.Pre = &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				TrustLine: pre,
			},
		}
	}
	if post != nil {
		c.Post = &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				TrustLine: post,
			},
		}
	}
	return c
}

func TestAddPoolShareTrustline(t *testing.T) {
	set := NewAssetStatSet("")
	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset: xdr.TrustLineAsset{
				Type:            xdr.AssetTypeAssetTypePoolShare,
				LiquidityPoolId: &xdr.PoolId{1, 2, 3},
			},
			Balance: 1,
			Flags:   xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
		},
		)),
	)
	all, m, cs := set.All()
	assert.Empty(t, all)
	assert.Empty(t, m)
	assert.Empty(t, cs)
}

func TestAddAssetStats(t *testing.T) {
	set := NewAssetStatSet("")
	eur := "EUR"
	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   eur,
		AssetIssuer: trustLineIssuer.Address(),
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 1,
	}

	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()).ToTrustLineAsset(),
			Balance:   1,
			Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
		},
		)),
	)
	assertAllEquals(t, set, []history.ExpAssetStat{eurAssetStat})

	eurAssetStat.Accounts.ClaimableBalances++
	eurAssetStat.Balances.ClaimableBalances = "23"
	eurAsset := xdr.MustNewCreditAsset(eur, trustLineIssuer.Address())
	assert.NoError(
		t,
		set.addDelta(
			eurAsset,
			delta{ClaimableBalances: 23},
			delta{ClaimableBalances: 1},
		),
	)

	assertAllEquals(t, set, []history.ExpAssetStat{eurAssetStat})

	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()).ToTrustLineAsset(),
			Balance:   24,
			Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag | xdr.TrustLineFlagsTrustlineClawbackEnabledFlag),
		})),
	)

	eurAssetStat.Balances.Authorized = "25"
	eurAssetStat.Amount = "25"
	eurAssetStat.Accounts.Authorized++
	eurAssetStat.NumAccounts++
	assertAllEquals(t, set, []history.ExpAssetStat{eurAssetStat})

	usd := "USD"
	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			Asset:     xdr.MustNewCreditAsset(usd, trustLineIssuer.Address()).ToTrustLineAsset(),
			Balance:   10,
			Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
		})),
	)

	ether := "ETHER"
	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			Asset:     xdr.MustNewCreditAsset(ether, trustLineIssuer.Address()).ToTrustLineAsset(),
			Balance:   3,
			Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
		})),
	)

	// AddTrustline an authorized_to_maintain_liabilities trust line
	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset(ether, trustLineIssuer.Address()).ToTrustLineAsset(),
			Balance:   4,
			Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedToMaintainLiabilitiesFlag),
		})),
	)

	// AddTrustline an unauthorized trust line
	assert.NoError(
		t,
		set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset(ether, trustLineIssuer.Address()).ToTrustLineAsset(),
			Balance:   5,
		})),
	)
	expected := []history.ExpAssetStat{
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetCode:   ether,
			AssetIssuer: trustLineIssuer.Address(),
			Accounts: history.ExpAssetStatAccounts{
				Authorized:                      1,
				AuthorizedToMaintainLiabilities: 1,
				Unauthorized:                    1,
			},
			Balances: history.ExpAssetStatBalances{
				Authorized:                      "3",
				AuthorizedToMaintainLiabilities: "4",
				Unauthorized:                    "5",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
				Contracts:                       "0",
			},
			Amount:      "3",
			NumAccounts: 1,
		},
		eurAssetStat,
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetCode:   usd,
			AssetIssuer: trustLineIssuer.Address(),
			Accounts: history.ExpAssetStatAccounts{
				Authorized: 1,
			},
			Balances: history.ExpAssetStatBalances{
				Authorized:                      "10",
				AuthorizedToMaintainLiabilities: "0",
				Unauthorized:                    "0",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
				Contracts:                       "0",
			},
			Amount:      "10",
			NumAccounts: 1,
		},
	}
	assertAllEquals(t, set, expected)
}

func TestOverflowAssetStatSet(t *testing.T) {
	set := NewAssetStatSet("")
	eur := "EUR"
	err := set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   math.MaxInt64,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all, m, cs := set.All()
	if len(all) != 1 {
		t.Fatalf("expected list of 1 asset stat but got %v", all)
	}
	assert.Empty(t, m)
	assert.Empty(t, cs)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   eur,
		AssetIssuer: trustLineIssuer.Address(),
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "9223372036854775807",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "9223372036854775807",
		NumAccounts: 1,
	}
	if all[0] != eurAssetStat {
		t.Fatalf("expected asset stat to be %v but got %v", eurAssetStat, all[0])
	}

	err = set.AddTrustline(trustlineChange(nil, &xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset(eur, trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   math.MaxInt64,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	all, m, cs = set.All()
	if len(all) != 1 {
		t.Fatalf("expected list of 1 asset stat but got %v", all)
	}
	assert.Empty(t, m)
	assert.Empty(t, cs)

	eurAssetStat = history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   eur,
		AssetIssuer: trustLineIssuer.Address(),
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 2,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "18446744073709551614",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "18446744073709551614",
		NumAccounts: 2,
	}
	if all[0] != eurAssetStat {
		t.Fatalf("expected asset stat to be %v but got %v", eurAssetStat, all[0])
	}
}
