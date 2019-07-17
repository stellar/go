// Package newingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package expingest

import (
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	ilog "github.com/stellar/go/support/log"
)

var log = ilog.DefaultLogger.WithField("service", "expingest")

type Config struct {
	CoreSession       *db.Session
	HistorySession    *db.Session
	HistoryArchiveURL string
	StellarCoreURL    string
}

type System struct {
	session  *ingest.LiveSession
	historyQ *history.Q
}

func NewSystem(config Config) (*System, error) {
	archive, err := createArchive(config.HistoryArchiveURL)
	if err != nil {
		return nil, errors.Wrap(err, "error creating history archive")
	}

	ledgerBackend, err := ledgerbackend.NewDatabaseBackendFromSession(config.CoreSession)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ledger backend")
	}

	historyQ := &history.Q{config.HistorySession}

	session := &ingest.LiveSession{
		Archive:        archive,
		LedgerBackend:  ledgerBackend,
		StatePipeline:  buildStatePipeline(historyQ),
		LedgerPipeline: buildLedgerPipeline(historyQ),
		StellarCoreClient: &stellarcore.Client{
			URL: config.StellarCoreURL,
		},

		StateReporter:  &LoggingStateReporter{Log: log, Interval: 100000},
		LedgerReporter: &LoggingLedgerReporter{Log: log},
	}

	addPipelineHooks(session.StatePipeline, config.HistorySession, session)
	addPipelineHooks(session.LedgerPipeline, config.HistorySession, session)

	return &System{session, historyQ}, nil
}

func (s *System) Run() {
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		panic(err)
	}

	if lastIngestedLedger == 0 {
		log.Info("Starting ingestion system from empty state...")
		err = s.session.Run()
	} else {
		log.WithField("last_ledger", lastIngestedLedger).
			Info("Resuming ingestion system from last processed ledger...")
		err = s.session.Resume(lastIngestedLedger + 1)
	}

	if err != nil {
		panic(err)
	}
}

func (s *System) Shutdown() {
	log.Info("Shutting down ingestion system...")
	s.session.Shutdown()
}

func createArchive(archiveURL string) (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{},
	)
}
