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

// TODO replace `fmt` with `encoding/csv`

func (p *CSVPrinter) fileHandle() (*os.File, error) {
	if p.Filename == "" {
		return os.Stdout, nil
	}

	return os.Create(p.Filename)
}

func (p *CSVPrinter) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer r.Close()
	defer w.Close()

	f, err := p.fileHandle()
	if err != nil {
		return err
	}

	if f != os.Stdout {
		defer f.Close()
	}

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

func (p *CSVPrinter) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReadCloser, w io.LedgerWriteCloser) error {
	defer r.Close()
	defer w.Close()

	f, err := p.fileHandle()
	if err != nil {
		return err
	}

	if f != os.Stdout {
		defer f.Close()
	}

	for {
		transaction, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		fmt.Fprintf(
			f,
			"%d,%t,%s,%d,%d\n",
			transaction.Index,
			transaction.Result.Result.Result.Code == xdr.TransactionResultCodeTxSuccess,
			transaction.Envelope.Tx.SourceAccount.Address(),
			len(transaction.Envelope.Tx.Operations),
			transaction.Envelope.Tx.Fee,
		)

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

var _ ingestpipeline.StateProcessor = &CSVPrinter{}
var _ ingestpipeline.LedgerProcessor = &CSVPrinter{}
