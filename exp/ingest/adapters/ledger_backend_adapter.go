package ingestadapters

import (
	"fmt"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/io"
)

type LedgerBackendAdapter struct {
	backend ingest.LedgerBackend
}

func (lba *LedgerBackendAdapter) GetLatestLedgerSequence() (uint32, error) {
	// TODO: placeholder
	return 0, fmt.Errorf("Not implemented yet")
}

func (lba *LedgerBackendAdapter) GetLedger(uint32) (io.LedgerReader, error) {
	// TODO: placeholder
	return nil, fmt.Errorf("not implemented yet")
}
