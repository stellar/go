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
	"github.com/stellar/go/xdr"
)

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

	if deltaBalance == 0 && deltaAccounts == 0 {
		return nil
	}

	var assetType xdr.AssetType
	var assetIssuer, assetCode string
	if err := trustline.Asset.Extract(&assetType, &assetCode, &assetIssuer); err != nil {
		return errors.Wrap(err, "could not extract asset info from trustline")
	}

	stat, err := p.AssetStatsQ.GetAssetStat(assetType, assetCode, assetIssuer)
	assetStatNotFound := err == sql.ErrNoRows
	if !assetStatNotFound && err != nil {
		return errors.Wrap(err, "could not fetch asset stat from db")
	}

	currentBalance := big.NewInt(0)
	if assetStatNotFound {
		stat.AssetType = assetType
		stat.AssetCode = assetCode
		stat.AssetIssuer = assetIssuer
	} else {
		_, ok := currentBalance.SetString(stat.Amount, 10)
		if !ok {
			return verify.NewStateError(errors.Errorf(
				"Could not parse asset stat amount %s when processing trustline: %s %s",
				stat.Amount,
				trustline.AccountId.Address(),
				trustline.Asset.String(),
			))
		}
	}
	currentBalance = currentBalance.Add(currentBalance, big.NewInt(int64(deltaBalance)))
	stat.Amount = currentBalance.String()
	stat.NumAccounts += deltaAccounts

	if currentBalance.Cmp(big.NewInt(0)) < 0 {
		return verify.NewStateError(errors.Errorf(
			"Asset stat has negative amount %s when processing trustline: %s %s",
			stat.Amount,
			trustline.AccountId.Address(),
			trustline.Asset.String(),
		))
	}

	if stat.NumAccounts < 0 {
		return verify.NewStateError(errors.Errorf(
			"Asset stat has negative num accounts when processing trustline: %s %s",
			trustline.AccountId.Address(),
			trustline.Asset.String(),
		))
	}

	var rowsAffected int64
	if assetStatNotFound {
		// deltaAccounts is 0 if we are updating an account
		// deltaAccounts is < 0 if we are removing an account
		// therefore if deltaAccounts <= 0 the asset stat must exist in the db
		if deltaAccounts <= 0 {
			return verify.NewStateError(errors.Errorf(
				"Expected asset stat to exist when processing trustline: %s %s",
				trustline.AccountId.Address(),
				trustline.Asset.String(),
			))
		}
		rowsAffected, err = p.AssetStatsQ.InsertAssetStat(stat)
		if err != nil {
			return errors.Wrap(err, "could not insert asset stat")
		}
	} else if stat.NumAccounts == 0 {
		if currentBalance.Cmp(big.NewInt(0)) != 0 {
			return verify.NewStateError(errors.Errorf(
				"Expected asset stat with no accounts to have amount of 0 "+
					"(amount was %s) when processing trustline: %s %s",
				stat.Amount,
				trustline.AccountId.Address(),
				trustline.Asset.String(),
			))
		}
		rowsAffected, err = p.AssetStatsQ.RemoveAssetStat(assetType, assetCode, assetIssuer)
		if err != nil {
			return errors.Wrap(err, "could not update asset stat")
		}
	} else {
		rowsAffected, err = p.AssetStatsQ.UpdateAssetStat(stat)
		if err != nil {
			return errors.Wrap(err, "could not update asset stat")
		}
	}

	if rowsAffected != 1 {
		return verify.NewStateError(errors.Errorf(
			"No rows affected when adjusting asset stat from trustline: %s %s",
			trustline.AccountId.Address(),
			trustline.Asset.String(),
		))
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
