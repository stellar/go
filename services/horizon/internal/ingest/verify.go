package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/guregu/null"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/services/horizon/internal/ingest/verify"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const verifyBatchSize = 50000
const assetStatsBatchSize = 500

// stateVerifierExpectedIngestionVersion defines a version of ingestion system
// required by state verifier. This is done to prevent situations where
// ingestion has been updated with new features but state verifier does not
// check them.
// There is a test that checks it, to fix it: update the actual `verifyState`
// method instead of just updating this value!
const stateVerifierExpectedIngestionVersion = 14

// verifyState is called as a go routine from pipeline post hook every 64
// ledgers. It checks if the state is correct. If another go routine is already
// running it exits.
func (s *system) verifyState(verifyAgainstLatestCheckpoint bool) error {
	s.stateVerificationMutex.Lock()
	if s.stateVerificationRunning {
		log.Warn("State verification is already running...")
		s.stateVerificationMutex.Unlock()
		return nil
	}
	s.stateVerificationRunning = true
	s.stateVerificationMutex.Unlock()
	defer func() {
		s.stateVerificationMutex.Lock()
		s.stateVerificationRunning = false
		s.stateVerificationMutex.Unlock()
	}()

	updateMetrics := false

	if stateVerifierExpectedIngestionVersion != CurrentVersion {
		log.Errorf(
			"State verification expected version is %d but actual is: %d",
			stateVerifierExpectedIngestionVersion,
			CurrentVersion,
		)
		return nil
	}

	historyQ := s.historyQ.CloneIngestionQ()
	defer historyQ.Rollback()
	err := historyQ.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return errors.Wrap(err, "Error starting transaction")
	}

	// Ensure the ledger is a checkpoint ledger
	ledgerSequence, err := historyQ.GetLastLedgerIngestNonBlocking()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.GetLastLedgerIngestNonBlocking")
	}

	localLog := log.WithFields(logpkg.F{
		"subservice": "state_verify",
		"ledger":     ledgerSequence,
	})

	if !s.checkpointManager.IsCheckpoint(ledgerSequence) {
		localLog.Info("Current ledger is not a checkpoint ledger. Cancelling...")
		return nil
	}

	localLog.Info("Starting state verification")

	if verifyAgainstLatestCheckpoint {
		retries := 0
		for {
			// Get root HAS to check if we're checking one of the latest ledgers or
			// Horizon is catching up. It doesn't make sense to verify old ledgers as
			// we want to check the latest state.
			var historyLatestSequence uint32
			historyLatestSequence, err = s.historyAdapter.GetLatestLedgerSequence()
			if err != nil {
				return errors.Wrap(err, "Error getting the latest ledger sequence")
			}

			if ledgerSequence < historyLatestSequence {
				localLog.Info("Current ledger is old. Cancelling...")
				return nil
			}

			if ledgerSequence == historyLatestSequence {
				break
			}

			localLog.Info("Waiting for stellar-core to publish HAS...")
			select {
			case <-s.ctx.Done():
				localLog.Info("State verifier shut down...")
				return nil
			case <-time.After(5 * time.Second):
				// Wait for stellar-core to publish HAS
				retries++
				if retries == 12 {
					localLog.Info("Checkpoint not published. Cancelling...")
					return nil
				}
			}
		}
	}

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		if updateMetrics {
			// Don't update metrics if context cancelled.
			if s.ctx.Err() != context.Canceled {
				s.Metrics().StateVerifyDuration.Observe(float64(duration))
			}
		}
		log.WithField("duration", duration).Info("State verification finished")

	}()

	localLog.Info("Creating state reader...")

	stateReader, err := s.historyAdapter.GetState(s.ctx, ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error running GetState")
	}
	defer stateReader.Close()

	verifier := &verify.StateVerifier{
		StateReader: stateReader,
	}

	assetStats := processors.AssetStatSet{}
	total := 0
	for {
		var keys []xdr.LedgerKey
		keys, err = verifier.GetLedgerKeys(verifyBatchSize)
		if err != nil {
			return errors.Wrap(err, "verifier.GetLedgerKeys")
		}

		if len(keys) == 0 {
			break
		}

		accounts := make([]string, 0, verifyBatchSize)
		data := make([]xdr.LedgerKeyData, 0, verifyBatchSize)
		offers := make([]int64, 0, verifyBatchSize)
		trustLines := make([]xdr.LedgerKeyTrustLine, 0, verifyBatchSize)
		cBalances := make([]xdr.ClaimableBalanceId, 0, verifyBatchSize)
		for _, key := range keys {
			switch key.Type {
			case xdr.LedgerEntryTypeAccount:
				accounts = append(accounts, key.Account.AccountId.Address())
			case xdr.LedgerEntryTypeData:
				data = append(data, *key.Data)
			case xdr.LedgerEntryTypeOffer:
				offers = append(offers, int64(key.Offer.OfferId))
			case xdr.LedgerEntryTypeTrustline:
				trustLines = append(trustLines, *key.TrustLine)
			case xdr.LedgerEntryTypeClaimableBalance:
				cBalances = append(cBalances, key.ClaimableBalance.BalanceId)
			default:
				return errors.New("GetLedgerKeys return unexpected type")
			}
		}

		err = addAccountsToStateVerifier(verifier, historyQ, accounts)
		if err != nil {
			return errors.Wrap(err, "addAccountsToStateVerifier failed")
		}

		err = addDataToStateVerifier(verifier, historyQ, data)
		if err != nil {
			return errors.Wrap(err, "addDataToStateVerifier failed")
		}

		err = addOffersToStateVerifier(verifier, historyQ, offers)
		if err != nil {
			return errors.Wrap(err, "addOffersToStateVerifier failed")
		}

		err = addTrustLinesToStateVerifier(verifier, assetStats, historyQ, trustLines)
		if err != nil {
			return errors.Wrap(err, "addTrustLinesToStateVerifier failed")
		}

		err = addClaimableBalanceToStateVerifier(verifier, assetStats, historyQ, cBalances)
		if err != nil {
			return errors.Wrap(err, "addClaimableBalanceToStateVerifier failed")
		}

		total += len(keys)
		localLog.WithField("total", total).Info("Batch added to StateVerifier")
	}

	localLog.WithField("total", total).Info("Finished writing to StateVerifier")

	countAccounts, err := historyQ.CountAccounts()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountAccounts")
	}

	countData, err := historyQ.CountAccountsData()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountData")
	}

	countOffers, err := historyQ.CountOffers()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountOffers")
	}

	countTrustLines, err := historyQ.CountTrustLines()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountTrustLines")
	}

	countClaimableBalances, err := historyQ.CountClaimableBalances()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountClaimableBalances")
	}

	err = verifier.Verify(countAccounts + countData + countOffers + countTrustLines + countClaimableBalances)
	if err != nil {
		return errors.Wrap(err, "verifier.Verify failed")
	}

	err = checkAssetStats(assetStats, historyQ)
	if err != nil {
		return errors.Wrap(err, "checkAssetStats failed")
	}

	localLog.Info("State correct")
	updateMetrics = true
	return nil
}

func checkAssetStats(set processors.AssetStatSet, q history.IngestionQ) error {
	page := db2.PageQuery{
		Order: "asc",
		Limit: assetStatsBatchSize,
	}

	for {
		assetStats, err := q.GetAssetStats("", "", page)
		if err != nil {
			return errors.Wrap(err, "could not fetch asset stats from db")
		}
		if len(assetStats) == 0 {
			break
		}

		for _, assetStat := range assetStats {
			fromSet, removed := set.Remove(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
			if !removed {
				return ingest.NewStateError(
					fmt.Errorf(
						"db contains asset stat with code %s issuer %s which is missing from HAS",
						assetStat.AssetCode, assetStat.AssetIssuer,
					),
				)
			}

			if fromSet != assetStat {
				return ingest.NewStateError(
					fmt.Errorf(
						"db asset stat with code %s issuer %s does not match asset stat from HAS",
						assetStat.AssetCode, assetStat.AssetIssuer,
					),
				)
			}
		}

		page.Cursor = assetStats[len(assetStats)-1].PagingToken()
	}

	if len(set) > 0 {
		return ingest.NewStateError(
			fmt.Errorf(
				"HAS contains %d more asset stats than db",
				len(set),
			),
		)
	}
	return nil
}

func addAccountsToStateVerifier(verifier *verify.StateVerifier, q history.IngestionQ, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	accounts, err := q.GetAccountsByIDs(ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetAccountsByIDs")
	}

	signers, err := q.SignersForAccounts(ids)
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
			return ingest.NewStateError(
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

func addDataToStateVerifier(verifier *verify.StateVerifier, q history.IngestionQ, keys []xdr.LedgerKeyData) error {
	if len(keys) == 0 {
		return nil
	}

	data, err := q.GetAccountDataByKeys(keys)
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
	verifier *verify.StateVerifier,
	q history.IngestionQ,
	ids []int64,
) error {
	if len(ids) == 0 {
		return nil
	}

	offers, err := q.GetOffersByIDs(ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetOfferByIDs")
	}

	for _, row := range offers {
		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
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

func addTrustLinesToStateVerifier(
	verifier *verify.StateVerifier,
	assetStats processors.AssetStatSet,
	q history.IngestionQ,
	keys []xdr.LedgerKeyTrustLine,
) error {
	if len(keys) == 0 {
		return nil
	}

	trustLines, err := q.GetTrustLinesByKeys(keys)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetTrustLinesByKeys")
	}

	for _, row := range trustLines {
		asset := xdr.MustNewCreditAsset(row.AssetCode, row.AssetIssuer)
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
		if err := verifier.Write(entry); err != nil {
			return err
		}
		if err := assetStats.AddTrustline(
			ingest.Change{
				Post: &xdr.LedgerEntry{
					Data: xdr.LedgerEntryData{
						TrustLine: &trustline,
					},
				},
			},
		); err != nil {
			return ingest.NewStateError(
				errors.Wrap(err, "could not add trustline to asset stats"),
			)
		}
	}

	return nil
}

func addClaimableBalanceToStateVerifier(
	verifier *verify.StateVerifier,
	assetStats processors.AssetStatSet,
	q history.IngestionQ,
	ids []xdr.ClaimableBalanceId,
) error {
	if len(ids) == 0 {
		return nil
	}

	cBalances, err := q.GetClaimableBalancesByID(ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetClaimableBalancesByID")
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
		var cBalance = xdr.ClaimableBalanceEntry{
			BalanceId: row.BalanceID,
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
			ingest.Change{
				Post: &xdr.LedgerEntry{
					Data: xdr.LedgerEntryData{
						ClaimableBalance: &cBalance,
					},
				},
			},
		); err != nil {
			return ingest.NewStateError(
				errors.Wrap(err, "could not add claimable balance to asset stats"),
			)
		}
	}

	return nil
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
