package main

import (
	"time"

	"github.com/stellar/go/support/log"
)

// LoggingStateReporter logs the progress of a session running its
// state pipelines
type LoggingStateReporter struct {
	logInterval int
	entryCount  int
	sequence    uint32
	startTime   time.Time
}

// NewLoggingStateReporter constructs a new LoggingStateReporter instance
func NewLoggingStateReporter(logInterval int) *LoggingStateReporter {
	return &LoggingStateReporter{
		logInterval: logInterval,
	}
}

// OnStartState logs that the session has started reading from the history archive snapshot
func (lr *LoggingStateReporter) OnStartState(sequence uint32) {
	log.WithField("sequence", sequence).Info("Reading from History Archive Snapshot")
	lr.entryCount = 0
	lr.sequence = sequence
	lr.startTime = time.Now()
}

// OnStateEntry logs that the session has processed an entry from the history archive snapshot
func (lr *LoggingStateReporter) OnStateEntry() {
	lr.entryCount++
	if lr.entryCount%lr.logInterval == 0 {
		log.WithField("sequence", lr.sequence).
			WithField("numEntries", lr.entryCount).
			Info("Processed entries from History Archive Snapshot")
	}
}

// OnEndState logs that the session has finished processing the history archive snapshot
func (lr *LoggingStateReporter) OnEndState(err error, shutdown bool) {
	elapsedTime := time.Now().Sub(lr.startTime)
	log.WithField("sequence", lr.sequence).
		WithField("numEntries", lr.entryCount).
		WithError(err).
		WithField("shutdown", shutdown).
		WithField("elapsedSeconds", elapsedTime.Seconds).
		Info("Finished processing History Archive Snapshot")
}

// LoggingLedgerReporter logs the progress of a session running its
// ledger pipelines
type LoggingLedgerReporter struct {
	entryCount int
	sequence   uint32
	startTime  time.Time
}

// NewLoggingLedgerReporter constructs a new LoggingLedgerReporter instance
func NewLoggingLedgerReporter() *LoggingLedgerReporter {
	return &LoggingLedgerReporter{}
}

// OnNewLedger logs that the session has started reading a new ledger
func (lr *LoggingLedgerReporter) OnNewLedger(sequence uint32) {
	log.WithField("sequence", sequence).Info("Reading new ledger")
	lr.entryCount = 0
	lr.sequence = sequence
	lr.startTime = time.Now()
}

// OnLedgerTransaction records that the session has processed a transaction from the ledger
func (lr *LoggingLedgerReporter) OnLedgerTransaction() {
	lr.entryCount++
}

// OnEndLedger logs that the session has finished processing the ledger
func (lr *LoggingLedgerReporter) OnEndLedger(err error, shutdown bool) {
	elapsedTime := time.Now().Sub(lr.startTime)
	log.WithField("sequence", lr.sequence).
		WithField("numEntries", lr.entryCount).
		WithError(err).
		WithField("shutdown", shutdown).
		WithField("elapsedSeconds", elapsedTime.Seconds).
		Info("Finished processing ledger")
}
