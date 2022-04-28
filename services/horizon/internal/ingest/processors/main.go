package processors

import (
	"github.com/guregu/null"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

var log = logpkg.DefaultLogger.WithField("service", "ingest")

const maxBatchSize = 100000

func ledgerEntrySponsorToNullString(entry xdr.LedgerEntry) null.String {
	sponsoringID := entry.SponsoringID()

	var sponsor null.String
	if sponsoringID != nil {
		sponsor.SetValid((*sponsoringID).Address())
	}

	return sponsor
}

func formatSequenceLedger(ledger xdr.Uint32) null.Int {
	return null.NewInt(int64(ledger), ledger != 0)
}

func formatSequenceTime(time xdr.TimePoint) null.Int {
	return null.NewInt(int64(time), time != 0)
}
