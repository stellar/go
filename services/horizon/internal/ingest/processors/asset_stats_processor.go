package processors

import (
	"context"
	"database/sql"
	"encoding/hex"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type AssetStatsProcessor struct {
	assetStatsQ         history.QAssetStats
	cache               *ingest.ChangeCompactor
	assetStatSet        AssetStatSet
	useLedgerEntryCache bool
	networkPassphrase   string
}

// NewAssetStatsProcessor constructs a new AssetStatsProcessor instance.
// If useLedgerEntryCache is false we don't use ledger cache and we just
// add trust lines to assetStatSet, then we insert all the stats in one
// insert query. This is done to make history buckets processing faster
// (batch inserting).
func NewAssetStatsProcessor(
	assetStatsQ history.QAssetStats,
	networkPassphrase string,
	useLedgerEntryCache bool,
) *AssetStatsProcessor {
	p := &AssetStatsProcessor{
		assetStatsQ:         assetStatsQ,
		useLedgerEntryCache: useLedgerEntryCache,
		networkPassphrase:   networkPassphrase,
	}
	p.reset()
	return p
}

func (p *AssetStatsProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
	p.assetStatSet = NewAssetStatSet(p.networkPassphrase)
}

func (p *AssetStatsProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeLiquidityPool &&
		change.Type != xdr.LedgerEntryTypeClaimableBalance &&
		change.Type != xdr.LedgerEntryTypeTrustline &&
		change.Type != xdr.LedgerEntryTypeContractData {
		return nil
	}
	if p.useLedgerEntryCache {
		return p.addToCache(ctx, change)
	}
	if change.Pre != nil || change.Post == nil {
		return errors.New("AssetStatsProcessor is in insert only mode")
	}

	switch change.Type {
	case xdr.LedgerEntryTypeLiquidityPool:
		return p.assetStatSet.AddLiquidityPool(change)
	case xdr.LedgerEntryTypeClaimableBalance:
		return p.assetStatSet.AddClaimableBalance(change)
	case xdr.LedgerEntryTypeTrustline:
		return p.assetStatSet.AddTrustline(change)
	case xdr.LedgerEntryTypeContractData:
		return p.assetStatSet.AddContractData(change)
	default:
		return nil
	}
}

func (p *AssetStatsProcessor) addToCache(ctx context.Context, change ingest.Change) error {
	err := p.cache.AddChange(change)
	if err != nil {
		return errors.Wrap(err, "error adding to ledgerCache")
	}

	if p.cache.Size() > maxBatchSize {
		err = p.Commit(ctx)
		if err != nil {
			return errors.Wrap(err, "error in Commit")
		}
		p.reset()
	}
	return nil
}

func (p *AssetStatsProcessor) Commit(ctx context.Context) error {
	if !p.useLedgerEntryCache {
		assetStatsDeltas, err := p.assetStatSet.AllFromSnapshot()
		if err != nil {
			return err
		}
		if len(assetStatsDeltas) == 0 {
			return nil
		}

		return p.assetStatsQ.InsertAssetStats(ctx, assetStatsDeltas, maxBatchSize)
	}

	changes := p.cache.GetChanges()
	for _, change := range changes {
		var err error
		switch change.Type {
		case xdr.LedgerEntryTypeLiquidityPool:
			err = p.assetStatSet.AddLiquidityPool(change)
		case xdr.LedgerEntryTypeClaimableBalance:
			err = p.assetStatSet.AddClaimableBalance(change)
		case xdr.LedgerEntryTypeTrustline:
			err = p.assetStatSet.AddTrustline(change)
		case xdr.LedgerEntryTypeContractData:
			err = p.assetStatSet.AddContractData(change)
		default:
			return errors.Errorf("Change type %v is unexpected", change.Type)
		}

		if err != nil {
			return errors.Wrap(err, "Error adjusting asset stat")
		}
	}

	assetStatsDeltas, contractToAsset := p.assetStatSet.All()
	for _, delta := range assetStatsDeltas {
		var rowsAffected int64
		var stat history.ExpAssetStat
		var err error

		stat, err = p.assetStatsQ.GetAssetStat(ctx,
			delta.AssetType,
			delta.AssetCode,
			delta.AssetIssuer,
		)
		assetStatNotFound := err == sql.ErrNoRows
		if !assetStatNotFound && err != nil {
			return errors.Wrap(err, "could not fetch asset stat from db")
		}
		if !assetStatNotFound {
			delta.ContractID = stat.ContractID
		}

		contractID, _, err := contractIDForAsset(
			stat.AssetType == xdr.AssetTypeAssetTypeNative,
			stat.AssetCode,
			stat.AssetIssuer,
			p.networkPassphrase,
		)
		if err != nil {
			return errors.Wrap(err, "cannot compute contract id for asset")
		}
		if asset, ok := contractToAsset[contractID]; ok && asset == nil {
			delta.ContractID = nil
		} else if ok {
			delta.SetContractID(contractID)
		}
		delete(contractToAsset, contractID)

		if assetStatNotFound {
			// Safety checks
			if delta.Accounts.Authorized < 0 {
				return ingest.NewStateError(errors.Errorf(
					"Authorized accounts negative but DB entry does not exist for asset: %s %s %s",
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			} else if delta.Accounts.AuthorizedToMaintainLiabilities < 0 {
				return ingest.NewStateError(errors.Errorf(
					"AuthorizedToMaintainLiabilities accounts negative but DB entry does not exist for asset: %s %s %s",
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			} else if delta.Accounts.Unauthorized < 0 {
				return ingest.NewStateError(errors.Errorf(
					"Unauthorized accounts negative but DB entry does not exist for asset: %s %s %s",
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			} else if delta.Accounts.ClaimableBalances < 0 {
				return ingest.NewStateError(errors.Errorf(
					"Claimable balance negative but DB entry does not exist for asset: %s %s %s",
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			} else if delta.Accounts.LiquidityPools < 0 {
				return ingest.NewStateError(errors.Errorf(
					"Liquidity pools negative but DB entry does not exist for asset: %s %s %s",
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			}

			// Insert
			var errInsert error
			rowsAffected, errInsert = p.assetStatsQ.InsertAssetStat(ctx, delta)
			if errInsert != nil {
				return errors.Wrap(errInsert, "could not insert asset stat")
			}
		} else {
			var statBalances assetStatBalances
			if err = statBalances.Parse(&stat.Balances); err != nil {
				return errors.Wrap(err, "Error parsing balances")
			}

			var deltaBalances assetStatBalances
			if err = deltaBalances.Parse(&delta.Balances); err != nil {
				return errors.Wrap(err, "Error parsing balances")
			}

			statBalances = statBalances.Add(deltaBalances)
			statAccounts := stat.Accounts.Add(delta.Accounts)

			// only remove asset stat if the Metadata contract data ledger entry for the token contract
			// has also been removed.
			if statAccounts.IsZero() && delta.ContractID == nil {
				// Remove stats
				if !statBalances.IsZero() {
					return ingest.NewStateError(errors.Errorf(
						"Removing asset stat by final amount non-zero for: %s %s %s",
						delta.AssetType,
						delta.AssetCode,
						delta.AssetIssuer,
					))
				}
				rowsAffected, err = p.assetStatsQ.RemoveAssetStat(ctx,
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				)
				if err != nil {
					return errors.Wrap(err, "could not remove asset stat")
				}
			} else {
				// Update
				rowsAffected, err = p.assetStatsQ.UpdateAssetStat(ctx, history.ExpAssetStat{
					AssetType:   delta.AssetType,
					AssetCode:   delta.AssetCode,
					AssetIssuer: delta.AssetIssuer,
					Accounts:    statAccounts,
					Balances:    statBalances.ConvertToHistoryObject(),
					Amount:      statBalances.Authorized.String(),
					NumAccounts: statAccounts.Authorized,
					ContractID:  delta.ContractID,
				})
				if err != nil {
					return errors.Wrap(err, "could not update asset stat")
				}
			}
		}

		if rowsAffected != 1 {
			return ingest.NewStateError(errors.Errorf(
				"%d rows affected when adjusting asset stat for asset: %s %s %s",
				rowsAffected,
				delta.AssetType,
				delta.AssetCode,
				delta.AssetIssuer,
			))
		}
	}

	return p.updateContractIDs(ctx, contractToAsset)
}

func (p *AssetStatsProcessor) updateContractIDs(ctx context.Context, contractToAsset map[[32]byte]*xdr.Asset) error {
	for contractID, asset := range contractToAsset {
		if err := p.updateContractID(ctx, contractID, asset); err != nil {
			return err
		}
	}
	return nil
}

func (p *AssetStatsProcessor) updateContractID(ctx context.Context, contractID [32]byte, asset *xdr.Asset) error {
	var rowsAffected int64
	if asset == nil {
		stat, err := p.assetStatsQ.GetAssetStatByContract(ctx, contractID)
		if err == sql.ErrNoRows {
			return ingest.NewStateError(errors.Errorf(
				"row for asset is missing: %s",
				asset.String(),
			))
		}
		if err != nil {
			return errors.Wrap(err, "could not find asset stat by contract id")
		}
		if stat.ContractID == nil {
			return ingest.NewStateError(errors.Errorf(
				"row has no contract id to remove %s: %s %s %s",
				hex.EncodeToString(contractID[:]),
				stat.AssetType,
				stat.AssetCode,
				stat.AssetIssuer,
			))
		}
		if stat.Accounts.IsZero() {
			rowsAffected, err = p.assetStatsQ.RemoveAssetStat(ctx,
				stat.AssetType,
				stat.AssetCode,
				stat.AssetIssuer,
			)
			if err != nil {
				return errors.Wrap(err, "could not remove asset stat")
			}
		} else {
			stat.ContractID = nil
			rowsAffected, err = p.assetStatsQ.UpdateAssetStat(ctx, stat)
			if err != nil {
				return errors.Wrap(err, "could not update asset stat")
			}
		}
	} else {
		var assetType xdr.AssetType
		var assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)
		stat, err := p.assetStatsQ.GetAssetStat(ctx, assetType, assetCode, assetIssuer)
		if err == sql.ErrNoRows {
			row := history.ExpAssetStat{
				AssetType:   assetType,
				AssetCode:   assetCode,
				AssetIssuer: assetIssuer,
				Accounts:    history.ExpAssetStatAccounts{},
				Balances:    newAssetStatBalance().ConvertToHistoryObject(),
				Amount:      amount.String(0),
				NumAccounts: 0,
			}
			row.SetContractID(contractID)
			rowsAffected, err = p.assetStatsQ.InsertAssetStat(ctx, row)
			if err != nil {
				return errors.Wrap(err, "could not insert asset stat")
			}
		} else if err != nil {
			return errors.Wrap(err, "could not find asset stat by contract id")
		} else if dbContractID, ok := stat.GetContractID(); ok {
			return ingest.NewStateError(errors.Errorf(
				"attempting to set contract id %s but row %s already has contract id set: %s",
				hex.EncodeToString(contractID[:]),
				asset.String(),
				hex.EncodeToString(dbContractID[:]),
			))
		} else {
			stat.SetContractID(contractID)
			rowsAffected, err = p.assetStatsQ.UpdateAssetStat(ctx, stat)
			if err != nil {
				return errors.Wrap(err, "could not update asset stat")
			}
		}
	}

	if rowsAffected != 1 {
		return ingest.NewStateError(errors.Errorf(
			"%d rows affected when adjusting asset stat for asset: %s",
			rowsAffected,
			asset.String(),
		))
	}
	return nil
}
