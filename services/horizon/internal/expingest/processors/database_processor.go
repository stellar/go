package processors

import (
	"context"
	"database/sql"
	"fmt"
	stdio "io"
	"math/big"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/verify"
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
	defer func() {
		// io.LedgerReader.Close() returns error if upgrade changes have not
		// been processed so it's worth checking the error.
		closeErr := r.Close()
		// Do not overwrite the previous error
		if err == nil {
			err = closeErr
		}
	}()
	defer w.Close()

	p.AssetStatSet = AssetStatSet{}

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
	} else if p.Action != Ledgers {
		actions = append(actions, p.Action)
	}

	var successTxCount, failedTxCount, opCount int

	// Process transaction meta
	for {
		transaction, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if transaction.Result.Result.Result.Code == xdr.TransactionResultCodeTxSuccess {
			successTxCount++
			opCount += len(transaction.Envelope.Tx.Operations)
		} else {
			failedTxCount++
		}

		for _, action := range actions {
			handler, ok := actionHandlers[action]
			if !ok {
				return errors.New("Unknown action")
			}

			// Remember that it's possible that transaction can remove a preauth
			// tx signer even when it's a failed transaction.

			for _, change := range transaction.GetChanges() {
				err := handler(change)
				if err != nil {
					return errors.Wrap(
						err,
						fmt.Sprintf("Error in %s handler", action),
					)
				}
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
		change, err := r.ReadUpgradeChange()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		for _, action := range actions {
			handler, ok := actionHandlers[action]
			if !ok {
				return errors.New("Unknown action")
			}

			err := handler(change)
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

	// Asset stats
	if p.Action == All || p.Action == TrustLines {
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
					return verify.NewStateError(errors.Errorf(
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
						return verify.NewStateError(errors.Errorf(
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
				return verify.NewStateError(errors.Errorf(
					"No rows affected when adjusting asset stat for asset: %s %s %s",
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			}
		}
	}

	return p.ingestLedgerHeader(ctx, r, successTxCount, failedTxCount, opCount)
}

func (p *DatabaseProcessor) ingestLedgerHeader(
	ctx context.Context, r io.LedgerReader, successTxCount, failedTxCount, opCount int,
) error {
	if p.Action != All && p.Action != Ledgers {
		return nil
	}

	// Exit early if not ingesting into a DB
	if v := ctx.Value(IngestUpdateDatabase); v == nil {
		return nil
	}

	rowsAffected, err := p.LedgersQ.InsertExpLedger(
		r.GetHeader(),
		successTxCount,
		failedTxCount,
		opCount,
		p.IngestVersion,
	)
	if err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf("Could not insert ledger"),
		)
	}
	if rowsAffected != 1 {
		log.WithField("rowsAffected", rowsAffected).
			WithField("sequence", r.GetSequence()).
			Error("No rows affected when ingesting new ledger")
		return errors.Errorf(
			"No rows affected when ingesting new ledger: %v",
			r.GetSequence(),
		)
	}

	// use an older lookup sequence because the experimental ingestion system and the
	// legacy ingestion system might not be in sync
	seq := int32(r.GetSequence() - 10)

	valid, err := p.LedgersQ.CheckExpLedger(seq)
	// only validate the ledger if it is present in both ingestion systems
	if err == sql.ErrNoRows {
		return nil
	}

	if err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf("Could not compare ledger %v", seq),
		)
	}

	if !valid {
		log.WithField("sequence", seq).
			Error("row in exp_history_ledgers does not match ledger in history_ledgers")
		return errors.Errorf(
			"ledger %v in exp_history_ledgers does not match ledger in history_ledgers",
			seq,
		)
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

	var rowsAffected int64
	var action string
	var accountID string

	switch {
	case change.Pre == nil && change.Post != nil:
		// Created
		action = "inserting"
		account := change.Post.Data.MustAccount()
		accountID = account.AccountId.Address()
		rowsAffected, err = p.AccountsQ.InsertAccount(account, change.Post.LastModifiedLedgerSeq)
	case change.Pre != nil && change.Post == nil:
		// Removed
		action = "removing"
		account := change.Pre.Data.MustAccount()
		accountID = account.AccountId.Address()
		rowsAffected, err = p.AccountsQ.RemoveAccount(accountID)
	default:
		// Updated
		action = "updating"
		account := change.Post.Data.MustAccount()
		accountID = account.AccountId.Address()
		rowsAffected, err = p.AccountsQ.UpdateAccount(account, change.Post.LastModifiedLedgerSeq)
	}

	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return verify.NewStateError(errors.Errorf(
			"No rows affected when %s account %s",
			action,
			accountID,
		))
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
		return verify.NewStateError(errors.Errorf(
			"No rows affected when %s data: %s %s",
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
				return verify.NewStateError(errors.Errorf(
					"Expected account=%s signer=%s in database but not found when removing",
					preAccountEntry.AccountId.Address(),
					signer,
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
				return verify.NewStateError(errors.Errorf(
					"No rows affected when inserting account=%s signer=%s to database",
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
		return verify.NewStateError(errors.Errorf(
			"No rows affected when %s offer %d",
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
		return verify.NewStateError(errors.New("both pre and post trustlines cannot be nil"))
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
	case change.Pre == nil && change.Post != nil:
		// Created
		action = "inserting"
		trustLine := change.Post.Data.MustTrustLine()
		err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
		if err != nil {
			return errors.Wrap(err, "Error creating ledger key")
		}
		err = p.adjustAssetStat(nil, &trustLine)
		if err != nil {
			return errors.Wrap(err, "Error adjusting asset stat")
		}
		rowsAffected, err = p.TrustLinesQ.InsertTrustLine(trustLine, change.Post.LastModifiedLedgerSeq)
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
	default:
		// Updated
		action = "updating"
		preTrustLine := change.Pre.Data.MustTrustLine()
		trustLine := change.Post.Data.MustTrustLine()
		err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
		if err != nil {
			return errors.Wrap(err, "Error creating ledger key")
		}
		err = p.adjustAssetStat(&preTrustLine, &trustLine)
		if err != nil {
			return errors.Wrap(err, "Error adjusting asset stat")
		}
		rowsAffected, err = p.TrustLinesQ.UpdateTrustLine(trustLine, change.Post.LastModifiedLedgerSeq)
	}

	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return verify.NewStateError(errors.Errorf(
			"No rows affected when %s trustline: %s %s",
			action,
			ledgerKey.TrustLine.AccountId.Address(),
			ledgerKey.TrustLine.Asset.String(),
		))
	}
	return nil
}

func (p *DatabaseProcessor) Name() string {
	return fmt.Sprintf("DatabaseProcessor (%s)", p.Action)
}

func (p *DatabaseProcessor) Reset() {}

var _ ingestpipeline.StateProcessor = &DatabaseProcessor{}
var _ ingestpipeline.LedgerProcessor = &DatabaseProcessor{}
