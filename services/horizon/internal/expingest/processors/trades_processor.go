package processors

import (
	"context"
	stdio "io"
	"time"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TradeProcessor operations processor
type TradeProcessor struct {
	TradesQ history.QTrades
}

// ProcessLedger process the given ledger
func (p *TradeProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
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
	r.IgnoreUpgradeChanges()

	// Exit early if not ingesting into a DB
	if v := ctx.Value(IngestUpdateDatabase); !(v != nil && v.(bool)) {
		return nil
	}

	ledger := r.GetHeader()

	var inserts []history.InsertTrade
	var buyers []string
	accountSet := map[string]int64{}
	assets := []xdr.Asset{}

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

		if !transaction.Successful() {
			continue
		}

		var txInserts []history.InsertTrade
		var txBuyers []string
		txInserts, txBuyers, err = p.extractTrades(ledger, transaction)
		if err != nil {
			return err
		}

		for i, insert := range txInserts {
			buyer := txBuyers[i]
			accountSet[insert.Trade.SellerId.Address()] = 0
			accountSet[buyer] = 0
			assets = append(assets, insert.Trade.AssetSold, insert.Trade.AssetBought)

			inserts = append(inserts, insert)
			buyers = append(buyers, buyer)
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	if len(inserts) > 0 {
		batch := p.TradesQ.NewTradeBatchInsertBuilder(maxBatchSize)
		accountSet, err = p.TradesQ.CreateAccounts(mapKeysToList(accountSet))
		if err != nil {
			return errors.Wrap(err, "Error creating account ids")
		}

		var assetMap map[string]history.Asset
		assetMap, err = p.TradesQ.CreateAssets(assets)
		if err != nil {
			return errors.Wrap(err, "Error creating asset ids")
		}

		for i, insert := range inserts {
			insert.BuyerAccountID = accountSet[buyers[i]]
			insert.SellerAccountID = accountSet[insert.Trade.SellerId.Address()]
			insert.SoldAssetID = assetMap[insert.Trade.AssetSold.String()].ID
			insert.BoughtAssetID = assetMap[insert.Trade.AssetBought.String()].ID
			if err = batch.Add(insert); err != nil {
				return errors.Wrap(err, "Error adding trade to batch")
			}

			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}

		if err = batch.Exec(); err != nil {
			return errors.Wrap(err, "Error flushing operation batch")
		}
	}

	return nil
}

func (p *TradeProcessor) findTradeSellPrice(
	transaction io.LedgerTransaction,
	opidx int,
	trade xdr.ClaimOfferAtom,
) (xdr.Price, error) {
	var price xdr.Price
	key := xdr.LedgerKey{}
	key.SetOffer(trade.SellerId, uint64(trade.OfferId))

	changes, err := transaction.GetOperationChanges(uint32(opidx))
	if err != nil {
		return price, errors.Wrap(err, "could not determine changes for operation")
	}

	found := false
	var change io.Change
	for i := len(changes) - 1; i >= 0; i-- {
		change = changes[i]
		if change.Pre != nil && key.Equals(change.Pre.LedgerKey()) {
			found = true
			break
		}
	}

	if !found {
		return price, errors.Wrap(err, "could not find change for trade offer")
	}

	return change.Pre.Data.Offer.Price, nil
}

func (p *TradeProcessor) extractTrades(
	ledger xdr.LedgerHeaderHistoryEntry,
	transaction io.LedgerTransaction,
) ([]history.InsertTrade, []string, error) {
	var inserts []history.InsertTrade
	var buyerAccounts []string

	closeTime := time.Unix(int64(ledger.Header.ScpValue.CloseTime), 0).UTC()

	opResults := transaction.Result.Result.Result.MustResults()
	for opidx, op := range transaction.Envelope.Tx.Operations {
		var trades []xdr.ClaimOfferAtom
		var buyOfferExists bool
		var buyOffer xdr.OfferEntry

		switch op.Body.Type {
		case xdr.OperationTypePathPaymentStrictReceive:
			trades = opResults[opidx].MustTr().MustPathPaymentStrictReceiveResult().
				MustSuccess().
				Offers

		case xdr.OperationTypePathPaymentStrictSend:
			trades = opResults[opidx].MustTr().
				MustPathPaymentStrictSendResult().
				MustSuccess().
				Offers

		case xdr.OperationTypeManageBuyOffer:
			manageOfferResult := opResults[opidx].MustTr().MustManageBuyOfferResult().
				MustSuccess()
			trades = manageOfferResult.OffersClaimed
			buyOffer, buyOfferExists = manageOfferResult.Offer.GetOffer()

		case xdr.OperationTypeManageSellOffer:
			manageOfferResult := opResults[opidx].MustTr().MustManageSellOfferResult().
				MustSuccess()
			trades = manageOfferResult.OffersClaimed
			buyOffer, buyOfferExists = manageOfferResult.Offer.GetOffer()

		case xdr.OperationTypeCreatePassiveSellOffer:
			result := opResults[opidx].MustTr()

			// KNOWN ISSUE:  stellar-core creates results for CreatePassiveOffer operations
			// with the wrong result arm set.
			if result.Type == xdr.OperationTypeManageSellOffer {
				manageOfferResult := result.MustManageSellOfferResult().MustSuccess()
				trades = manageOfferResult.OffersClaimed
				buyOffer, buyOfferExists = manageOfferResult.Offer.GetOffer()
			} else {
				passiveOfferResult := result.MustCreatePassiveSellOfferResult().MustSuccess()
				trades = passiveOfferResult.OffersClaimed
				buyOffer, buyOfferExists = passiveOfferResult.Offer.GetOffer()
			}
		}

		opID := toid.New(
			int32(ledger.Header.LedgerSeq), int32(transaction.Index), int32(opidx+1),
		).ToInt64()
		for order, trade := range trades {
			// stellar-core will opportunisticly garbage collect invalid offers (in the
			// event that a trader spends down their balance).  These garbage collected
			// offers get emitted in the result with the amount values set to zero.
			//
			// These zeroed ClaimOfferAtom values do not represent trades, and so we
			// skip them.
			if trade.AmountBought == 0 && trade.AmountSold == 0 {
				continue
			}

			sellOfferPrice, err := p.findTradeSellPrice(transaction, opidx, trade)
			if err != nil {
				return nil, nil, err
			}

			inserts = append(inserts, history.InsertTrade{
				HistoryOperationID: opID,
				Order:              int32(order),
				LedgerCloseTime:    closeTime,
				BuyOfferExists:     buyOfferExists,
				Trade:              trade,
				SellPrice:          sellOfferPrice,
				BuyOfferID:         int64(buyOffer.OfferId),
			})

			var buyerAddress string
			if buyer := op.SourceAccount; buyer != nil {
				buyerAddress = buyer.Address()
			} else {
				buyerAddress = transaction.Envelope.Tx.SourceAccount.Address()
			}
			buyerAccounts = append(buyerAccounts, buyerAddress)
		}
	}

	return inserts, buyerAccounts, nil
}

func mapKeysToList(set map[string]int64) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	return keys
}

// Name processor name
func (p *TradeProcessor) Name() string {
	return "TradeProcessor"
}

// Reset resets processor
func (p *TradeProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &TradeProcessor{}
