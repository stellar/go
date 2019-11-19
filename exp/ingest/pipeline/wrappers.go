package pipeline

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (w *stateProcessorWrapper) Process(ctx context.Context, store *supportPipeline.Store, reader supportPipeline.Reader, writer supportPipeline.Writer) error {
	return w.StateProcessor.ProcessState(
		ctx,
		store,
		&readerWrapperState{reader},
		&writerWrapperState{writer},
	)
}

func (w *ledgerProcessorWrapper) Process(ctx context.Context, store *supportPipeline.Store, reader supportPipeline.Reader, writer supportPipeline.Writer) error {
	return w.LedgerProcessor.ProcessLedger(
		ctx,
		store,
		&readerWrapperLedger{
			Reader:         reader,
			upgradeChanges: GetLedgerUpgradeChangesFromContext(reader.GetContext()),
		},
		&writerWrapperLedger{writer},
	)
}

func (w *stateReaderWrapper) GetContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, LedgerSequenceContextKey, w.StateReader.GetSequence())
	return ctx
}

func (w *stateReaderWrapper) Read() (interface{}, error) {
	return w.StateReader.Read()
}

func (w *readerWrapperState) GetSequence() uint32 {
	return GetLedgerSequenceFromContext(w.Reader.GetContext())
}

func (w *ledgerReaderWrapper) GetContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, LedgerSequenceContextKey, w.LedgerReader.GetSequence())
	ctx = context.WithValue(ctx, LedgerHeaderContextKey, w.LedgerReader.GetHeader())
	ctx = context.WithValue(ctx, LedgerSuccessTxCountContextKey, w.LedgerReader.SuccessfulLedgerOperationCount())
	ctx = context.WithValue(ctx, LedgerFailedTxCountContextKey, w.LedgerReader.FailedTransactionCount())
	ctx = context.WithValue(ctx, LedgerOpCountContextKey, w.LedgerReader.SuccessfulLedgerOperationCount())
	ctx = context.WithValue(ctx, LedgerCloseTimeContextKey, w.LedgerReader.CloseTime())

	// Save upgrade changes in context. UpgradeChangesContainer is implemented by
	// *io.DBLedgerReader and readerWrapperLedger.
	reader, ok := w.LedgerReader.(io.UpgradeChangesContainer)
	if !ok {
		panic(fmt.Sprintf("Cannot get upgrade changes from unknown reader type: %T", w.LedgerReader))
	}

	ctx = context.WithValue(ctx, LedgerUpgradeChangesContextKey, reader.GetUpgradeChanges())

	return ctx
}

func (w *ledgerReaderWrapper) Read() (interface{}, error) {
	return w.LedgerReader.Read()
}

func (w *readerWrapperState) Read() (xdr.LedgerEntryChange, error) {
	object, err := w.Reader.Read()
	if err != nil {
		return xdr.LedgerEntryChange{}, err
	}

	entry, ok := object.(xdr.LedgerEntryChange)
	if !ok {
		return xdr.LedgerEntryChange{}, errors.New("Read object is not xdr.LedgerEntryChange")
	}

	return entry, nil
}

func (w *readerWrapperLedger) GetSequence() uint32 {
	return GetLedgerSequenceFromContext(w.Reader.GetContext())
}

func (w *readerWrapperLedger) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return GetLedgerHeaderFromContext(w.Reader.GetContext())
}

func (w *readerWrapperLedger) Read() (io.LedgerTransaction, error) {
	object, err := w.Reader.Read()
	if err != nil {
		return io.LedgerTransaction{}, err
	}

	entry, ok := object.(io.LedgerTransaction)
	if !ok {
		return io.LedgerTransaction{}, errors.New("Read object is not io.LedgerTransaction")
	}

	return entry, nil
}

func (w *readerWrapperLedger) SuccessfulTransactionCount() int {
	return GetSuccessTxCountFromContext(w.Reader.GetContext())
}

func (w *readerWrapperLedger) FailedTransactionCount() int {
	return GetFailedTxCountContext(w.Reader.GetContext())
}

func (w *readerWrapperLedger) SuccessfulLedgerOperationCount() int {
	return GetSuccessTxCountFromContext(w.Reader.GetContext())
}

func (w *readerWrapperLedger) CloseTime() int64 {
	return GetCloseTimeFromContext(w.Reader.GetContext())
}

// ReadUpgradeChange returns the next ledger upgrade change or EOF if there are
// no more upgrade changes. Not safe for concurrent use!
func (w *readerWrapperLedger) ReadUpgradeChange() (io.Change, error) {
	w.readUpgradeChangeCalled = true

	if w.currentUpgradeChange < len(w.upgradeChanges) {
		change := w.upgradeChanges[w.currentUpgradeChange]
		w.currentUpgradeChange++
		return change, nil
	}

	return io.Change{}, stdio.EOF
}

func (w *readerWrapperLedger) IgnoreUpgradeChanges() {
	w.ignoreUpgradeChanges = true
}

func (w *readerWrapperLedger) GetUpgradeChanges() []io.Change {
	return w.upgradeChanges
}

func (w *readerWrapperLedger) Close() error {
	if !w.ignoreUpgradeChanges &&
		(!w.readUpgradeChangeCalled || w.currentUpgradeChange != len(w.upgradeChanges)) {
		return errors.New("Ledger upgrade changes not read! Use ReadUpgradeChange() method.")
	}

	// Call IgnoreUpgradeChanges on a wrapped reader because `readerWrapperLedger`
	// is responsible for streaming ledger upgrade changes now.
	wrapper, ok := w.Reader.(*ledgerReaderWrapper)
	if ok {
		wrapper.LedgerReader.IgnoreUpgradeChanges()
	}

	return w.Reader.Close()
}

func (w *writerWrapperState) Write(entry xdr.LedgerEntryChange) error {
	return w.Writer.Write(entry)
}

func (w *writerWrapperLedger) Write(entry io.LedgerTransaction) error {
	return w.Writer.Write(entry)
}
