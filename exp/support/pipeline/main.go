package pipeline

import (
	"context"
	"sync"
)

// BufferedReadWriteCloser implements ReadCloser and WriteCloser and acts
// like a pipe. All writes are queued in a buffered channel and are waiting
// to be consumed.
//
// Used internally by Pipeline but also helpful for testing.
type BufferedReadWriteCloser struct {
	initOnce sync.Once

	context context.Context

	// readEntriesMutex protects readEntries variable
	readEntriesMutex sync.Mutex
	readEntries      int

	// writeCloseMutex protects from writing to a closed buffer
	// and wroteEntries variable
	writeCloseMutex sync.Mutex
	wroteEntries    int

	// closeOnce protects from closing buffer twice
	closeOnce sync.Once
	buffer    chan interface{}
	closed    bool
}

type multiWriteCloser struct {
	writers []WriteCloser

	mutex        sync.Mutex
	closeAfter   int
	wroteEntries int
}

type Pipeline struct {
	root *PipelineNode

	runningMutex sync.Mutex
	running      bool
	shutDown     bool

	preProcessingHooks  []func(context.Context) error
	postProcessingHooks []func(context.Context, error) error

	cancelledMutex sync.Mutex
	cancelled      bool
	cancelFunc     context.CancelFunc
}

// PipelineInterface is an interface that defines common pipeline methods
// in structs that embed Pipeline.
type PipelineInterface interface {
	SetRoot(rootProcessor *PipelineNode)
	AddPreProcessingHook(hook func(context.Context) error)
	AddPostProcessingHook(hook func(context.Context, error) error)
	Shutdown()
	PrintStatus()
}

var _ PipelineInterface = &Pipeline{}

type PipelineNode struct {
	// Remember to update reset() method if you ever add a new field to this struct!
	Processor Processor
	Children  []*PipelineNode

	jobs            int
	readEntries     int
	readsPerSecond  int
	queuedEntries   int
	wroteEntries    int
	writesPerSecond int
}

// ReadCloser interface placeholder
type ReadCloser interface {
	// GetContext returns context with values of the current reader. Can be
	// helpful to provide data to structs that wrap `ReadCloser`.
	GetContext() context.Context
	// Read should return next entry. If there are no more
	// entries it should return `io.EOF` error.
	Read() (interface{}, error)
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some entries available so reader can stop
	// streaming them.
	Close() error
}

// WriteCloser interface placeholder
type WriteCloser interface {
	// Write is used to pass entry to the next processor. It can return
	// `io.ErrClosedPipe` when the pipe between processors has been closed meaning
	// that next processor does not need more data. In such situation the current
	// processor can terminate as sending more entries to a `WriteCloser`
	// does not make sense (will not be read).
	Write(interface{}) error
	// Close should be called when there are no more entries
	// to write.
	Close() error
}

// Processor defines methods required by the processing pipeline.
type Processor interface {
	// Process is a main method of `Processor`. It receives `ReadCloser`
	// that contains object passed down the pipeline from the previous procesor. Writes to
	// `WriteCloser` will be passed to the next processor. WARNING! `Process`
	// should **always** call `Close()` on `WriteCloser` when no more object will be
	// written and `Close()` on `ReadCloser` when reading is finished.
	// Data required by following processors (like aggregated data) should be saved in
	// `Store`. Read `Store` godoc to understand how to use it.
	// The first argument `ctx` is a context with cancel. Processor should monitor
	// `ctx.Done()` channel and exit when it returns a value. This can happen when
	// pipeline execution is interrupted, ex. due to an error.
	//
	// Given all information above `Process` should always look like this:
	//
	//    func (p *Processor) Process(ctx context.Context, store *pipeline.Store, r ReadCloser, w WriteCloser) error {
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
	//    				return errors.Wrap(err, "Error reading from ReadCloser in [ProcessorName]")
	//    			}
	//    		}
	//
	//    		// Process entry...
	//
	//    		// Write to WriteCloser if needed but exit if pipe is closed:
	//    		err = w.Write(entry)
	//    		if err != nil {
	//    			if err == io.ErrClosedPipe {
	//    				// Reader does not need more data
	//    				return nil
	//    			}
	//    			return errors.Wrap(err, "Error writing to WriteCloser in [ProcessorName]")
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
	Process(context.Context, *Store, ReadCloser, WriteCloser) error
	// IsConcurrent defines if processing pipeline should start a single instance
	// of the processor or multiple instances. Multiple instances will read
	// from the same ReadCloser and write to the same WriteCloser.
	// Example: the processor can insert entries to a DB in a single job but it
	// probably will be faster with multiple DB writers (especially when you want
	// to do some data conversions before inserting).
	IsConcurrent() bool
	// Returns processor name. Helpful for errors, debuging and reports.
	Name() string
	// Reset resets internal state of the processor. This is run by the pipeline
	// everytime before the pipeline starts running.
	// It is extremely important to implement this method, otherwise internal
	// state of the processor will be maintained between pipeline runs and may
	// result in an invalid data.
	Reset()
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
