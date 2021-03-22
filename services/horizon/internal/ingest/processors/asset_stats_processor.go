package processors

import (
	"database/sql"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type AssetStatsProcessor struct {
	assetStatsQ history.QAssetStats

	cache               *ingest.ChangeCompactor
	assetStatSet        AssetStatSet
	useLedgerEntryCache bool
}

// NewAssetStatsProcessor constructs a new AssetStatsProcessor instance.
// If useLedgerEntryCache is false we don't use ledger cache and we just
// add trust lines to assetStatSet, then we insert all the stats in one
// insert query. This is done to make history buckets processing faster
// (batch inserting).
func NewAssetStatsProcessor(
	assetStatsQ history.QAssetStats,
	useLedgerEntryCache bool,
) *AssetStatsProcessor {
	p := &AssetStatsProcessor{
		assetStatsQ:         assetStatsQ,
		useLedgerEntryCache: useLedgerEntryCache,
	}
	p.reset()
	return p
}

func (p *AssetStatsProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
	p.assetStatSet = AssetStatSet{}
}

func (p *AssetStatsProcessor) ProcessChange(change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeTrustline {
		return nil
	}

	if p.useLedgerEntryCache {
		err := p.cache.AddChange(change)
		if err != nil {
			return errors.Wrap(err, "error adding to ledgerCache")
		}

		if p.cache.Size() > maxBatchSize {
			err = p.Commit()
			if err != nil {
				return errors.Wrap(err, "error in Commit")
			}
			p.reset()
		}
		return nil
	}

	if !(change.Pre == nil && change.Post != nil) {
		return errors.New("AssetStatsProcessor is in insert only mode")
	}

	postTrustLine := change.Post.Data.MustTrustLine()
	err := p.adjustAssetStat(nil, &postTrustLine)
	if err != nil {
		return errors.Wrap(err, "Error adjusting asset stat")
	}

	return nil
}

func (p *AssetStatsProcessor) Commit() error {
	if !p.useLedgerEntryCache {
		return p.assetStatsQ.InsertAssetStats(p.assetStatSet.All(), maxBatchSize)
	}

	changes := p.cache.GetChanges()
	for _, change := range changes {
		var err error

		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			postTrustLine := change.Post.Data.MustTrustLine()
			err = p.adjustAssetStat(nil, &postTrustLine)
		case change.Pre != nil && change.Post != nil:
			// Updated
			preTrustLine := change.Pre.Data.MustTrustLine()
			postTrustLine := change.Post.Data.MustTrustLine()
			err = p.adjustAssetStat(&preTrustLine, &postTrustLine)
		case change.Pre != nil && change.Post == nil:
			// Removed
			preTrustLine := change.Pre.Data.MustTrustLine()
			err = p.adjustAssetStat(&preTrustLine, nil)
		default:
			return errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
		}

		if err != nil {
			return errors.Wrap(err, "Error adjusting asset stat")
		}
	}

	assetStatsDeltas := p.assetStatSet.All()
	for _, delta := range assetStatsDeltas {
		var rowsAffected int64
		var stat history.ExpAssetStat
		var err error

		stat, err = p.assetStatsQ.GetAssetStat(
			delta.AssetType,
			delta.AssetCode,
			delta.AssetIssuer,
		)
		assetStatNotFound := err == sql.ErrNoRows
		if !assetStatNotFound && err != nil {
			return errors.Wrap(err, "could not fetch asset stat from db")
		}

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
			}

			// Insert
			var errInsert error
			rowsAffected, errInsert = p.assetStatsQ.InsertAssetStat(delta)
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

			if statAccounts.IsZero() {
				// Remove stats
				if !statBalances.IsZero() {
					return ingest.NewStateError(errors.Errorf(
						"Removing asset stat by final amount non-zero for: %s %s %s",
						delta.AssetType,
						delta.AssetCode,
						delta.AssetIssuer,
					))
				}
				rowsAffected, err = p.assetStatsQ.RemoveAssetStat(
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				)
				if err != nil {
					return errors.Wrap(err, "could not remove asset stat")
				}
			} else {
				// Update
				rowsAffected, err = p.assetStatsQ.UpdateAssetStat(history.ExpAssetStat{
					AssetType:   delta.AssetType,
					AssetCode:   delta.AssetCode,
					AssetIssuer: delta.AssetIssuer,
					Accounts:    statAccounts,
					Balances:    statBalances.ConvertToHistoryObject(),
					Amount:      statBalances.Authorized.String(),
					NumAccounts: statAccounts.Authorized,
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

	return nil
}

func (p *AssetStatsProcessor) adjustAssetStat(
	preTrustline *xdr.TrustLineEntry,
	postTrustline *xdr.TrustLineEntry,
) error {
	deltaAccounts := map[xdr.Uint32]int32{}
	deltaBalances := map[xdr.Uint32]int64{}

	if preTrustline == nil && postTrustline == nil {
		return ingest.NewStateError(errors.New("both pre and post trustlines cannot be nil"))
	}

	var trustline xdr.TrustLineEntry
	if preTrustline != nil {
		trustline = *preTrustline
		deltaAccounts[preTrustline.Flags] -= 1
		deltaBalances[preTrustline.Flags] -= int64(preTrustline.Balance)
	}
	if postTrustline != nil {
		trustline = *postTrustline
		deltaAccounts[postTrustline.Flags] += 1
		deltaBalances[postTrustline.Flags] += int64(postTrustline.Balance)
	}

	err := p.assetStatSet.AddDelta(trustline.Asset, deltaBalances, deltaAccounts)
	if err != nil {
		return errors.Wrap(err, "error running AssetStatSet.AddDelta")
	}
	return nil
}
