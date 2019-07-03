package processors

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
)

func (p *EntryTypeFilter) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer r.Close()
	defer w.Close()

	for {
		entryChange, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if entryChange.EntryType() == p.Type {
			err := w.Write(entryChange)
			if err != nil {
				if err == stdio.ErrClosedPipe {
					// Reader does not need more data
					return nil
				}
				return err
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	return nil
}

func (p *EntryTypeFilter) Name() string {
	return fmt.Sprintf("EntryTypeFilter (%s)", p.Type)
}

var _ ingestpipeline.StateProcessor = &EntryTypeFilter{}
