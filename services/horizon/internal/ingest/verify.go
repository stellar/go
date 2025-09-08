package ingest

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/guregu/null"

	ingestsdk "github.com/stellar/go/ingest"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/services/horizon/internal/verify"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const assetStatsBatchSize = 500
const verifyBatchSize = 50000

func checkAssetStats(
	ctx context.Context,
	set processors.AssetStatSet,
	contractAssetStatSet *processors.ContractAssetStatSet,
	q history.IngestionQ,
) error {
	all, balances, err := extractAssetStatsAndBalances(set, contractAssetStatSet)
	if err != nil {
		return err
	}

	page := db2.PageQuery{
		Order: "asc",
		Limit: assetStatsBatchSize,
	}
	for {
		assetStats, err := q.GetAssetStats(ctx, "", "", page)
		if err != nil {
			return errors.Wrap(err, "could not fetch asset stats from db")
		}
		if len(assetStats) == 0 {
			break
		}

		for _, assetStat := range assetStats {
			key := assetStat.AssetCode + ":" + assetStat.AssetIssuer
			fromSet, ok := all[key]
			if !ok {
				return ingestsdk.NewStateError(
					fmt.Errorf(
						"db contains asset stat with code %s issuer %s which is missing from HAS",
						assetStat.AssetCode, assetStat.AssetIssuer,
					),
				)
			}
			delete(all, key)

			if !fromSet.Equals(assetStat) {
				return ingestsdk.NewStateError(
					fmt.Errorf(
						"db asset stat with code %s issuer %s does not match asset stat from HAS: expected=%v actual=%v",
						assetStat.AssetCode, assetStat.AssetIssuer, fromSet, assetStat,
					),
				)
			}
		}

		page.Cursor = assetStats[len(assetStats)-1].PagingToken()
	}

	if len(all) > 0 {
		return ingestsdk.NewStateError(
			fmt.Errorf(
				"HAS contains %d more asset stats than db",
				len(all),
			),
		)
	}

	if err := checkContractBalances(ctx, balances, q); err != nil {
		return err
	}
	return nil
}

func extractAssetStatsAndBalances(set processors.AssetStatSet, contractAssetStatSet *processors.ContractAssetStatSet) (map[string]history.AssetAndContractStat, []history.ContractAssetBalance, error) {
	all := map[string]history.AssetAndContractStat{}
	for _, assetStat := range set.All() {
		// no need to handle the native asset because asset stats only
		// include non-native assets.
		all[assetStat.AssetCode+":"+assetStat.AssetIssuer] = history.AssetAndContractStat{
			ExpAssetStat: assetStat,
			Contracts: history.ContractStat{
				ActiveBalance: "0",
				ActiveHolders: 0,
			},
		}
	}

	contractToStats := map[xdr.ContractId]history.ContractAssetStatRow{}
	for _, row := range contractAssetStatSet.GetContractStats() {
		var contractID xdr.ContractId
		copy(contractID[:], row.ContractID)
		contractToStats[contractID] = row
	}

	assetContracts, err := contractAssetStatSet.GetCreatedAssetContracts()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error getting created asset contracts")
	}
	for _, assetContract := range assetContracts {
		key := assetContract.AssetCode + ":" + assetContract.AssetIssuer
		entry, ok := all[key]
		if !ok {
			assetType := xdr.AssetTypeAssetTypeCreditAlphanum4
			if len(assetContract.AssetCode) > 4 {
				assetType = xdr.AssetTypeAssetTypeCreditAlphanum12
			}
			entry = history.AssetAndContractStat{
				ExpAssetStat: history.ExpAssetStat{
					AssetType:   assetType,
					AssetCode:   assetContract.AssetCode,
					AssetIssuer: assetContract.AssetIssuer,
					Accounts:    history.ExpAssetStatAccounts{},
					Balances: history.ExpAssetStatBalances{
						Authorized:                      "0",
						AuthorizedToMaintainLiabilities: "0",
						ClaimableBalances:               "0",
						LiquidityPools:                  "0",
						Unauthorized:                    "0",
					},
				},
			}
		}
		contractID := assetContract.ContractID
		entry.ContractID = &contractID
		var contractIDHash xdr.ContractId
		copy(contractIDHash[:], assetContract.ContractID)
		contractStats, ok := contractToStats[contractIDHash]
		if !ok {
			entry.Contracts = history.ContractStat{
				ActiveBalance: "0",
				ActiveHolders: 0,
			}
		} else {
			entry.Contracts = contractStats.Stat
		}
		all[key] = entry
	}

	// only check contract asset balances which belong to stellar asset contracts
	// because other balances may be forged.
	var filteredBalances []history.ContractAssetBalance
	for _, balance := range contractAssetStatSet.GetCreatedBalances() {
		var contractID xdr.ContractId
		copy(contractID[:], balance.ContractID)
		if _, ok := contractToStats[contractID]; ok {
			filteredBalances = append(filteredBalances, balance)
		}
	}
	return all, filteredBalances, nil
}

func checkContractBalances(
	ctx context.Context,
	balances []history.ContractAssetBalance,
	q history.IngestionQ,
) error {
	for i := 0; i < len(balances); {
		end := i + assetStatsBatchSize
		if end > len(balances) {
			end = len(balances)
		}

		subset := balances[i:end]
		var keys []xdr.Hash
		byKey := map[xdr.Hash]history.ContractAssetBalance{}
		for _, balance := range subset {
			var key xdr.Hash
			copy(key[:], balance.KeyHash)
			keys = append(keys, key)
			byKey[key] = balance
		}

		rows, err := q.GetContractAssetBalances(ctx, keys)
		if err != nil {
			return err
		}

		for _, row := range rows {
			var key xdr.Hash
			copy(key[:], row.KeyHash)
			expected := byKey[key]

			if !bytes.Equal(row.ContractID, expected.ContractID) {
				return ingestsdk.NewStateError(
					fmt.Errorf(
						"contract balance %v has contract %v in HAS but is %v in db",
						key,
						expected.ContractID,
						row.ContractID,
					),
				)
			}

			if row.ExpirationLedger != expected.ExpirationLedger {
				return ingestsdk.NewStateError(
					fmt.Errorf(
						"contract balance %v has expiration %v in HAS but is %v in db",
						key,
						expected.ExpirationLedger,
						row.ExpirationLedger,
					),
				)
			}

			if row.Amount != expected.Amount {
				return ingestsdk.NewStateError(
					fmt.Errorf(
						"contract balance %v has amount %v in HAS but is %v in db",
						key,
						expected.Amount,
						row.Amount,
					),
				)
			}
		}

		i = end
	}
	return nil
}

func addAccountsToStateVerifier(ctx context.Context, verifier *verify.StateVerifier, q history.IngestionQ, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	accounts, err := q.GetAccountsByIDs(ctx, ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetAccountsByIDs")
	}

	signers, err := q.SignersForAccounts(ctx, ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.SignersForAccounts")
	}

	masterWeightMap := make(map[string]int32)
	signersMap := make(map[string][]xdr.Signer)
	// map[accountID]map[signerKey]sponsor
	sponsoringSignersMap := make(map[string]map[string]string)
	for _, row := range signers {
		if row.Account == row.Signer {
			masterWeightMap[row.Account] = row.Weight
		} else {
			signersMap[row.Account] = append(
				signersMap[row.Account],
				xdr.Signer{
					Key:    xdr.MustSigner(row.Signer),
					Weight: xdr.Uint32(row.Weight),
				},
			)
			if sponsoringSignersMap[row.Account] == nil {
				sponsoringSignersMap[row.Account] = make(map[string]string)
			}
			sponsoringSignersMap[row.Account][row.Signer] = row.Sponsor.String
		}
	}

	for _, row := range accounts {
		var inflationDest *xdr.AccountId
		if row.InflationDestination != "" {
			t := xdr.MustAddress(row.InflationDestination)
			inflationDest = &t
		}

		// Ensure master weight matches, if not it's a state error!
		if int32(row.MasterWeight) != masterWeightMap[row.AccountID] {
			return ingestsdk.NewStateError(
				fmt.Errorf(
					"Master key weight in account %s does not match (expected=%d, actual=%d)",
					row.AccountID,
					masterWeightMap[row.AccountID],
					int32(row.MasterWeight),
				),
			)
		}

		signers := xdr.SortSignersByKey(signersMap[row.AccountID])
		signerSponsoringIDs := make([]xdr.SponsorshipDescriptor, len(signers))
		for i, signer := range signers {
			sponsor := sponsoringSignersMap[row.AccountID][signer.Key.Address()]
			if sponsor != "" {
				signerSponsoringIDs[i] = xdr.MustAddressPtr(sponsor)
			}
		}

		// Accounts that haven't done anything since Protocol 19 will not have a
		// V3 extension, so we need to check whether or not this extension needs
		// to be filled out.
		v3extension := xdr.AccountEntryExtensionV2Ext{V: 0}
		if row.SequenceLedger.Valid && row.SequenceTime.Valid {
			v3extension.V = 3
			v3extension.V3 = &xdr.AccountEntryExtensionV3{
				SeqLedger: xdr.Uint32(row.SequenceLedger.Int64),
				SeqTime:   xdr.TimePoint(row.SequenceTime.Int64),
			}
		}

		account := &xdr.AccountEntry{
			AccountId:     xdr.MustAddress(row.AccountID),
			Balance:       xdr.Int64(row.Balance),
			SeqNum:        xdr.SequenceNumber(row.SequenceNumber),
			NumSubEntries: xdr.Uint32(row.NumSubEntries),
			InflationDest: inflationDest,
			Flags:         xdr.Uint32(row.Flags),
			HomeDomain:    xdr.String32(row.HomeDomain),
			Thresholds: xdr.Thresholds{
				row.MasterWeight,
				row.ThresholdLow,
				row.ThresholdMedium,
				row.ThresholdHigh,
			},
			Signers: signers,
			Ext: xdr.AccountEntryExt{
				V: 1,
				V1: &xdr.AccountEntryExtensionV1{
					Liabilities: xdr.Liabilities{
						Buying:  xdr.Int64(row.BuyingLiabilities),
						Selling: xdr.Int64(row.SellingLiabilities),
					},
					Ext: xdr.AccountEntryExtensionV1Ext{
						V: 2,
						V2: &xdr.AccountEntryExtensionV2{
							NumSponsored:        xdr.Uint32(row.NumSponsored),
							NumSponsoring:       xdr.Uint32(row.NumSponsoring),
							SignerSponsoringIDs: signerSponsoringIDs,
							Ext:                 v3extension,
						},
					},
				},
			},
		}

		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: account,
			},
		}
		addLedgerEntrySponsor(&entry, row.Sponsor)
		err = verifier.Write(entry)
		if err != nil {
			return err
		}
	}

	return nil
}

func addDataToStateVerifier(ctx context.Context, verifier *verify.StateVerifier, q history.IngestionQ, lkeys []xdr.LedgerKeyData) error {
	if len(lkeys) == 0 {
		return nil
	}
	var keys []history.AccountDataKey
	for _, k := range lkeys {
		keys = append(keys, history.AccountDataKey{
			AccountID: k.AccountId.Address(),
			DataName:  string(k.DataName),
		})
	}
	data, err := q.GetAccountDataByKeys(ctx, keys)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetAccountDataByKeys")
	}

	for _, row := range data {
		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &xdr.DataEntry{
					AccountId: xdr.MustAddress(row.AccountID),
					DataName:  xdr.String64(row.Name),
					DataValue: xdr.DataValue(row.Value),
				},
			},
		}
		addLedgerEntrySponsor(&entry, row.Sponsor)
		err := verifier.Write(entry)
		if err != nil {
			return err
		}
	}

	return nil
}

func addOffersToStateVerifier(
	ctx context.Context,
	verifier *verify.StateVerifier,
	q history.IngestionQ,
	ids []int64,
) error {
	if len(ids) == 0 {
		return nil
	}

	offers, err := q.GetOffersByIDs(ctx, ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetOfferByIDs")
	}

	for _, row := range offers {
		offerXDR := offerToXDR(row)
		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type:  xdr.LedgerEntryTypeOffer,
				Offer: &offerXDR,
			},
		}
		addLedgerEntrySponsor(&entry, row.Sponsor)
		err := verifier.Write(entry)
		if err != nil {
			return err
		}
	}

	return nil
}

func offerToXDR(row history.Offer) xdr.OfferEntry {
	return xdr.OfferEntry{
		SellerId: xdr.MustAddress(row.SellerID),
		OfferId:  xdr.Int64(row.OfferID),
		Selling:  row.SellingAsset,
		Buying:   row.BuyingAsset,
		Amount:   xdr.Int64(row.Amount),
		Price: xdr.Price{
			N: xdr.Int32(row.Pricen),
			D: xdr.Int32(row.Priced),
		},
		Flags: xdr.Uint32(row.Flags),
	}
}

func addTrustLinesToStateVerifier(
	ctx context.Context,
	verifier *verify.StateVerifier,
	assetStats processors.AssetStatSet,
	q history.IngestionQ,
	keys []xdr.LedgerKeyTrustLine,
) error {
	if len(keys) == 0 {
		return nil
	}

	var ledgerKeyStrings []string
	for _, key := range keys {
		var ledgerKey xdr.LedgerKey
		if err := ledgerKey.SetTrustline(key.AccountId, key.Asset); err != nil {
			return errors.Wrap(err, "Error running ledgerKey.SetTrustline")
		}
		b64, err := ledgerKey.MarshalBinaryBase64()
		if err != nil {
			return errors.Wrap(err, "Error running ledgerKey.MarshalBinaryBase64")
		}
		ledgerKeyStrings = append(ledgerKeyStrings, b64)
	}

	trustLines, err := q.GetTrustLinesByKeys(ctx, ledgerKeyStrings)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetTrustLinesByKeys")
	}

	for _, row := range trustLines {
		var entry xdr.LedgerEntry
		entry, err = trustLineToXDR(row)
		if err != nil {
			return err
		}

		if err = verifier.Write(entry); err != nil {
			return err
		}
		if err = assetStats.AddTrustline(
			ingestsdk.Change{
				Post: &entry,
			},
		); err != nil {
			return ingestsdk.NewStateError(
				errors.Wrap(err, "could not add trustline to asset stats"),
			)
		}
	}

	return nil
}

func trustLineToXDR(row history.TrustLine) (xdr.LedgerEntry, error) {
	var asset xdr.TrustLineAsset
	switch row.AssetType {
	case xdr.AssetTypeAssetTypePoolShare:
		asset = xdr.TrustLineAsset{
			Type:            xdr.AssetTypeAssetTypePoolShare,
			LiquidityPoolId: &xdr.PoolId{},
		}
		_, err := hex.Decode((*asset.LiquidityPoolId)[:], []byte(row.LiquidityPoolID))
		if err != nil {
			return xdr.LedgerEntry{}, errors.Wrap(err, "Error decoding liquidity pool id")
		}
	case xdr.AssetTypeAssetTypeNative:
		asset = xdr.MustNewNativeAsset().ToTrustLineAsset()
	default:
		creditAsset, err := xdr.NewCreditAsset(row.AssetCode, row.AssetIssuer)
		if err != nil {
			return xdr.LedgerEntry{}, errors.Wrap(err, "Error decoding credit asset")
		}
		asset = creditAsset.ToTrustLineAsset()
	}

	trustline := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress(row.AccountID),
		Asset:     asset,
		Balance:   xdr.Int64(row.Balance),
		Limit:     xdr.Int64(row.Limit),
		Flags:     xdr.Uint32(row.Flags),
		Ext: xdr.TrustLineEntryExt{
			V: 1,
			V1: &xdr.TrustLineEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  xdr.Int64(row.BuyingLiabilities),
					Selling: xdr.Int64(row.SellingLiabilities),
				},
			},
		},
	}
	entry := xdr.LedgerEntry{
		LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
		Data: xdr.LedgerEntryData{
			Type:      xdr.LedgerEntryTypeTrustline,
			TrustLine: &trustline,
		},
	}
	addLedgerEntrySponsor(&entry, row.Sponsor)
	return entry, nil
}

func addClaimableBalanceToStateVerifier(
	ctx context.Context,
	verifier *verify.StateVerifier,
	assetStats processors.AssetStatSet,
	q history.IngestionQ,
	ids []xdr.ClaimableBalanceId,
) error {
	if len(ids) == 0 {
		return nil
	}

	var idStrings []string
	e := xdr.NewEncodingBuffer()
	for _, id := range ids {
		idString, err := e.MarshalHex(id)
		if err != nil {
			return err
		}
		idStrings = append(idStrings, idString)
	}
	cBalances, err := q.GetClaimableBalancesByID(ctx, idStrings)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetClaimableBalancesByID")
	}

	cBalancesClaimants, err := q.GetClaimantsByClaimableBalances(ctx, idStrings)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetClaimantsByClaimableBalances")
	}

	for _, row := range cBalances {
		claimants := []xdr.Claimant{}
		for _, claimant := range row.Claimants {
			claimants = append(claimants, xdr.Claimant{
				Type: xdr.ClaimantTypeClaimantTypeV0,
				V0: &xdr.ClaimantV0{
					Destination: xdr.MustAddress(claimant.Destination),
					Predicate:   claimant.Predicate,
				},
			})
		}
		claimants = xdr.SortClaimantsByDestination(claimants)

		// Check if balances in claimable_balance_claimants table match.
		if len(claimants) != len(cBalancesClaimants[row.BalanceID]) {
			return ingestsdk.NewStateError(
				fmt.Errorf(
					"claimable_balance_claimants length (%d) for claimants doesn't match claimable_balance table (%d)",
					len(cBalancesClaimants[row.BalanceID]), len(claimants),
				),
			)
		}

		for i, claimant := range claimants {
			if claimant.MustV0().Destination.Address() != cBalancesClaimants[row.BalanceID][i].Destination ||
				row.LastModifiedLedger != cBalancesClaimants[row.BalanceID][i].LastModifiedLedger {
				return fmt.Errorf(
					"claimable_balance_claimants table for balance %s does not match. expectedDestination=%s actualDestination=%s, expectedLastModifiedLedger=%d actualLastModifiedLedger=%d",
					row.BalanceID,
					claimant.MustV0().Destination.Address(),
					cBalancesClaimants[row.BalanceID][i].Destination,
					row.LastModifiedLedger,
					cBalancesClaimants[row.BalanceID][i].LastModifiedLedger,
				)
			}
		}

		var balanceID xdr.ClaimableBalanceId
		if err := xdr.SafeUnmarshalHex(row.BalanceID, &balanceID); err != nil {
			return err
		}
		cBalance := xdr.ClaimableBalanceEntry{
			BalanceId: balanceID,
			Claimants: claimants,
			Asset:     row.Asset,
			Amount:    row.Amount,
		}
		if row.Flags != 0 {
			cBalance.Ext = xdr.ClaimableBalanceEntryExt{
				V: 1,
				V1: &xdr.ClaimableBalanceEntryExtensionV1{
					Flags: xdr.Uint32(row.Flags),
				},
			}
		}
		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &cBalance,
			},
		}
		addLedgerEntrySponsor(&entry, row.Sponsor)
		if err := verifier.Write(entry); err != nil {
			return err
		}

		if err := assetStats.AddClaimableBalance(
			ingestsdk.Change{
				Post: &entry,
			},
		); err != nil {
			return ingestsdk.NewStateError(
				errors.Wrap(err, "could not add claimable balance to asset stats"),
			)
		}
	}

	return nil
}

func addLiquidityPoolsToStateVerifier(
	ctx context.Context,
	verifier *verify.StateVerifier,
	assetStats processors.AssetStatSet,
	q history.IngestionQ,
	ids []xdr.PoolId,
) error {
	if len(ids) == 0 {
		return nil
	}
	var idsHex = make([]string, len(ids))
	for i, id := range ids {
		idsHex[i] = processors.PoolIDToString(id)

	}
	lPools, err := q.GetLiquidityPoolsByID(ctx, idsHex)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetLiquidityPoolsByID")
	}

	for _, row := range lPools {
		lPoolEntry, err := liquidityPoolToXDR(row)
		if err != nil {
			return errors.Wrap(err, "Invalid liquidity pool row")
		}

		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type:          xdr.LedgerEntryTypeLiquidityPool,
				LiquidityPool: &lPoolEntry,
			},
		}
		if err := verifier.Write(entry); err != nil {
			return err
		}

		if err := assetStats.AddLiquidityPool(
			ingestsdk.Change{
				Post: &entry,
			},
		); err != nil {
			return ingestsdk.NewStateError(
				errors.Wrap(err, "could not add claimable balance to asset stats"),
			)
		}
	}

	return nil
}

func liquidityPoolToXDR(row history.LiquidityPool) (xdr.LiquidityPoolEntry, error) {
	if len(row.AssetReserves) != 2 {
		return xdr.LiquidityPoolEntry{}, fmt.Errorf("unexpected number of asset reserves (%d), expected %d", len(row.AssetReserves), 2)
	}
	id, err := hex.DecodeString(row.PoolID)
	if err != nil {
		return xdr.LiquidityPoolEntry{}, errors.Wrap(err, "Error decoding pool ID")
	}
	var poolID xdr.PoolId
	if len(id) != len(poolID) {
		return xdr.LiquidityPoolEntry{}, fmt.Errorf("Error decoding pool ID, incorrect length (%d)", len(id))
	}
	copy(poolID[:], id)

	var lPoolEntry = xdr.LiquidityPoolEntry{
		LiquidityPoolId: poolID,
		Body: xdr.LiquidityPoolEntryBody{
			Type: row.Type,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: row.AssetReserves[0].Asset,
					AssetB: row.AssetReserves[1].Asset,
					Fee:    xdr.Int32(row.Fee),
				},
				ReserveA:                 xdr.Int64(row.AssetReserves[0].Reserve),
				ReserveB:                 xdr.Int64(row.AssetReserves[1].Reserve),
				TotalPoolShares:          xdr.Int64(row.ShareCount),
				PoolSharesTrustLineCount: xdr.Int64(row.TrustlineCount),
			},
		},
	}
	return lPoolEntry, nil
}

func addLedgerEntrySponsor(entry *xdr.LedgerEntry, sponsor null.String) {
	ledgerEntrySponsor := xdr.SponsorshipDescriptor(nil)

	if !sponsor.IsZero() {
		ledgerEntrySponsor = xdr.MustAddressPtr(sponsor.String)
	}
	entry.Ext = xdr.LedgerEntryExt{
		V: 1,
		V1: &xdr.LedgerEntryExtensionV1{
			SponsoringId: ledgerEntrySponsor,
		},
	}
}
