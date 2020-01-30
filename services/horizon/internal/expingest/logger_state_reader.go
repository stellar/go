package expingest

import (
	"github.com/stellar/go/exp/ingest/io"
	logpkg "github.com/stellar/go/support/log"
)

// loggerStateReader extends io.StateReader with logging capabilities.
//
type loggerStateReader struct {
	io.StateReader
	logger      *logpkg.Entry
	readChanges int
	// how often should the logger report
	every int
}

func newLoggerStateReader(reader io.StateReader, logger *logpkg.Entry, every int) *loggerStateReader {
	return &loggerStateReader{
		StateReader: reader,
		logger:      logger,
		every:       every,
	}
}

// Ensure loggerStateReader implements ChangeReader
var _ io.ChangeReader = &loggerStateReader{}

// Read returns a new ledger entry change on each call, returning io.EOF when the stream ends.
func (lsr *loggerStateReader) Read() (io.Change, error) {
	change, err := lsr.StateReader.Read()

	if err == nil {
		lsr.readChanges++

		if lsr.readChanges%lsr.every == 0 {
			lsr.logger.Infof("Entries processed from HAS: %d", lsr.readChanges)
		}
	}

	return change, err
}
