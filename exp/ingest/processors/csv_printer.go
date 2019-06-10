package processors

import (
	"context"
	"encoding/base64"
	"fmt"
	stdio "io"
	"os"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (p *CSVPrinter) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer r.Close()
	defer w.Close()

	f, err := os.Create(p.Filename)
	if err != nil {
		return err
	}

	defer f.Close()

	for {
		entryChange, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if entryChange.Type != xdr.LedgerEntryChangeTypeLedgerEntryState {
			return errors.New("CSVPrinter requires LedgerEntryChangeTypeLedgerEntryState changes only")
		}

		entry := entryChange.State

		switch entry.Data.Type {
		case xdr.LedgerEntryTypeAccount:
			fmt.Fprintf(
				f,
				"%s,%d,%d\n",
				entry.Data.Account.AccountId.Address(),
				entry.Data.Account.Balance,
				entry.Data.Account.SeqNum,
			)
		case xdr.LedgerEntryTypeTrustline:
			fmt.Println("TODO")
		case xdr.LedgerEntryTypeOffer:
			fmt.Println("TODO")
		case xdr.LedgerEntryTypeData:
			fmt.Fprintf(
				f,
				"%s,%s,%s\n",
				entry.Data.Data.AccountId.Address(),
				entry.Data.Data.DataName,
				base64.StdEncoding.EncodeToString(entry.Data.Data.DataValue),
			)
		default:
			return fmt.Errorf("Invalid LedgerEntryType: %d", entryChange.Type)
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

func (n *CSVPrinter) IsConcurrent() bool {
	return false
}

func (p *CSVPrinter) Name() string {
	return "CSVPrinter"
}

var _ ingestpipeline.StateProcessor = &EntryTypeFilter{}
