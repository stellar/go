package expingest

import (
	"github.com/stellar/go/exp/ingest/io"
	logpkg "github.com/stellar/go/support/log"
)

// loggerStateReader extends io.StateReader with logging capabilities.
//
type loggerStateReader struct {
	io.StateReader
	logger     *logpkg.Entry
	entryCount int
	// how often should the logger report
	frequency int
}

func newLoggerStateReader(reader io.StateReader, logger *logpkg.Entry, every int) *loggerStateReader {
	return &loggerStateReader{
		StateReader: reader,
		logger:      logger,
		frequency:   every,
	}
}

// Ensure loggerStateReader implements StateReader
var _ io.StateReader = &loggerStateReader{}

// Read returns a new ledger entry change on each call, returning io.EOF when the stream ends.
func (lsr *loggerStateReader) Read() (io.Change, error) {
	change, err := lsr.StateReader.Read()

	if err == nil {
		lsr.entryCount++

		if lsr.entryCount%lsr.frequency == 0 {
			lsr.logger.WithField("ledger", lsr.GetSequence()).
				WithField("numEntries", lsr.entryCount).
				Info("Processing entries from History Archive Snapshot")
		}
	}

	return change, err
}
