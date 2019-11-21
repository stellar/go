// +build go1.13

package hubble

import (
	"context"
	"fmt"
	stdio "io"
	"strconv"

	"github.com/olivere/elastic/v7"
	"github.com/stellar/go/exp/ingest/io"
	ingestPipeline "github.com/stellar/go/exp/ingest/pipeline"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
)

// ESProcessor serializes ledger change entries as JSONs and writes them
// to an ElasticSearch cluster.
type ESProcessor struct {
	client *elastic.Client
	index  string
}

var _ ingestPipeline.StateProcessor = &ESProcessor{}

// Reset is a no-op for this processor.
func (p *ESProcessor) Reset() {}

// ProcessState reads, prints, and writes changes to ledger state to ElasticSearch.
func (p *ESProcessor) ProcessState(ctx context.Context, store *supportPipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer w.Close()
	defer r.Close()

	numEntries := 0
	for {
		entry, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		// Step 1: convert entry to JSON-ified string.
		// TODO: Move this to a separate processor. Currently, this is not possible,
		// as the ingestion system does not read and write custom structs
		// between pipeline nodes.
		entryJSONStr, err := serializeLedgerEntryChange(entry)
		if err != nil {
			return errors.Wrap(err, "couldn't convert ledgerentry to json")
		}

		// Step 2: Augment the JSON-fied data.
		// TODO: Implement Step 2 as a separate processor.

		// Step 3: put entry as JSON in ElasticSearch.
		// TODO: Take ID from entry, rather than a counter.
		err = p.PutEntry(ctx, entryJSONStr, numEntries)
		if err != nil {
			return errors.Wrap(err, "couldn't put entry json in elasticsearch")
		}
		numEntries++

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	fmt.Printf("Found %d total entries\n", numEntries)
	return nil
}

// Name returns the processor name.
func (p *ESProcessor) Name() string {
	return "ESProcessor"
}

// PutEntry puts a ledger entry in ElasticSearch.
func (p *ESProcessor) PutEntry(ctx context.Context, entry string, id int) error {
	idStr := strconv.Itoa(id)
	_, err := p.client.Index().Index(p.index).Id(idStr).BodyString(entry).Do(ctx)
	return err
}
