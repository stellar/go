package pipeline

import (
	"context"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/xdr"
)

type ContextKey string

const (
	LedgerSequenceContextKey       ContextKey = "ledger_sequence"
	LedgerHeaderContextKey         ContextKey = "ledger_header"
	LedgerUpgradeChangesContextKey ContextKey = "ledger_upgrade_changes"
	LedgerSuccessTxCountContextKey ContextKey = "ledger_success_tx_count"
	LedgerFailedTxCountContextKey  ContextKey = "ledger_failed_tx_count"
	LedgerOpCountContextKey        ContextKey = "ledger_op_count"
	LedgerCloseTimeContextKey      ContextKey = "ledger_close_time"
)

func GetSuccessTxCountFromContext(ctx context.Context) int {
	v := ctx.Value(LedgerSuccessTxCountContextKey)

	if v == nil {
		panic("ledger success tx count not found in context")
	}

	return v.(int)
}

func GetFailedTxCountContext(ctx context.Context) int {
	v := ctx.Value(LedgerFailedTxCountContextKey)

	if v == nil {
		panic("ledger failed tx count not found in context")
	}

	return v.(int)
}

func GetOpCountFromContext(ctx context.Context) int {
	v := ctx.Value(LedgerOpCountContextKey)

	if v == nil {
		panic("ledger op count not found in context")
	}

	return v.(int)
}

func GetCloseTimeFromContext(ctx context.Context) int64 {
	v := ctx.Value(LedgerOpCountContextKey)

	if v == nil {
		panic("ledger close time not found in context")
	}

	return v.(int64)
}

func GetLedgerSequenceFromContext(ctx context.Context) uint32 {
	v := ctx.Value(LedgerSequenceContextKey)

	if v == nil {
		panic("ledger sequence not found in context")
	}

	return v.(uint32)
}

func GetLedgerHeaderFromContext(ctx context.Context) xdr.LedgerHeaderHistoryEntry {
	v := ctx.Value(LedgerHeaderContextKey)

	if v == nil {
		panic("ledger header not found in context")
	}

	return v.(xdr.LedgerHeaderHistoryEntry)
}

func GetLedgerUpgradeChangesFromContext(ctx context.Context) []io.Change {
	v := ctx.Value(LedgerUpgradeChangesContextKey)

	if v == nil {
		panic("ledger upgrade changes not found in context")
	}

	return v.([]io.Change)
}

type StatePipeline struct {
	supportPipeline.Pipeline
}

type LedgerPipeline struct {
	supportPipeline.Pipeline
}

// StateProcessor defines methods required by state processing pipeline.
type StateProcessor interface {
	// ProcessState is a main method of `StateProcessor`. It receives `io.StateReader`
	// that contains object passed down the pipeline from the previous procesor. Writes to
	// `io.StateWriter` will be passed to the next processor. WARNING! `ProcessState`
	// should **always** call `Close()` on `io.StateWriter` when no more object will be
	// written and `Close()` on `io.StateReader` when reading is finished.
	// Data required by following processors (like aggregated data) should be saved in
	// `Store`. Read `Store` godoc to understand how to use it.
	// The first argument `ctx` is a context with cancel. Processor should monitor
	// `ctx.Done()` channel and exit when it returns a value. This can happen when
	// pipeline execution is interrupted, ex. due to an error.
	//
	// Given all information above `ProcessState` should always look like this:
	//
	//    func (p *Processor) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReader, w io.StateWriter) error {
	//    	defer r.Close()
	//    	defer w.Close()
	//
	//    	// Some pre code...
	//
	//    	for {
	//    		entry, err := r.Read()
	//    		if err != nil {
	//    			if err == io.EOF {
	//    				break
	//    			} else {
	//    				return errors.Wrap(err, "Error reading from StateReader in [ProcessorName]")
	//    			}
	//    		}
	//
	//    		// Process entry...
	//
	//    		// Write to StateWriter if needed but exit if pipe is closed:
	//    		err = w.Write(entry)
	//    		if err != nil {
	//    			if err == io.ErrClosedPipe {
	//    				//    Reader does not need more data
	//    				return nil
	//    			}
	//    			return errors.Wrap(err, "Error writing to StateWriter in [ProcessorName]")
	//    		}
	//
	//    		// Return errors if needed...
	//
	//    		// Exit when pipeline terminated due to an error in another processor...
	//    		select {
	//    		case <-ctx.Done():
	//    			return nil
	//    		default:
	//    			continue
	//    		}
	//    	}
	//
	//    	// Some post code...
	//
	//    	return nil
	//    }
	ProcessState(context.Context, *supportPipeline.Store, io.StateReader, io.StateWriter) error
	// Returns processor name. Helpful for errors, debuging and reports.
	Name() string
	// Reset resets internal state of the processor. This is run by the pipeline
	// everytime the processing is done. It is extremely important to implement
	// this method, otherwise internal state of the processor will be maintained
	// between pipeline runs and may result in invalid data.
	Reset()
}

// LedgerProcessor defines methods required by ledger processing pipeline.
type LedgerProcessor interface {
	// ProcessLedger is a main method of `LedgerProcessor`. It receives `io.LedgerReader`
	// that contains object passed down the pipeline from the previous procesor. Writes to
	// `io.LedgerWriter` will be passed to the next processor. WARNING! `ProcessLedger`
	// should **always** call `Close()` on `io.LedgerWriter` when no more object will be
	// written and `Close()` on `io.LedgerReader` when reading is finished.
	// Data required by following processors (like aggregated data) should be saved in
	// `Store`. Read `Store` godoc to understand how to use it.
	// The first argument `ctx` is a context with cancel. Processor should monitor
	// `ctx.Done()` channel and exit when it returns a value. This can happen when
	// pipeline execution is interrupted, ex. due to an error.
	// Please note that processor can filter transactions (by not passing them to
	// `io.LedgerWriter`) but it cannot filter ledger upgrade changes
	// (`io.LeaderReader.ReadUpgradeChange`). All upgrade changes will be available
	// for the next processor to read.
	//
	// Given all information above `ProcessLedger` should always look like this:
	//
	//    func (p *Processor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	//      defer func() {
	//      	// io.LedgerReader.Close() returns error if upgrade changes have not
	//      	// been processed so it's worth checking the error.
	//      	closeErr := r.Close()
	//      	// Do not overwrite the previous error
	//      	if err == nil {
	//      		err = closeErr
	//      	}
	//      }()
	//    	defer w.Close()
	//
	//    	// Some pre code...
	//
	//    	for {
	//    		entry, err := r.Read()
	//    		if err != nil {
	//    			if err == io.EOF {
	//    				break
	//    			} else {
	//    				return errors.Wrap(err, "Error reading from LedgerReader in [ProcessorName]")
	//    			}
	//    		}
	//
	//    		// Process entry...
	//
	//    		// Write to LedgerWriter if needed but exit if pipe is closed:
	//    		err = w.Write(entry)
	//    		if err != nil {
	//    			if err == io.ErrClosedPipe {
	//    				//    Reader does not need more data
	//    				return nil
	//    			}
	//    			return errors.Wrap(err, "Error writing to LedgerWriter in [ProcessorName]")
	//    		}
	//
	//    		// Return errors if needed...
	//
	//    		// Exit when pipeline terminated due to an error in another processor...
	//    		select {
	//    		case <-ctx.Done():
	//    			return nil
	//    		default:
	//    			continue
	//    		}
	//    	}
	//
	//    	for {
	//    		change, err := r.ReadUpgradeChange()
	//    		if err != nil {
	//    			if err == stdio.EOF {
	//    				break
	//    		} else {
	//    			return err
	//    		}
	//
	//    		// Process ledger upgrade change...
	//    	}
	//
	//    	// Some post code...
	//
	//    	return nil
	//    }
	ProcessLedger(context.Context, *supportPipeline.Store, io.LedgerReader, io.LedgerWriter) error
	// Returns processor name. Helpful for errors, debuging and reports.
	Name() string
	// Reset resets internal state of the processor. This is run by the pipeline
	// everytime the processing is done. It is extremely important to implement
	// this method, otherwise internal state of the processor will be maintained
	// between pipeline runs and may result in invalid data.
	Reset()
}

// stateProcessorWrapper wraps StateProcessor to implement pipeline.Processor interface.
type stateProcessorWrapper struct {
	StateProcessor
}

var _ supportPipeline.Processor = &stateProcessorWrapper{}

// ledgerProcessorWrapper wraps LedgerProcessor to implement pipeline.Processor interface.
type ledgerProcessorWrapper struct {
	LedgerProcessor
}

var _ supportPipeline.Processor = &ledgerProcessorWrapper{}

// stateReaderWrapper wraps StateReader to implement pipeline.Reader interface.
type stateReaderWrapper struct {
	io.StateReader
}

var _ supportPipeline.Reader = &stateReaderWrapper{}

// ledgerReaderWrapper wraps LedgerReader to implement pipeline.Reader interface.
type ledgerReaderWrapper struct {
	io.LedgerReader
}

var _ supportPipeline.Reader = &ledgerReaderWrapper{}

// readerWrapperState wraps pipeline.Reader to implement StateReader interface.
type readerWrapperState struct {
	supportPipeline.Reader
}

var _ io.StateReader = &readerWrapperState{}

// readerWrapperLedger wraps pipeline.Reader to implement LedgerReader interface.
type readerWrapperLedger struct {
	supportPipeline.Reader

	upgradeChanges []io.Change
	// currentUpgrade points to the upgrade to be read by `ReadUpgradeChange`.
	currentUpgradeChange    int
	readUpgradeChangeCalled bool
	ignoreUpgradeChanges    bool
}

var _ io.LedgerReader = &readerWrapperLedger{}

// writerWrapperState wraps pipeline.Writer to implement StateWriter interface.
type writerWrapperState struct {
	supportPipeline.Writer
}

var _ io.StateWriter = &writerWrapperState{}

// writerWrapperLedger wraps pipeline.Writer to implement LedgerWriter interface.
type writerWrapperLedger struct {
	supportPipeline.Writer
}

var _ io.LedgerWriter = &writerWrapperLedger{}
