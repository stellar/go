// +build go1.13

package hubble

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// PrettyPrintEntryProcessor reads and pretty prints account entries.
// Note that now, it prints the first encountered example of an entry, to allow
// for quicker debugging and testing of our printing process.
type PrettyPrintEntryProcessor struct{}

// Reset is a no-op for this processor.
func (p *PrettyPrintEntryProcessor) Reset() {}

// ProcessState reads, prints, and writes all changes to ledger state.
func (p *PrettyPrintEntryProcessor) ProcessState(ctx context.Context, store *supportPipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer w.Close()
	defer r.Close()

	entryTypeSet := make(map[string]bool)
	for {
		entry, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		// If we have found an example of each of the 4 ledger entry types, exit.
		if len(entryTypeSet) == 4 {
			break
		}

		// Skip entries that are not of type `State`.
		// This can be swapped with other types: Removed, Created, Updated.
		if entry.Type != xdr.LedgerEntryChangeTypeLedgerEntryState {
			continue
		}

		// If we've already seen an example of this entry, we break,
		// as we only wish to print a single example now.
		entryType := entry.EntryType().String()
		if entryTypeSet[entryType] {
			continue
		} else {
			entryTypeSet[entryType] = true
		}
		bytes, err := serializeLedgerEntryChange(entry)
		if err != nil {
			return errors.Wrap(err, "converting ledgerentry to json")
		}
		fmt.Printf("%s\n", bytes)

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	fmt.Printf("Found %d entries\n", len(entryTypeSet))
	return nil
}

// Name returns the processor name.
func (p *PrettyPrintEntryProcessor) Name() string {
	return "PrettyPrintEntryProcessor"
}
