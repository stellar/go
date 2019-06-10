package processors

import (
	"github.com/stellar/go/xdr"
)

// CSVPrinter prints ledger entries to a file.
type CSVPrinter struct {
	Filename string
}

// StatusLogger prints number of processed entries every N
// entries.
type StatusLogger struct {
	N int
}

// EntryTypeFilter is a pipeline.StateProcessor that filters out all
// entries that are not of type `Type`.
type EntryTypeFilter struct {
	Type xdr.LedgerEntryType

	concurrentProcessor
}

type concurrentProcessor struct{}

func (n *concurrentProcessor) IsConcurrent() bool {
	return true
}
