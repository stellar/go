package processors

import (
	"context"
	"database/sql"
	"fmt"
	stdio "io"
	"math/big"

	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

var log = logpkg.DefaultLogger.WithField("service", "expingest")

const maxBatchSize = 100000

func (p *DatabaseProcessor) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer r.Close()
	defer w.Close()

	var (
		accountsBatch      history.AccountsBatchInsertBuilder
		accountDataBatch   history.AccountDataBatchInsertBuilder
		accountSignerBatch history.AccountSignersBatchInsertBuilder
		offersBatch        history.OffersBatchInsertBuilder
		trustLinesBatch    history.TrustLinesBatchInsertBuilder
	)
	assetStats := AssetStatSet{}

	switch p.Action {
	case Accounts:
		accountsBatch = p.AccountsQ.NewAccountsBatchInsertBuilder(maxBatchSize)
	case Data:
		accountDataBatch = p.DataQ.NewAccountDataBatchInsertBuilder(maxBatchSize)
	case AccountsForSigner:
		accountSignerBatch = p.SignersQ.NewAccountSignersBatchInsertBuilder(maxBatchSize)
	case Offers:
		offersBatch = p.OffersQ.NewOffersBatchInsertBuilder(maxBatchSize)
	case TrustLines:
		trustLinesBatch = p.TrustLinesQ.NewTrustLinesBatchInsertBuilder(maxBatchSize)
	default:
		return errors.Errorf("Invalid action type (%s)", p.Action)
	}

	for {
		entryChange, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if entryChange.Type != xdr.LedgerEntryChangeTypeLedgerEntryState {
			return errors.New("DatabaseProcessor requires LedgerEntryChangeTypeLedgerEntryState changes only")
		}

		switch p.Action {
		case Accounts:
			// We're interested in accounts only
			if entryChange.EntryType() != xdr.LedgerEntryTypeAccount {
				continue
			}

			err = accountsBatch.Add(
				entryChange.MustState().Data.MustAccount(),
				entryChange.MustState().LastModifiedLedgerSeq,
			)
			if err != nil {
				return errors.Wrap(err, "Error adding row to accountSignerBatch")
			}
		case Data:
			// We're interested in data only
			if entryChange.EntryType() != xdr.LedgerEntryTypeData {
				continue
			}

			err = accountDataBatch.Add(
				entryChange.MustState().Data.MustData(),
				entryChange.MustState().LastModifiedLedgerSeq,
			)
			if err != nil {
				return errors.Wrap(err, "Error adding row to accountSignerBatch")
			}
		case AccountsForSigner:
			// We're interested in accounts only
			if entryChange.EntryType() != xdr.LedgerEntryTypeAccount {
				continue
			}

			accountEntry := entryChange.MustState().Data.MustAccount()
			account := accountEntry.AccountId.Address()
			for signer, weight := range accountEntry.SignerSummary() {
				err = accountSignerBatch.Add(history.AccountSigner{
					Account: account,
					Signer:  signer,
					Weight:  weight,
				})
				if err != nil {
					return errors.Wrap(err, "Error adding row to accountSignerBatch")
				}
			}
		case Offers:
			// We're interested in offers only
			if entryChange.EntryType() != xdr.LedgerEntryTypeOffer {
				continue
			}

			err = offersBatch.Add(
				entryChange.MustState().Data.MustOffer(),
				entryChange.MustState().LastModifiedLedgerSeq,
			)
			if err != nil {
				return errors.Wrap(err, "Error adding row to offersBatch")
			}
		case TrustLines:
			// We're interested in trust lines only
			if entryChange.EntryType() != xdr.LedgerEntryTypeTrustline {
				continue
			}

			trustline := entryChange.MustState().Data.MustTrustLine()
			err = assetStats.Add(trustline)
			if err != nil {
				return errors.Wrap(err, "Error adding trustline to asset stats set")
			}

			err = trustLinesBatch.Add(
				trustline,
				entryChange.MustState().LastModifiedLedgerSeq,
			)
			if err != nil {
				return errors.Wrap(err, "Error adding row to trustLinesBatch")
			}
		default:
			return errors.New("Unknown action")
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	var err error

	switch p.Action {
	case Accounts:
		err = accountsBatch.Exec()
	case Data:
		err = accountDataBatch.Exec()
	case AccountsForSigner:
		err = accountSignerBatch.Exec()
	case Offers:
		err = offersBatch.Exec()
	case TrustLines:
		err = trustLinesBatch.Exec()
		if err == nil {
			err = p.AssetStatsQ.InsertAssetStats(assetStats.All(), maxBatchSize)
		}
	default:
		return errors.Errorf("Invalid action type (%s)", p.Action)
	}

	if err != nil {
		return errors.Wrap(err, "Error batch inserting rows")
	}

	return nil
}

func (p *DatabaseProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer w.Close()

	// Exit early if not ingesting state (history catchup). The filtering in parent
	// processor should do it, unfortunately it won't work in case of meta upgrades.
	// Should be fixed after ingest refactoring.
	if v := ctx.Value(IngestUpdateState); !(v != nil && v.(bool)) {
		return nil
	}

	ledgerCache := io.NewLedgerEntryChangeCache()
	p.AssetStatSet = AssetStatSet{}
	p.batchUpsertTrustLines = []xdr.LedgerEntry{}
	p.batchUpsertAccounts = []xdr.LedgerEntry{}

	actionHandlers := map[DatabaseProcessorActionType]func(change io.Change) error{
		Accounts:          p.processLedgerAccounts,
		AccountsForSigner: p.processLedgerAccountSigners,
		Data:              p.processLedgerAccountData,
		Offers:            p.processLedgerOffers,
		TrustLines:        p.processLedgerTrustLines,
	}

	actions := []DatabaseProcessorActionType{}

	if p.Action == All {
		actions = []DatabaseProcessorActionType{
			Accounts, AccountsForSigner, Data, Offers, TrustLines,
		}
	} else {
		actions = append(actions, p.Action)
	}

	var successTxCount, failedTxCount, opCount int

	// Get all transactions
	var transactions []io.LedgerTransaction
	for {
		var transaction io.LedgerTransaction
		transaction, err = r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if transaction.Successful() {
			successTxCount++
			opCount += len(transaction.Envelope.Tx.Operations)
		} else {
			failedTxCount++
		}

		transactions = append(transactions, transaction)
	}

	// Remember that it's possible that transaction can remove a preauth
	// tx signer even when it's a failed transaction so we need to check
	// failed transactions too.

	// Fees are processed before everything else.
	for _, transaction := range transactions {
		for _, change := range transaction.GetFeeChanges() {
			err = ledgerCache.AddChange(change)
			if err != nil {
				return errors.Wrap(err, "error adding to ledgerCache")
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	// Tx meta
	for _, transaction := range transactions {
		var changes []io.Change
		changes, err = transaction.GetChanges()
		if err != nil {
			return errors.Wrap(err, "Error in transaction.GetChanges()")
		}
		for _, change := range changes {
			err = ledgerCache.AddChange(change)
			if err != nil {
				return errors.Wrap(err, "error adding to ledgerCache")
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	// Process upgrades meta
	for {
		var change io.Change
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		err = ledgerCache.AddChange(change)
		if err != nil {
			return errors.Wrap(err, "error addint to ledgerCache")
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	changes := ledgerCache.GetChanges()
	for _, action := range actions {
		handler, ok := actionHandlers[action]
		if !ok {
			return errors.New("Unknown action")
		}

		for _, change := range changes {
			err = handler(change)
			if err != nil {
				return errors.Wrap(
					err,
					fmt.Sprintf("Error in %s handler", action),
				)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	if p.Action == All || p.Action == Accounts {
		// Upsert accounts
		if len(p.batchUpsertAccounts) > 0 {
			err = p.AccountsQ.UpsertAccounts(p.batchUpsertAccounts)
			if err != nil {
				return errors.Wrap(err, "errors in UpsertAccounts")
			}
		}
	}

	// Asset stats
	if p.Action == All || p.Action == TrustLines {
		// Upsert trust lines
		if len(p.batchUpsertTrustLines) > 0 {
			err = p.TrustLinesQ.UpsertTrustLines(p.batchUpsertTrustLines)
			if err != nil {
				return errors.Wrap(err, "errors in UpsertTrustLines")
			}
		}

		assetStatsDeltas := p.AssetStatSet.All()
		for _, delta := range assetStatsDeltas {
			var rowsAffected int64

			stat, err := p.AssetStatsQ.GetAssetStat(
				delta.AssetType,
				delta.AssetCode,
				delta.AssetIssuer,
			)
			assetStatNotFound := err == sql.ErrNoRows
			if !assetStatNotFound && err != nil {
				return errors.Wrap(err, "could not fetch asset stat from db")
			}

			if assetStatNotFound {
				// Insert
				if delta.NumAccounts < 0 {
					return ingesterrors.NewStateError(errors.Errorf(
						"NumAccounts negative but DB entry does not exist for asset: %s %s %s",
						delta.AssetType,
						delta.AssetCode,
						delta.AssetIssuer,
					))
				}

				var errInsert error
				rowsAffected, errInsert = p.AssetStatsQ.InsertAssetStat(delta)
				if errInsert != nil {
					return errors.Wrap(errInsert, "could not insert asset stat")
				}
			} else {
				statBalance, ok := new(big.Int).SetString(stat.Amount, 10)
				if !ok {
					return errors.New("Error parsing: " + stat.Amount)
				}

				deltaBalance, ok := new(big.Int).SetString(delta.Amount, 10)
				if !ok {
					return errors.New("Error parsing: " + stat.Amount)
				}

				// statBalance = statBalance + deltaBalance
				statBalance.Add(statBalance, deltaBalance)
				statAccounts := stat.NumAccounts + delta.NumAccounts

				if statAccounts == 0 {
					// Remove stats
					if statBalance.Cmp(big.NewInt(0)) != 0 {
						return ingesterrors.NewStateError(errors.Errorf(
							"Removing asset stat by final amount non-zero for: %s %s %s",
							delta.AssetType,
							delta.AssetCode,
							delta.AssetIssuer,
						))
					}
					rowsAffected, err = p.AssetStatsQ.RemoveAssetStat(
						delta.AssetType,
						delta.AssetCode,
						delta.AssetIssuer,
					)
					if err != nil {
						return errors.Wrap(err, "could not remove asset stat")
					}
				} else {
					// Update
					rowsAffected, err = p.AssetStatsQ.UpdateAssetStat(history.ExpAssetStat{
						AssetType:   delta.AssetType,
						AssetCode:   delta.AssetCode,
						AssetIssuer: delta.AssetIssuer,
						Amount:      statBalance.String(),
						NumAccounts: statAccounts,
					})
					if err != nil {
						return errors.Wrap(err, "could not update asset stat")
					}
				}
			}

			if rowsAffected != 1 {
				return ingesterrors.NewStateError(errors.Errorf(
					"%d rows affected when adjusting asset stat for asset: %s %s %s",
					rowsAffected,
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			}
		}
	}

	return nil
}

func (p *DatabaseProcessor) processLedgerAccounts(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeAccount {
		return nil
	}

	changed, err := change.AccountChangedExceptSigners()
	if err != nil {
		return errors.Wrap(err, "Error running change.AccountChangedExceptSigners")
	}

	if !changed {
		return nil
	}

	var accountID string

	switch {
	case change.Post != nil:
		// Created and updated
		p.batchUpsertAccounts = append(p.batchUpsertAccounts, *change.Post)
	case change.Pre != nil && change.Post == nil:
		// Removed
		account := change.Pre.Data.MustAccount()
		accountID = account.AccountId.Address()
		rowsAffected, err := p.AccountsQ.RemoveAccount(accountID)

		if err != nil {
			return err
		}

		if rowsAffected != 1 {
			return ingesterrors.NewStateError(errors.Errorf(
				"%d No rows affected when removing account %s",
				rowsAffected,
				accountID,
			))
		}
	default:
		return errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
	}

	return nil
}

func (p *DatabaseProcessor) processLedgerAccountData(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeData {
		return nil
	}

	var rowsAffected int64
	var err error
	var action string
	var ledgerKey xdr.LedgerKey

	switch {
	case change.Pre == nil && change.Post != nil:
		// Created
		action = "inserting"
		data := change.Post.Data.MustData()
		err = ledgerKey.SetData(data.AccountId, string(data.DataName))
		if err != nil {
			return errors.Wrap(err, "Error creating ledger key")
		}
		rowsAffected, err = p.DataQ.InsertAccountData(data, change.Post.LastModifiedLedgerSeq)
	case change.Pre != nil && change.Post == nil:
		// Removed
		action = "removing"
		data := change.Pre.Data.MustData()
		err = ledgerKey.SetData(data.AccountId, string(data.DataName))
		if err != nil {
			return errors.Wrap(err, "Error creating ledger key")
		}
		rowsAffected, err = p.DataQ.RemoveAccountData(*ledgerKey.Data)
	default:
		// Updated
		action = "updating"
		data := change.Post.Data.MustData()
		err = ledgerKey.SetData(data.AccountId, string(data.DataName))
		if err != nil {
			return errors.Wrap(err, "Error creating ledger key")
		}
		rowsAffected, err = p.DataQ.UpdateAccountData(data, change.Post.LastModifiedLedgerSeq)
	}

	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return ingesterrors.NewStateError(errors.Errorf(
			"%d rows affected when %s data: %s %s",
			rowsAffected,
			action,
			ledgerKey.Data.AccountId.Address(),
			ledgerKey.Data.DataName,
		))
	}

	return nil
}

func (p *DatabaseProcessor) processLedgerAccountSigners(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeAccount {
		return nil
	}

	if !change.AccountSignersChanged() {
		return nil
	}

	// The code below removes all Pre signers adds Post signers but
	// can be improved by finding a diff (check performance first).
	if change.Pre != nil {
		preAccountEntry := change.Pre.Data.MustAccount()
		for signer := range preAccountEntry.SignerSummary() {
			rowsAffected, err := p.SignersQ.RemoveAccountSigner(preAccountEntry.AccountId.Address(), signer)
			if err != nil {
				return errors.Wrap(err, "Error removing a signer")
			}

			if rowsAffected != 1 {
				return ingesterrors.NewStateError(errors.Errorf(
					"Expected account=%s signer=%s in database but not found when removing (rows affected = %d)",
					preAccountEntry.AccountId.Address(),
					signer,
					rowsAffected,
				))
			}
		}
	}

	if change.Post != nil {
		postAccountEntry := change.Post.Data.MustAccount()
		for signer, weight := range postAccountEntry.SignerSummary() {
			rowsAffected, err := p.SignersQ.CreateAccountSigner(postAccountEntry.AccountId.Address(), signer, weight)
			if err != nil {
				return errors.Wrap(err, "Error inserting a signer")
			}

			if rowsAffected != 1 {
				return ingesterrors.NewStateError(errors.Errorf(
					"%d rows affected when inserting account=%s signer=%s to database",
					rowsAffected,
					postAccountEntry.AccountId.Address(),
					signer,
				))
			}
		}
	}

	return nil
}

func (p *DatabaseProcessor) processLedgerOffers(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeOffer {
		return nil
	}

	var rowsAffected int64
	var err error
	var action string
	var offerID xdr.Int64

	switch {
	case change.Pre == nil && change.Post != nil:
		// Created
		action = "inserting"
		offer := change.Post.Data.MustOffer()
		offerID = offer.OfferId
		rowsAffected, err = p.OffersQ.InsertOffer(offer, change.Post.LastModifiedLedgerSeq)
	case change.Pre != nil && change.Post == nil:
		// Removed
		action = "removing"
		offer := change.Pre.Data.MustOffer()
		offerID = offer.OfferId
		rowsAffected, err = p.OffersQ.RemoveOffer(offer.OfferId)
	default:
		// Updated
		action = "updating"
		offer := change.Post.Data.MustOffer()
		offerID = offer.OfferId
		rowsAffected, err = p.OffersQ.UpdateOffer(offer, change.Post.LastModifiedLedgerSeq)
	}

	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return ingesterrors.NewStateError(errors.Errorf(
			"%d rows affected when %s offer %d",
			rowsAffected,
			action,
			offerID,
		))
	}
	return nil
}

func (p *DatabaseProcessor) adjustAssetStat(
	preTrustline *xdr.TrustLineEntry,
	postTrustline *xdr.TrustLineEntry,
) error {
	var deltaBalance xdr.Int64
	var deltaAccounts int32
	var trustline xdr.TrustLineEntry

	if preTrustline != nil && postTrustline == nil {
		trustline = *preTrustline
		// removing a trustline
		if xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() {
			deltaAccounts = -1
			deltaBalance = -preTrustline.Balance
		}
	} else if preTrustline == nil && postTrustline != nil {
		trustline = *postTrustline
		// adding a trustline
		if xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			deltaAccounts = 1
			deltaBalance = postTrustline.Balance
		}
	} else if preTrustline != nil && postTrustline != nil {
		trustline = *postTrustline
		// updating a trustline
		if xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() &&
			xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			// trustline remains authorized
			deltaAccounts = 0
			deltaBalance = postTrustline.Balance - preTrustline.Balance
		} else if xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() &&
			!xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			// trustline was authorized and became unauthorized
			deltaAccounts = -1
			deltaBalance = -preTrustline.Balance
		} else if !xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() &&
			xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			// trustline was unauthorized and became authorized
			deltaAccounts = 1
			deltaBalance = postTrustline.Balance
		}
		// else, trustline was unauthorized and remains unauthorized
		// so there is no change to accounts or balances
	} else {
		return ingesterrors.NewStateError(errors.New("both pre and post trustlines cannot be nil"))
	}

	err := p.AssetStatSet.AddDelta(trustline.Asset, int64(deltaBalance), deltaAccounts)
	if err != nil {
		return errors.Wrap(err, "error running AssetStatSet.AddDelta")
	}
	return nil
}

func (p *DatabaseProcessor) processLedgerTrustLines(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeTrustline {
		return nil
	}

	var rowsAffected int64
	var err error
	var action string
	var ledgerKey xdr.LedgerKey

	switch {
	case change.Post != nil:
		// Created and updated
		postTrustLine := change.Post.Data.MustTrustLine()
		p.batchUpsertTrustLines = append(p.batchUpsertTrustLines, *change.Post)
		if change.Pre == nil {
			err = p.adjustAssetStat(nil, &postTrustLine)
		} else {
			preTrustLine := change.Pre.Data.MustTrustLine()
			err = p.adjustAssetStat(&preTrustLine, &postTrustLine)
		}
		if err != nil {
			return errors.Wrap(err, "Error adjusting asset stat")
		}
	case change.Pre != nil && change.Post == nil:
		// Removed
		action = "removing"
		trustLine := change.Pre.Data.MustTrustLine()
		err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
		if err != nil {
			return errors.Wrap(err, "Error creating ledger key")
		}
		err = p.adjustAssetStat(&trustLine, nil)
		if err != nil {
			return errors.Wrap(err, "Error adjusting asset stat")
		}
		rowsAffected, err = p.TrustLinesQ.RemoveTrustLine(*ledgerKey.TrustLine)
		if err != nil {
			return err
		}

		if rowsAffected != 1 {
			return ingesterrors.NewStateError(errors.Errorf(
				"%d rows affected when %s trustline: %s %s",
				rowsAffected,
				action,
				ledgerKey.TrustLine.AccountId.Address(),
				ledgerKey.TrustLine.Asset.String(),
			))
		}
	default:
		return errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
	}

	return nil
}

func (p *DatabaseProcessor) Name() string {
	return fmt.Sprintf("DatabaseProcessor (%s)", p.Action)
}

func (p *DatabaseProcessor) Reset() {}

var _ ingestpipeline.StateProcessor = &DatabaseProcessor{}
var _ ingestpipeline.LedgerProcessor = &DatabaseProcessor{}
