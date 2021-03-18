package processors

import (
	"math/big"

	protocol "github.com/stellar/go/protocols/horizon"
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
	accounts assetStatNumAccounts
}
type assetStatBalances struct {
	Authorized                      *big.Int
	AuthorizedToMaintainLiabilities *big.Int
	Unauthorized                    *big.Int
}
type assetStatNumAccounts struct {
	Authorized                      int32
	AuthorizedToMaintainLiabilities int32
	Unauthorized                    int32
}

func (value assetStatValue) Finish() history.ExpAssetStat {
	balances := protocol.AssetStatBalances{
		Authorized:                      value.balances.Authorized.String(),
		AuthorizedToMaintainLiabilities: value.balances.AuthorizedToMaintainLiabilities.String(),
		Unauthorized:                    value.balances.Unauthorized.String(),
	}
	return history.ExpAssetStat{
		AssetType:   value.assetType,
		AssetCode:   value.assetCode,
		AssetIssuer: value.assetIssuer,
		Accounts:    protocol.AssetStatAccounts(value.accounts),
		Balances:    balances,
		Amount:      balances.Authorized,
		NumAccounts: value.accounts.Authorized,
	}
}

// AssetStatSet represents a collection of asset stats
type AssetStatSet map[assetStatKey]*assetStatValue

// Add updates the set with a trustline entry from a history archive snapshot
// if the trustline is authorized.
func (s AssetStatSet) Add(trustLine xdr.TrustLineEntry) error {
	return s.AddDelta(trustLine.Asset, int64(trustLine.Balance), 1, xdr.TrustLineFlags(trustLine.Flags))
}

// AddDelta adds a delta balance and delta accounts to a given asset trustline.
func (s AssetStatSet) AddDelta(asset xdr.Asset, deltaBalance int64, deltaAccounts int32, flags xdr.TrustLineFlags) error {
	if deltaBalance == 0 && deltaAccounts == 0 {
		return nil
	}

	var key assetStatKey
	if err := asset.Extract(&key.assetType, &key.assetCode, &key.assetIssuer); err != nil {
		return errors.Wrap(err, "could not extract asset info from trustline")
	}

	current, ok := s[key]
	if !ok {
		current = &assetStatValue{assetStatKey: key}
		s[key] = current
	}

	// TODO: Do we need to handle clawback authorized here?
	if flags.IsAuthorized() {
		current.balances.Authorized.Add(current.balances.Authorized, big.NewInt(int64(deltaBalance)))
		current.accounts.Authorized += deltaAccounts
	} else if flags.IsAuthorizedToMaintainLiabilitiesFlag() {
		current.balances.AuthorizedToMaintainLiabilities.Add(current.balances.AuthorizedToMaintainLiabilities, big.NewInt(int64(deltaBalance)))
		current.accounts.AuthorizedToMaintainLiabilities += deltaAccounts
	} else {
		current.balances.Unauthorized.Add(current.balances.Unauthorized, big.NewInt(int64(deltaBalance)))
		current.accounts.Unauthorized += deltaAccounts
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

	return value.Finish(), true
}

// All returns a list of all `history.ExpAssetStat` contained within the set
func (s AssetStatSet) All() []history.ExpAssetStat {
	assetStats := make([]history.ExpAssetStat, 0, len(s))
	for _, value := range s {
		assetStats = append(assetStats, value.Finish())
	}
	return assetStats
}
