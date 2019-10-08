package processors

import (
	"math/big"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type assetStatKey struct {
	assetType   xdr.AssetType
	assetIssuer string
	assetCode   string
}
type assetStatValue struct {
	amount      *big.Int
	numAccounts int32
}
type assetStatSet map[assetStatKey]assetStatValue

func (s assetStatSet) add(trustLine xdr.TrustLineEntry) error {
	var key assetStatKey
	if err := trustLine.Asset.Extract(&key.assetType, &key.assetCode, &key.assetIssuer); err != nil {
		return errors.Wrap(err, "could not extract asset info from trustline")
	}

	current := s[key]
	amount := current.amount
	if amount == nil {
		amount = big.NewInt(int64(trustLine.Balance))
	} else {
		amount = amount.Add(amount, big.NewInt(int64(trustLine.Balance)))
	}
	s[key] = assetStatValue{
		amount:      amount,
		numAccounts: current.numAccounts + 1,
	}
	return nil
}

func (s assetStatSet) all() []history.ExpAssetStat {
	assetStats := make([]history.ExpAssetStat, 0, len(s))
	for key, value := range s {
		assetStats = append(assetStats, history.ExpAssetStat{
			AssetType:   key.assetType,
			AssetCode:   key.assetCode,
			AssetIssuer: key.assetIssuer,
			Amount:      value.amount.String(),
			NumAccounts: value.numAccounts,
		})
	}
	return assetStats
}
