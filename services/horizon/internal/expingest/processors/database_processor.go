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
		accountSignerBatch history.AccountSignersBatchInsertBuilder
		offersBatch        history.OffersBatchInsertBuilder
		trustLinesBatch    history.TrustLinesBatchInsertBuilder
	)
	assetStats := AssetStatSet{}

	switch p.Action {
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

func (p *DatabaseProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) error {
	defer r.Close()
	defer w.Close()

	for {
		transaction, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if transaction.Result.Result.Result.Code != xdr.TransactionResultCodeTxSuccess {
			continue
		}

		switch p.Action {
		case AccountsForSigner:
			err := p.processLedgerAccountsForSigner(transaction)
			if err != nil {
				return errors.Wrap(err, "Error in processLedgerAccountsForSigner")
			}
		case Offers:
			err := p.processLedgerOffers(transaction, r.GetSequence())
			if err != nil {
				return errors.Wrap(err, "Error in processLedgerOffers")
			}
		case TrustLines:
			err := p.processLedgerTrustLines(transaction, r.GetSequence())
			if err != nil {
				return errors.Wrap(err, "Error in processLedgerTrustLines")
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

	return nil
}

func (p *DatabaseProcessor) processLedgerAccountsForSigner(transaction io.LedgerTransaction) error {
	for _, change := range transaction.GetChanges() {
		if change.Type != xdr.LedgerEntryTypeAccount {
			continue
		}

		if !change.AccountSignersChanged() {
			continue
		}

		// The code below removes all Pre signers adds Post signers but
		// can be improved by finding a diff (check performance first).
		if change.Pre != nil {
			preAccountEntry := change.Pre.MustAccount()
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
			postAccountEntry := change.Post.MustAccount()
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
	}
	return nil
}

func (p *DatabaseProcessor) processLedgerOffers(transaction io.LedgerTransaction, currentLedger uint32) error {
	for _, change := range transaction.GetChanges() {
		if change.Type != xdr.LedgerEntryTypeOffer {
			continue
		}

		var rowsAffected int64
		var err error
		var action string
		var offerID xdr.Int64

		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			action = "inserting"
			offer := change.Post.MustOffer()
			offerID = offer.OfferId
			rowsAffected, err = p.OffersQ.InsertOffer(offer, xdr.Uint32(currentLedger))
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			offer := change.Pre.MustOffer()
			offerID = offer.OfferId
			rowsAffected, err = p.OffersQ.RemoveOffer(offer.OfferId)
		default:
			// Updated
			action = "updating"
			offer := change.Post.MustOffer()
			offerID = offer.OfferId
			rowsAffected, err = p.OffersQ.UpdateOffer(offer, xdr.Uint32(currentLedger))
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
	}
	return nil
}

func (p *DatabaseProcessor) adjustAssetStat(
	trustline xdr.TrustLineEntry,
	deltaBalance xdr.Int64,
	deltaAccounts int32,
) error {
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

func (p *DatabaseProcessor) processLedgerTrustLines(transaction io.LedgerTransaction, currentLedger uint32) error {
	for _, change := range transaction.GetChanges() {
		if change.Type != xdr.LedgerEntryTypeTrustline {
			continue
		}

		var rowsAffected int64
		var err error
		var action string
		var ledgerKey xdr.LedgerKey

		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			action = "inserting"
			trustLine := change.Post.MustTrustLine()
			err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			err = p.adjustAssetStat(trustLine, trustLine.Balance, 1)
			if err != nil {
				return errors.Wrap(err, "Error adjusting asset stat")
			}
			rowsAffected, err = p.TrustLinesQ.InsertTrustLine(trustLine, xdr.Uint32(currentLedger))
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			trustLine := change.Pre.MustTrustLine()
			err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			err = p.adjustAssetStat(trustLine, -trustLine.Balance, -1)
			if err != nil {
				return errors.Wrap(err, "Error adjusting asset stat")
			}
			rowsAffected, err = p.TrustLinesQ.RemoveTrustLine(*ledgerKey.TrustLine)
		default:
			// Updated
			action = "updating"
			preTrustLine := change.Pre.MustTrustLine()
			trustLine := change.Post.MustTrustLine()
			err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			err = p.adjustAssetStat(trustLine, trustLine.Balance-preTrustLine.Balance, 0)
			if err != nil {
				return errors.Wrap(err, "Error adjusting asset stat")
			}
			rowsAffected, err = p.TrustLinesQ.UpdateTrustLine(trustLine, xdr.Uint32(currentLedger))
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
	}
	return nil
}

func (p *DatabaseProcessor) Name() string {
	return fmt.Sprintf("DatabaseProcessor (%s)", p.Action)
}

func (p *DatabaseProcessor) Reset() {}

var _ ingestpipeline.StateProcessor = &DatabaseProcessor{}
var _ ingestpipeline.LedgerProcessor = &DatabaseProcessor{}
