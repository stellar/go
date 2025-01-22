package processors

import (
	"encoding/base64"
	"fmt"

	"github.com/guregu/null"
	"github.com/pkg/errors"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

// TransformTrustline converts a trustline from the history archive ingestion system into a form suitable for BigQuery
func TransformTrustline(ledgerChange ingest.Change, header xdr.LedgerHeaderHistoryEntry) (TrustlineOutput, error) {
	ledgerEntry, changeType, outputDeleted, err := ExtractEntryFromChange(ledgerChange)
	if err != nil {
		return TrustlineOutput{}, err
	}

	trustEntry, ok := ledgerEntry.Data.GetTrustLine()
	if !ok {
		return TrustlineOutput{}, fmt.Errorf("could not extract trustline data from ledger entry; actual type is %s", ledgerEntry.Data.Type)
	}

	outputAccountID, err := trustEntry.AccountId.GetAddress()
	if err != nil {
		return TrustlineOutput{}, err
	}

	var assetType, outputAssetCode, outputAssetIssuer, poolID string

	asset := trustEntry.Asset

	outputLedgerKey, err := trustLineEntryToLedgerKeyString(trustEntry)
	if err != nil {
		return TrustlineOutput{}, errors.Wrap(err, fmt.Sprintf("could not create ledger key string for trustline with account %s and asset %s", outputAccountID, asset.ToAsset().StringCanonical()))
	}

	if asset.Type == xdr.AssetTypeAssetTypePoolShare {
		poolID = PoolIDToString(trustEntry.Asset.MustLiquidityPoolId())
		assetType = "pool_share"
	} else {
		if err = asset.Extract(&assetType, &outputAssetCode, &outputAssetIssuer); err != nil {
			return TrustlineOutput{}, errors.Wrap(err, fmt.Sprintf("could not parse asset for trustline with account %s", outputAccountID))
		}
	}

	outputAssetID := FarmHashAsset(outputAssetCode, outputAssetIssuer, asset.Type.String())

	liabilities := trustEntry.Liabilities()

	closedAt, err := TimePointToUTCTimeStamp(header.Header.ScpValue.CloseTime)
	if err != nil {
		return TrustlineOutput{}, err
	}

	ledgerSequence := header.Header.LedgerSeq

	transformedTrustline := TrustlineOutput{
		LedgerKey:          outputLedgerKey,
		AccountID:          outputAccountID,
		AssetType:          assetType,
		AssetCode:          outputAssetCode,
		AssetIssuer:        outputAssetIssuer,
		AssetID:            outputAssetID,
		Balance:            ConvertStroopValueToReal(trustEntry.Balance),
		TrustlineLimit:     int64(trustEntry.Limit),
		LiquidityPoolID:    poolID,
		BuyingLiabilities:  ConvertStroopValueToReal(liabilities.Buying),
		SellingLiabilities: ConvertStroopValueToReal(liabilities.Selling),
		Flags:              uint32(trustEntry.Flags),
		LastModifiedLedger: uint32(ledgerEntry.LastModifiedLedgerSeq),
		LedgerEntryChange:  uint32(changeType),
		Sponsor:            ledgerEntrySponsorToNullString(ledgerEntry),
		Deleted:            outputDeleted,
		ClosedAt:           closedAt,
		LedgerSequence:     uint32(ledgerSequence),
	}

	return transformedTrustline, nil
}

func trustLineEntryToLedgerKeyString(trustLine xdr.TrustLineEntry) (string, error) {
	ledgerKey := &xdr.LedgerKey{}
	err := ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
	if err != nil {
		return "", fmt.Errorf("error running ledgerKey.SetTrustline when calculating ledger key")
	}

	key, err := ledgerKey.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("error running MarshalBinaryCompress when calculating ledger key")
	}

	return base64.StdEncoding.EncodeToString(key), nil

}

func ledgerEntrySponsorToNullString(entry xdr.LedgerEntry) null.String {
	sponsoringID := entry.SponsoringID()

	var sponsor null.String
	if sponsoringID != nil {
		sponsor.SetValid((*sponsoringID).Address())
	}

	return sponsor
}
