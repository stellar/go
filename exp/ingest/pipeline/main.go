package pipeline

import (
	"sync"
	"time"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
)

type bufferedStateReadWriteCloser struct {
	initOnce sync.Once

	// readEntriesMutex protects readEntries variable
	readEntriesMutex sync.Mutex
	readEntries      int

	// writeCloseMutex protects from writing to a closed buffer
	// and wroteEntries variable
	writeCloseMutex sync.Mutex
	wroteEntries    int

	// closeOnce protects from closing buffer twice
	closeOnce sync.Once
	buffer    chan xdr.LedgerEntry
	closed    bool
}

type multiWriteCloser struct {
	writers []io.StateWriteCloser

	mutex        sync.Mutex
	closeAfter   int
	wroteEntries int
}

type Pipeline struct {
	rootStateProcessor *PipelineNode
	done               bool
}

type PipelineNode struct {
	Processor StateProcessor
	Children  []*PipelineNode

	duration        time.Duration
	jobs            int
	readEntries     int
	readsPerSecond  int
	queuedEntries   int
	wroteEntries    int
	writesPerSecond int
}

// StateProcessor defines methods required by state processing pipeline.
type StateProcessor interface {
	// ProcessState is a main method of `StateProcessor`. It receives `io.StateReadCloser`
	// that contains object passed down the pipeline from the previous procesor. Writes to
	// `io.StateWriteCloser` will be passed to the next processor. WARNING! `ProcessState`
	// should **always** call `Close()` on `io.StateWriteCloser` when no more object will be
	// written and `Close()` on `io.StateReadCloser` when reading is finished.
	// Data required by following processors (like aggregated data) should be saved in
	// `Store`. Read `Store` godoc to understand how to use it.
	ProcessState(store *Store, readCloser io.StateReadCloser, writeCloser io.StateWriteCloser) (err error)
	// IsConcurrent defines if processing pipeline should start a single instance
	// of the processor or multiple instances. Multiple instances will read
	// from the same StateReader and write to the same StateWriter.
	// Example: the processor can insert entries to a DB in a single job but it
	// probably will be faster with multiple DB writers (especially when you want
	// to do some data conversions before inserting).
	IsConcurrent() bool
	// RequiresInput defines if processor requires input data (StateReader). If not,
	// it will receive empty reader, it's parent process will write to "void" and
	// writes to `writer` will go to "void".
	// This is useful for processors resposible for saving aggregated data that don't
	// need state objects.
	// TODO!
	RequiresInput() bool
	// Returns processor name. Helpful for errors, debuging and reports.
	Name() string
}

// Store allows storing data connected to pipeline execution.
// It exposes `Lock()` and `Unlock()` methods that must be called
// when accessing the `Store` for both `Put` and `Get` calls.
//
// Example (incrementing a number):
// s.Lock()
// v := s.Get("value")
// s.Put("value", v.(int)+1)
// s.Unlock()
type Store struct {
	sync.Mutex
	initOnce sync.Once
	values   map[string]interface{}
}

// ReduceStateProcessor forwards the final produced by applying all the
// ledger entries to the writer.
// Let's say that there are 3 ledger entries:
//     - Create account A (XLM balance = 20)
//     - Update XLM balance of A to 5
//     - Update XLM balance of A to 15
// Running ReduceStateProcessor will add a single ledger entry:
//     - Create account A (XLM balance = 15)
// to the writer.
type ReduceStateProcessor struct {
	//
}
