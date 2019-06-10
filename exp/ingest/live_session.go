package ingest

import (
	"fmt"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/support/errors"
)

var _ Session = &LiveSession{}

func (s *LiveSession) Run() error {
	s.shutdown = make(chan bool)

	historyAdapter := adapters.MakeHistoryArchiveAdapter(s.Archive)

	latestSequence, err := historyAdapter.GetLatestLedgerSequence()
	if err != nil {
		return errors.Wrap(err, "Error getting the latest ledger sequence")
	}

	latestSequence = uint32(23991935)

	fmt.Printf("Initializing state for ledger=%d\n", latestSequence)

	err = s.initState(historyAdapter, latestSequence)
	if err != nil {
		return errors.Wrap(err, "initState errored")
	}

	return nil
}

// AddPipeline - TODO it should be possible to add multiple pipelines
func (s *LiveSession) AddPipeline(p *pipeline.StatePipeline) {
	s.pipeline = p
}

func (s *LiveSession) Shutdown() {
	close(s.shutdown)
}

func (s *LiveSession) initState(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32) error {
	stateReader, err := historyAdapter.GetState(sequence)
	if err != nil {
		return errors.Wrap(err, "Error getting state from history archive")
	}

	errChan := s.pipeline.Process(stateReader)
	select {
	case err := <-errChan:
		if err != nil {
			return errors.Wrap(err, "State pipeline errored")
		}
	case <-s.shutdown:
		s.pipeline.Shutdown()
	}

	return nil
}
