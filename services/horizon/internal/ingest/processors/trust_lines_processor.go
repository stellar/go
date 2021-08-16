package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type TrustLinesProcessor struct {
	trustLinesQ history.QTrustLines

	cache *ingest.ChangeCompactor
}

func NewTrustLinesProcessor(trustLinesQ history.QTrustLines) *TrustLinesProcessor {
	p := &TrustLinesProcessor{trustLinesQ: trustLinesQ}
	p.reset()
	return p
}

func (p *TrustLinesProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
}

func (p *TrustLinesProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeTrustline {
		return nil
	}

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

func (p *TrustLinesProcessor) Commit(ctx context.Context) error {
	var batchUpsertTrustLines []history.TrustLine

	changes := p.cache.GetChanges()
	for _, change := range changes {
		var rowsAffected int64
		var err error
		var action string
		var ledgerKey xdr.LedgerKey

		switch {
		case change.Post != nil:
			// Created and updated
			trustLine := change.Post.Data.MustTrustLine()
			err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			var ledgerKeyString string
			ledgerKeyString, err = ledgerKey.MarshalBinaryBase64()
			if err != nil {
				return errors.Wrap(err, "Error marshalling ledger key")
			}

			assetType := trustLine.Asset.Type
			var assetCode, assetIssuer, poolID string
			if assetType == xdr.AssetTypeAssetTypePoolShare {
				poolID = poolIDToString(trustLine.Asset.MustLiquidityPoolId())
			} else {
				if err = trustLine.Asset.ToAsset().Extract(&assetType, &assetCode, &assetIssuer); err != nil {
					return errors.Wrap(err, "Error extracting asset from trustline")
				}
			}

			liabilities := trustLine.Liabilities()
			batchUpsertTrustLines = append(batchUpsertTrustLines, history.TrustLine{
				AccountID:          trustLine.AccountId.Address(),
				AssetType:          assetType,
				AssetIssuer:        assetIssuer,
				AssetCode:          assetCode,
				Balance:            int64(trustLine.Balance),
				LedgerKey:          ledgerKeyString,
				Limit:              int64(trustLine.Limit),
				LiquidityPoolID:    poolID,
				BuyingLiabilities:  int64(liabilities.Buying),
				SellingLiabilities: int64(liabilities.Selling),
				Flags:              uint32(trustLine.Flags),
				LastModifiedLedger: uint32(change.Post.LastModifiedLedgerSeq),
				Sponsor:            ledgerEntrySponsorToNullString(*change.Post),
			})
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			trustLine := change.Pre.Data.MustTrustLine()
			err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			ledgerKeyString, err := ledgerKey.MarshalBinaryBase64()
			if err != nil {
				return errors.Wrap(err, "Error marshalling ledger key")
			}

			rowsAffected, err = p.trustLinesQ.RemoveTrustLine(ctx, ledgerKeyString)
			if err != nil {
				return err
			}

			if rowsAffected != 1 {
				return ingest.NewStateError(errors.Errorf(
					"%d rows affected when %s trustline: %s %s",
					rowsAffected,
					action,
					ledgerKey.TrustLine.AccountId.Address(),
					ledgerKeyString,
				))
			}
		default:
			return errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
		}
	}

	// Upsert accounts
	if len(batchUpsertTrustLines) > 0 {
		err := p.trustLinesQ.UpsertTrustLines(ctx, batchUpsertTrustLines)
		if err != nil {
			return errors.Wrap(err, "errors in UpsertTrustLines")
		}
	}

	return nil
}
