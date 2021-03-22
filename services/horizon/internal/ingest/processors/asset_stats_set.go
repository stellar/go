package processors

import (
	"math/big"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type assetStatKey struct {
	assetType   xdr.AssetType
	assetCode   string
	assetIssuer string
}

type assetStatValue struct {
	assetStatKey
	balances assetStatBalances
	accounts history.ExpAssetStatAccounts
}

type assetStatBalances struct {
	Authorized                      *big.Int
	AuthorizedToMaintainLiabilities *big.Int
	Unauthorized                    *big.Int
}

func (a *assetStatBalances) Parse(b *history.ExpAssetStatBalances) error {
	authorized, ok := new(big.Int).SetString(b.Authorized, 10)
	if !ok {
		return errors.New("Error parsing: " + b.Authorized)
	}
	a.Authorized = authorized

	authorizedToMaintainLiabilities, ok := new(big.Int).SetString(b.AuthorizedToMaintainLiabilities, 10)
	if !ok {
		return errors.New("Error parsing: " + b.AuthorizedToMaintainLiabilities)
	}
	a.AuthorizedToMaintainLiabilities = authorizedToMaintainLiabilities

	unauthorized, ok := new(big.Int).SetString(b.Unauthorized, 10)
	if !ok {
		return errors.New("Error parsing: " + b.Unauthorized)
	}
	a.Unauthorized = unauthorized

	return nil
}

func (a assetStatBalances) Add(b assetStatBalances) assetStatBalances {
	return assetStatBalances{
		Authorized:                      big.NewInt(0).Add(a.Authorized, b.Authorized),
		AuthorizedToMaintainLiabilities: big.NewInt(0).Add(a.AuthorizedToMaintainLiabilities, b.AuthorizedToMaintainLiabilities),
		Unauthorized:                    big.NewInt(0).Add(a.Unauthorized, b.Unauthorized),
	}
}

func (a assetStatBalances) IsZero() bool {
	return a.Authorized.Cmp(big.NewInt(0)) == 0 && a.AuthorizedToMaintainLiabilities.Cmp(big.NewInt(0)) == 0 && a.Unauthorized.Cmp(big.NewInt(0)) == 0
}

func (a assetStatBalances) ConvertToHistoryObject() history.ExpAssetStatBalances {
	return history.ExpAssetStatBalances{
		Authorized:                      a.Authorized.String(),
		AuthorizedToMaintainLiabilities: a.AuthorizedToMaintainLiabilities.String(),
		Unauthorized:                    a.Unauthorized.String(),
	}
}

func (value assetStatValue) ConvertToHistoryObject() history.ExpAssetStat {
	balances := value.balances.ConvertToHistoryObject()
	return history.ExpAssetStat{
		AssetType:   value.assetType,
		AssetCode:   value.assetCode,
		AssetIssuer: value.assetIssuer,
		Accounts:    value.accounts,
		Balances:    balances,
		Amount:      balances.Authorized,
		NumAccounts: value.accounts.Authorized,
	}
}

// AssetStatSet represents a collection of asset stats
type AssetStatSet map[assetStatKey]*assetStatValue

// Add updates the set with a trustline entry from a history archive snapshot.
func (s AssetStatSet) Add(trustLine xdr.TrustLineEntry) error {
	flags := trustLine.Flags
	return s.AddDelta(
		trustLine.Asset,
		map[xdr.Uint32]int64{flags: int64(trustLine.Balance)},
		map[xdr.Uint32]int32{flags: 1},
	)
}

// AddDelta adds a delta balance and delta accounts to a given asset trustline.
func (s AssetStatSet) AddDelta(asset xdr.Asset, deltaBalances map[xdr.Uint32]int64, deltaAccounts map[xdr.Uint32]int32) error {
	accountsEmpty := true
	for _, v := range deltaAccounts {
		if v != 0 {
			accountsEmpty = false
			break
		}
	}
	balancesEmpty := true
	for _, v := range deltaBalances {
		if v != 0 {
			balancesEmpty = false
			break
		}
	}
	if accountsEmpty && balancesEmpty {
		return nil
	}

	var key assetStatKey
	if err := asset.Extract(&key.assetType, &key.assetCode, &key.assetIssuer); err != nil {
		return errors.Wrap(err, "could not extract asset info from trustline")
	}

	current, ok := s[key]
	if !ok {
		current = &assetStatValue{assetStatKey: key, balances: assetStatBalances{
			Authorized:                      big.NewInt(0),
			AuthorizedToMaintainLiabilities: big.NewInt(0),
			Unauthorized:                    big.NewInt(0),
		}}
		s[key] = current
	}

	for k, v := range deltaAccounts {
		flags := xdr.TrustLineFlags(k)
		if flags.IsAuthorized() {
			current.accounts.Authorized += v
		} else if flags.IsAuthorizedToMaintainLiabilitiesFlag() {
			current.accounts.AuthorizedToMaintainLiabilities += v
		} else {
			current.accounts.Unauthorized += v
		}
	}

	for k, v := range deltaBalances {
		flags := xdr.TrustLineFlags(k)
		bigV := big.NewInt(v)
		if flags.IsAuthorized() {
			current.balances.Authorized.Add(current.balances.Authorized, bigV)
		} else if flags.IsAuthorizedToMaintainLiabilitiesFlag() {
			current.balances.AuthorizedToMaintainLiabilities.Add(current.balances.AuthorizedToMaintainLiabilities, bigV)
		} else {
			current.balances.Unauthorized.Add(current.balances.Unauthorized, bigV)
		}
	}

	// Note: it's possible that after operations above:
	// numAccounts != 0 && amount == 0 (ex. two accounts send some of their assets to third account)
	//  OR
	// numAccounts == 0 && amount != 0 (ex. issuer issued an asset)
	if current.balances.Authorized.Cmp(big.NewInt(0)) == 0 &&
		current.balances.AuthorizedToMaintainLiabilities.Cmp(big.NewInt(0)) == 0 &&
		current.balances.Unauthorized.Cmp(big.NewInt(0)) == 0 &&
		current.accounts.Authorized == 0 &&
		current.accounts.AuthorizedToMaintainLiabilities == 0 &&
		current.accounts.Unauthorized == 0 {
		delete(s, key)
	}

	return nil
}

// Remove deletes an asset stat from the set
func (s AssetStatSet) Remove(assetType xdr.AssetType, assetCode string, assetIssuer string) (history.ExpAssetStat, bool) {
	key := assetStatKey{assetType: assetType, assetIssuer: assetIssuer, assetCode: assetCode}
	value, ok := s[key]
	if !ok {
		return history.ExpAssetStat{}, false
	}

	delete(s, key)

	return value.ConvertToHistoryObject(), true
}

// All returns a list of all `history.ExpAssetStat` contained within the set
func (s AssetStatSet) All() []history.ExpAssetStat {
	assetStats := make([]history.ExpAssetStat, 0, len(s))
	for _, value := range s {
		assetStats = append(assetStats, value.ConvertToHistoryObject())
	}
	return assetStats
}
