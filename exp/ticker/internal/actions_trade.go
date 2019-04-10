package ticker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	"github.com/stellar/go/exp/ticker/internal/scraper"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// StreamTrades constantly streams and ingests new trades directly from horizon.
// Use context.WithCancel to stop streaming or context.Background() to stream indefinitely.
func StreamTrades(ctx context.Context, s *tickerdb.TickerSession, c *horizonclient.Client) error {
	handler := func(trade hProtocol.Trade) {
		fmt.Print("New trade arrived:", trade.ID, trade.LedgerCloseTime)
		bID, cID, err := findBaseAndCounter(s, trade)
		if err != nil {
			return
		}
		dbTrade, err := hProtocolTradeToDBTrade(trade, bID, cID)
		if err != nil {
			return
		}

		err = s.BulkInsertTrades([]tickerdb.Trade{dbTrade})
		if err != nil {
			fmt.Println("Could not insert trade in database:", trade.ID)
		}
	}

	// Ensure we start streaming from the last stored trade
	lastTrade, err := s.GetLastTrade()
	if err != nil {
		return err
	}

	cursor := lastTrade.HorizonID
	return scraper.StreamNewTrades(ctx, c, handler, cursor)
}

// BackfillTrades ingest the most recent trades (limited to numDays) directly from Horizon
// into the database.
func BackfillTrades(s *tickerdb.TickerSession, c *horizonclient.Client, numDays int, limit int) error {
	now := time.Now()
	since := now.AddDate(0, 0, -numDays)
	trades, err := scraper.FetchAllTrades(c, since, limit)
	if err != nil {
		return err
	}

	var dbTrades []tickerdb.Trade

	for _, trade := range trades {
		bID, cID, err := findBaseAndCounter(s, trade)
		if err != nil {
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

// findBaseAndCounter tries to find the Base and Counter assets IDs in the database,
// and returns an error if it doesn't find any.
func findBaseAndCounter(s *tickerdb.TickerSession, trade hProtocol.Trade) (bID int32, cID int32, err error) {
	bFound, bID, err := s.GetAssetByCodeAndIssuerAccount(
		trade.BaseAssetCode,
		trade.BaseAssetIssuer,
	)
	if err != nil {
		return
	}

	cFound, cID, err := s.GetAssetByCodeAndIssuerAccount(
		trade.CounterAssetCode,
		trade.CounterAssetIssuer,
	)
	if err != nil {
		return
	}

	if !bFound || !cFound {
		err = errors.New("base or counter asset no found")
		return
	}

	return
}

// hProtocolTradeToDBTrade converts from a hProtocol.Trade to a tickerdb.Trade
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
