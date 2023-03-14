package processors

import (
	"context"
	"database/sql"
	"encoding/hex"
	"math/big"

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

	assetStatsDeltas, contractToAsset, contractAssetStats := p.assetStatSet.All()
	for _, delta := range assetStatsDeltas {
		var rowsAffected int64
		var stat history.ExpAssetStat
		var err error
		// asset stats only supports non-native assets
		asset := xdr.MustNewCreditAsset(delta.AssetCode, delta.AssetIssuer)
		contractID, err := asset.ContractID(p.networkPassphrase)
		if err != nil {
			return errors.Wrap(err, "cannot compute contract id for asset")
		}

		if contractAssetStat, ok := contractAssetStats[contractID]; ok {
			delta.Balances.Contracts = contractAssetStat.balance.String()
			delta.Accounts.Contracts = contractAssetStat.numHolders
			delete(contractAssetStats, contractID)
		}

		stat, err = p.assetStatsQ.GetAssetStat(ctx,
			delta.AssetType,
			delta.AssetCode,
			delta.AssetIssuer,
		)
		assetStatNotFound := err == sql.ErrNoRows
		if !assetStatNotFound && err != nil {
			return errors.Wrap(err, "could not fetch asset stat from db")
		}
		assetStatFound := !assetStatNotFound
		if assetStatFound {
			delta.ContractID = stat.ContractID
		}

		if asset, ok := contractToAsset[contractID]; ok && asset == nil {
			if assetStatFound && stat.ContractID == nil {
				return ingest.NewStateError(errors.Errorf(
					"row has no contract id to remove %s: %s %s %s",
					hex.EncodeToString(contractID[:]),
					stat.AssetType,
					stat.AssetCode,
					stat.AssetIssuer,
				))
			}
			delta.ContractID = nil
		} else if ok {
			if assetStatFound && stat.ContractID != nil {
				return ingest.NewStateError(errors.Errorf(
					"attempting to set contract id %s but row %s already has contract id set: %s",
					hex.EncodeToString(contractID[:]),
					asset.String(),
					hex.EncodeToString((*stat.ContractID)[:]),
				))
			}
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
				"%d rows affected (expected exactly 1) when adjusting asset stat for asset: %s %s %s",
				rowsAffected,
				delta.AssetType,
				delta.AssetCode,
				delta.AssetIssuer,
			))
		}
	}

	if err := p.updateContractIDs(ctx, contractToAsset, contractAssetStats); err != nil {
		return err
	}
	return p.updateContractAssetStats(ctx, contractAssetStats)
}

func (p *AssetStatsProcessor) updateContractIDs(
	ctx context.Context,
	contractToAsset map[[32]byte]*xdr.Asset,
	contractAssetStats map[[32]byte]contractAssetStatValue,
) error {
	for contractID, asset := range contractToAsset {
		if err := p.updateContractID(ctx, contractAssetStats, contractID, asset); err != nil {
			return err
		}
	}
	return nil
}

// updateContractID will update the asset stat row for the corresponding asset to either add or remove the given contract id
func (p *AssetStatsProcessor) updateContractID(
	ctx context.Context,
	contractAssetStats map[[32]byte]contractAssetStatValue,
	contractID [32]byte,
	asset *xdr.Asset,
) error {
	var rowsAffected int64
	// asset is nil so we need to set the contract_id column to NULL
	if asset == nil {
		stat, err := p.assetStatsQ.GetAssetStatByContract(ctx, contractID)
		if err == sql.ErrNoRows {
			return ingest.NewStateError(errors.Errorf(
				"row for asset with contract %s is missing",
				hex.EncodeToString(contractID[:]),
			))
		}
		if err != nil {
			return errors.Wrap(err, "error querying asset by contract id")
		}

		if err = p.maybeAddContractAssetStat(contractAssetStats, contractID, &stat); err != nil {
			return errors.Wrapf(err, "could not update asset stat with contract id %v with contract delta", contractID)
		}

		if stat.Accounts.IsZero() {
			if !stat.Balances.IsZero() {
				return ingest.NewStateError(errors.Errorf(
					"asset stat has 0 holders but non zero balance: %s",
					hex.EncodeToString(contractID[:]),
				))
			}
			// the asset stat is empty so we can remove the row entirely
			rowsAffected, err = p.assetStatsQ.RemoveAssetStat(ctx,
				stat.AssetType,
				stat.AssetCode,
				stat.AssetIssuer,
			)
			if err != nil {
				return errors.Wrap(err, "could not remove asset stat")
			}
		} else if stat.Accounts.Contracts != 0 || stat.Balances.Contracts != "0" {
			return ingest.NewStateError(errors.Errorf(
				"asset stat has contract holders but is attempting to remove contract id: %s",
				hex.EncodeToString(contractID[:]),
			))
		} else {
			// update the row to set the contract_id column to NULL
			stat.ContractID = nil
			rowsAffected, err = p.assetStatsQ.UpdateAssetStat(ctx, stat)
			if err != nil {
				return errors.Wrap(err, "could not update asset stat")
			}
		}
	} else { // asset is non nil, so we need to populate the contract_id column
		var assetType xdr.AssetType
		var assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)
		stat, err := p.assetStatsQ.GetAssetStat(ctx, assetType, assetCode, assetIssuer)
		if err == sql.ErrNoRows {
			// there is no asset stat for the given asset so we need to create a new row
			row := history.ExpAssetStat{
				AssetType:   assetType,
				AssetCode:   assetCode,
				AssetIssuer: assetIssuer,
				Accounts:    history.ExpAssetStatAccounts{},
				Balances:    newAssetStatBalance().ConvertToHistoryObject(),
				Amount:      "0",
				NumAccounts: 0,
			}
			row.SetContractID(contractID)
			if err = p.maybeAddContractAssetStat(contractAssetStats, contractID, &row); err != nil {
				return errors.Wrapf(err, "could not update asset stat with contract id %v with contract delta", contractID)
			}

			rowsAffected, err = p.assetStatsQ.InsertAssetStat(ctx, row)
			if err != nil {
				return errors.Wrap(err, "could not insert asset stat")
			}
		} else if err != nil {
			return errors.Wrap(err, "error querying asset by asset code and issuer")
		} else if dbContractID, ok := stat.GetContractID(); ok {
			// the asset stat already has a column_id set which is unexpected (the column should be NULL)
			return ingest.NewStateError(errors.Errorf(
				"attempting to set contract id %s but row %s already has contract id set: %s",
				hex.EncodeToString(contractID[:]),
				asset.String(),
				hex.EncodeToString(dbContractID[:]),
			))
		} else {
			// update the column_id column
			stat.SetContractID(contractID)
			if err = p.maybeAddContractAssetStat(contractAssetStats, contractID, &stat); err != nil {
				return errors.Wrapf(err, "could not update asset stat with contract id %v with contract delta", contractID)
			}

			rowsAffected, err = p.assetStatsQ.UpdateAssetStat(ctx, stat)
			if err != nil {
				return errors.Wrap(err, "could not update asset stat")
			}
		}
	}

	if rowsAffected != 1 {
		// assert that we have updated exactly one row
		return ingest.NewStateError(errors.Errorf(
			"%d rows affected (expected exactly 1) when adjusting asset stat for asset: %s",
			rowsAffected,
			asset.String(),
		))
	}
	return nil
}

func (p *AssetStatsProcessor) addContractAssetStat(contractAssetStat contractAssetStatValue, stat *history.ExpAssetStat) error {
	stat.Accounts.Contracts += contractAssetStat.numHolders
	contracts, ok := new(big.Int).SetString(stat.Balances.Contracts, 10)
	if !ok {
		return errors.New("Error parsing: " + stat.Balances.Contracts)
	}
	stat.Balances.Contracts = (new(big.Int).Add(contracts, contractAssetStat.balance)).String()
	return nil
}

func (p *AssetStatsProcessor) maybeAddContractAssetStat(contractAssetStats map[[32]byte]contractAssetStatValue, contractID [32]byte, stat *history.ExpAssetStat) error {
	if contractAssetStat, ok := contractAssetStats[contractID]; ok {
		if err := p.addContractAssetStat(contractAssetStat, stat); err != nil {
			return err
		}
		delete(contractAssetStats, contractID)
	}
	return nil
}

func (p *AssetStatsProcessor) updateContractAssetStats(
	ctx context.Context,
	contractAssetStats map[[32]byte]contractAssetStatValue,
) error {
	for contractID, contractAssetStat := range contractAssetStats {
		if err := p.updateContractAssetStat(ctx, contractID, contractAssetStat); err != nil {
			return err
		}
	}
	return nil
}

// updateContractAssetStat will look up an asset stat by contract id and, if it exists,
// it will adjust the contract balance and holders based on contractAssetStatValue
func (p *AssetStatsProcessor) updateContractAssetStat(
	ctx context.Context,
	contractID [32]byte,
	contractAssetStat contractAssetStatValue,
) error {
	stat, err := p.assetStatsQ.GetAssetStatByContract(ctx, contractID)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "error querying asset by contract id")
	}
	if err = p.addContractAssetStat(contractAssetStat, &stat); err != nil {
		return errors.Wrapf(err, "could not update asset stat with contract id %v with contract delta", contractID)
	}

	var rowsAffected int64
	rowsAffected, err = p.assetStatsQ.UpdateAssetStat(ctx, stat)
	if err != nil {
		return errors.Wrap(err, "could not update asset stat")
	}

	if rowsAffected != 1 {
		// assert that we have updated exactly one row
		return ingest.NewStateError(errors.Errorf(
			"%d rows affected (expected exactly 1) when adjusting asset stat for asset: %s",
			rowsAffected,
			stat.AssetCode+":"+stat.AssetIssuer,
		))
	}
	return nil
}
