package hubble

import (
	"context"
	"fmt"
	stdio "io"
	"sync"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/xdr"
)

// SimpleProcessor is a basic unit to use in more complex processors.
// It wraps a mutex and number of calls.
type SimpleProcessor struct {
	sync.Mutex
	callCount int
}

// Reset re-initializes the processor by putting the call count to 0.
func (n *SimpleProcessor) Reset() {
	n.callCount = 0
}

// IncrementAndReturnCallCount notes an additional call in the processor state.
func (n *SimpleProcessor) IncrementAndReturnCallCount() int {
	n.Lock()
	defer n.Unlock()
	n.callCount++
	return n.callCount
}

// PrettyPrintEntryProcessor reads and pretty prints account entries.
// Note that now, it prints the first encountered example of an entry, to allow
// for quicker debugging and testing of our printing process.
type PrettyPrintEntryProcessor struct {
	SimpleProcessor
}

// ProcessState reads, prints, and writes all changes to ledger state.
func (p *PrettyPrintEntryProcessor) ProcessState(ctx context.Context, store *supportPipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer w.Close()
	defer r.Close()

	entries := 0
	prefix := "\t"
	entriesCountDict := make(map[string]int)
	for {
		entry, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		// The below logic is meant to only pretty print one example for each entry type.
		// TODO: Remove the below checks, up and until printing the entry.
		// If we have found an example of each of the 4 ledger entry types, exit.
		if len(entriesCountDict) == 4 {
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
		if _, ok := entriesCountDict[entryType]; ok {
			// entriesCountDict[entryType]++
			continue
		} else {
			entriesCountDict[entryType] = 1
		}

		entries++
		fmt.Println(prettyPrintEntry(entry, prefix))

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	fmt.Printf("Found %d entries\n", entries)
	return nil
}

// Name returns the processor name.
func (p *PrettyPrintEntryProcessor) Name() string {
	return "PrettyPrintEntryProcessor"
}
