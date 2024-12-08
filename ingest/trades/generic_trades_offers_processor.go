package trades

import (
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"io"
)

func ProcessTradesFromLedger(ledger xdr.LedgerCloseMeta, networkPassPhrase string) ([]TradeEvent, error) {
	changeReader, err := ingest.NewLedgerChangeReaderFromLedgerCloseMeta(networkPassPhrase, ledger)
	if err != nil {
		return []TradeEvent{}, errors.Wrap(err, "Error creating ledger change reader")
	}
	defer changeReader.Close()

	tradeEvents := make([]TradeEvent, 0)
	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []TradeEvent{}, errors.Wrap(err, "Error reading ledger change")
		}
		// Process trades from the change
		tradesFromChange, err := processTradesFromChange(change)
		if err != nil {
			return nil, errors.Wrap(err, "Error processing trades from change")
		}

		// Append to the overall trade events slice
		tradeEvents = append(tradeEvents, tradesFromChange...)
	}

	return tradeEvents, nil
}

func processTradesFromChange(change ingest.Change) ([]TradeEvent, error) {
	var tradeEvents []TradeEvent

	switch change.Type {
	case xdr.LedgerEntryTypeOffer:
		trades, err := processOffersFromChange(change)
		if err != nil {
			return nil, errors.Wrap(err, "Error processing offers")
		}
		tradeEvents = append(tradeEvents, trades...)
	case xdr.LedgerEntryTypeLiquidityPool:
		trades, err := processLiquidityPoolEventsFromChange(change)
		if err != nil {
			return nil, errors.Wrap(err, "Error processing liquidity pool events")
		}
		tradeEvents = append(tradeEvents, trades...)

	default:
		// Nothing to do
	}

	return tradeEvents, nil
}

func processOffersFromChange(change ingest.Change) ([]TradeEvent, error) {

	return []TradeEvent{}, nil
}

func processLiquidityPoolEventsFromChange(change ingest.Change) ([]TradeEvent, error) {
	return []TradeEvent{}, nil
}
