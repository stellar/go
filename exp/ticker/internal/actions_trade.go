package ticker

import (
	"fmt"
	"strconv"
	"time"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	"github.com/stellar/go/exp/ticker/internal/scraper"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// BackfillTrades ingest the most recent trades (limited to numDays) directly from Horizon
// into the database.
func BackfillTrades(s *tickerdb.TickerSession, numDays int, limit int) error {
	c := horizonclient.DefaultPublicNetClient
	now := time.Now()
	since := now.AddDate(0, 0, -numDays)
	trades, err := scraper.FetchAllTrades(c, since, limit)
	if err != nil {
		return err
	}

	var dbTrades []tickerdb.Trade

	for _, trade := range trades {
		bFound, bID, err := s.GetAssetByCodeAndPublicKey(
			trade.BaseAssetCode,
			trade.BaseAssetIssuer,
		)
		if err != nil {
			return err
		}

		cFound, cID, err := s.GetAssetByCodeAndPublicKey(
			trade.CounterAssetCode,
			trade.CounterAssetIssuer,
		)
		if err != nil {
			return err
		}

		if !bFound || !cFound {
			continue
		}

		dbTrade, err := hProtocolTradeToDBTrade(trade, bID, cID)
		if err != nil {
			fmt.Println("Could not convert entry to DB Trade:", err)
			continue
		}
		dbTrades = append(dbTrades, dbTrade)
	}

	fmt.Printf("Inserting %d entries in the database.\n", len(dbTrades))
	err = s.BulkInsertTrades(dbTrades)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func hProtocolTradeToDBTrade(
	hpt hProtocol.Trade,
	baseAssetID int32,
	counterAssetID int32,
) (trade tickerdb.Trade, err error) {
	fBaseAmount, err := strconv.ParseFloat(hpt.BaseAmount, 64)
	if err != nil {
		return
	}
	fCounterAmount, err := strconv.ParseFloat(hpt.CounterAmount, 64)
	if err != nil {
		return
	}

	fPrice := float64(hpt.Price.D) / float64(hpt.Price.N)

	trade = tickerdb.Trade{
		HorizonID:       hpt.ID,
		LedgerCloseTime: hpt.LedgerCloseTime,
		OfferID:         hpt.OfferID,
		BaseOfferID:     hpt.BaseOfferID,
		BaseAccount:     hpt.BaseAccount,
		BaseAmount:      fBaseAmount,
		BaseAssetID:     baseAssetID,
		CounterOfferID:  hpt.CounterOfferID,
		CounterAccount:  hpt.CounterAccount,
		CounterAmount:   fCounterAmount,
		CounterAssetID:  counterAssetID,
		BaseIsSeller:    hpt.BaseIsSeller,
		Price:           fPrice,
	}

	return
}
