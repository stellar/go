package hubble

import (
	"context"
	"fmt"
	stdio "io"
	"sync"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
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

// SerializeEntryProcessor reads and serializes account entries.
// Note that it only serializes the first encountered example of an entry, to allow
// for quicker debugging and testing of our serialization process.
// TODO: Do not only process the first example entry.
type SerializeEntryProcessor struct {
	SimpleProcessor
}

// ProcessState reads, prints, and writes all changes to ledger state.
func (p *SerializeEntryProcessor) ProcessState(ctx context.Context, store *supportPipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer w.Close()
	defer r.Close()

	entries := 0
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

		entryType := entry.EntryType().String()

		// If we've already seen an example of this entry, we break,
		// as we only wish to serialize a single example now.
		// TODO: Remove this check.
		if _, ok := entriesCountDict[entryType]; ok {
			// entriesCountDict[entryType]++
			continue
		} else {
			entriesCountDict[entryType] = 1
		}

		entries++
		serializeEntry(entry)

		// TODO: Remove. Only here because we want to serialize a single entry.
		// if entriesCountDict[entryType] == 10 {
		// 	break
		// }

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
func (p *SerializeEntryProcessor) Name() string {
	return "SerializeEntryProcessor"
}
