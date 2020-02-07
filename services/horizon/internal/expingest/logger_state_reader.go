package expingest

import (
	"runtime"

	"github.com/stellar/go/exp/ingest/io"
	logpkg "github.com/stellar/go/support/log"
)

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func getMemStats() (uint64, uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return bToMb(m.HeapAlloc), bToMb(m.HeapSys)
}

// loggerStateReader extends io.StateReader with logging capabilities.
//
type loggerStateReader struct {
	io.StateReader
	logger     *logpkg.Entry
	entryCount int
	// how often should the logger report
	frequency int
	profile   bool
}

func newLoggerStateReader(
	reader io.StateReader,
	logger *logpkg.Entry,
	every int,
	profile bool,
) *loggerStateReader {
	return &loggerStateReader{
		StateReader: reader,
		logger:      logger,
		frequency:   every,
		profile:     profile,
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
			logger := lsr.logger.WithField("ledger", lsr.GetSequence()).
				WithField("numEntries", lsr.entryCount)

			if lsr.profile {
				curHeap, sysHeap := getMemStats()
				logger = logger.
					WithField("currentHeapSizeMB", curHeap).
					WithField("systemHeapSizeMB", sysHeap)
			}

			logger.Info("Processing entries from History Archive Snapshot")
		}
	}

	return change, err
}

type loggerChangeReader struct {
	io.ChangeReader
	logger     *logpkg.Entry
	profile    bool
	entryCount int
	// how often should the logger report
	frequency int
}

func newLoggerChangeReader(
	reader io.ChangeReader,
	logger *logpkg.Entry,
	every int,
	profile bool,
) *loggerChangeReader {
	return &loggerChangeReader{
		ChangeReader: reader,
		logger:       logger,
		frequency:    every,
		profile:      profile,
	}
}

// Read returns a new ledger entry change on each call, returning io.EOF when the stream ends.
func (lcr *loggerChangeReader) Read() (io.Change, error) {
	change, err := lcr.ChangeReader.Read()

	if err == nil {
		lcr.entryCount++

		if lcr.entryCount%lcr.frequency == 0 {
			logger := lcr.logger.WithField("numEntries", lcr.entryCount)

			if lcr.profile {
				curHeap, sysHeap := getMemStats()
				logger = logger.
					WithField("currentHeapSizeMB", curHeap).
					WithField("systemHeapSizeMB", sysHeap)
			}
			logger.Info("Processing entries from ledger")
		}
	}

	return change, err
}
