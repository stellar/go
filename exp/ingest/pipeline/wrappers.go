package pipeline

import (
	"context"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (w *stateProcessorWrapper) Process(ctx context.Context, store *supportPipeline.Store, readCloser supportPipeline.ReadCloser, writeCloser supportPipeline.WriteCloser) error {
	return w.StateProcessor.ProcessState(
		ctx,
		store,
		&readCloserWrapperState{readCloser},
		&writeCloserWrapperState{writeCloser},
	)
}

func (w *ledgerProcessorWrapper) Process(ctx context.Context, store *supportPipeline.Store, readCloser supportPipeline.ReadCloser, writeCloser supportPipeline.WriteCloser) error {
	return w.LedgerProcessor.ProcessLedger(
		ctx,
		store,
		&readCloserWrapperLedger{readCloser},
		&writeCloserWrapperLedger{writeCloser},
	)
}

func (w *stateReadCloserWrapper) GetContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, LedgerSequenceContextKey, w.StateReadCloser.GetSequence())
	return ctx
}

func (w *stateReadCloserWrapper) Read() (interface{}, error) {
	return w.StateReadCloser.Read()
}

func (w *readCloserWrapperState) GetSequence() uint32 {
	return GetLedgerSequenceFromContext(w.ReadCloser.GetContext())
}

func (w *ledgerReadCloserWrapper) GetContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, LedgerSequenceContextKey, w.LedgerReadCloser.GetSequence())
	ctx = context.WithValue(ctx, LedgerHeaderContextKey, w.LedgerReadCloser.GetHeader())
	return ctx
}

func (w *ledgerReadCloserWrapper) Read() (interface{}, error) {
	return w.LedgerReadCloser.Read()
}

func (w *readCloserWrapperState) Read() (xdr.LedgerEntryChange, error) {
	object, err := w.ReadCloser.Read()
	if err != nil {
		return xdr.LedgerEntryChange{}, err
	}

	entry, ok := object.(xdr.LedgerEntryChange)
	if !ok {
		return xdr.LedgerEntryChange{}, errors.New("Read object is not xdr.LedgerEntryChange")
	}

	return entry, nil
}

func (w *readCloserWrapperLedger) GetSequence() uint32 {
	return GetLedgerSequenceFromContext(w.ReadCloser.GetContext())
}

func (w *readCloserWrapperLedger) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return GetLedgerHeaderFromContext(w.ReadCloser.GetContext())
}

func (w *readCloserWrapperLedger) Read() (io.LedgerTransaction, error) {
	object, err := w.ReadCloser.Read()
	if err != nil {
		return io.LedgerTransaction{}, err
	}

	entry, ok := object.(io.LedgerTransaction)
	if !ok {
		return io.LedgerTransaction{}, errors.New("Read object is not io.LedgerTransaction")
	}

	return entry, nil
}

func (w *writeCloserWrapperState) Write(entry xdr.LedgerEntryChange) error {
	return w.WriteCloser.Write(entry)
}

func (w *writeCloserWrapperLedger) Write(entry io.LedgerTransaction) error {
	return w.WriteCloser.Write(entry)
}
