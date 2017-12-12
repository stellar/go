package ingest

import (
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	herr "github.com/stellar/go/services/horizon/internal/errors"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/log"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	err2 "github.com/stellar/go/support/errors"
)

// ClearAll removes all previously ingested historical data from the horizon
// database.
func (i *System) ClearAll() error {

	hdb := i.HorizonDB.Clone()
	ingestion := &Ingestion{DB: hdb}

	err := ingestion.Start()
	if err != nil {
		return errors.Wrap(err, "failed to begin ingestion")
	}

	err = ingestion.ClearAll()
	if err != nil {
		return errors.Wrap(err, "failed to clear history tables")
	}

	err = ingestion.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close ingestion")
	}

	log.Infof("cleared all history")

	return nil
}

// RebaseHistory re-establishes horizon's history database using the provided
// sequence as a starting point.
func (i *System) RebaseHistory(sequence int32) error {

	err := i.ClearAll()
	if err != nil {
		return errors.Wrap(err, "failed to  clear db")
	}

	err = i.ReingestSingle(sequence)
	if err != nil {
		return errors.Wrap(err, "failed to reingest new base")
	}

	return nil
}

// ReingestAll re-ingests all ledgers
func (i *System) ReingestAll() (int, error) {

	err := i.trimAbandondedLedgers()
	if err != nil {
		return 0, err
	}

	var coreElder int32
	var coreLatest int32
	cq := core.Q{Session: i.CoreDB}

	err = cq.ElderLedger(&coreElder)
	if err != nil {
		return 0, errors.Wrap(err, "load core elder ledger failed")
	}

	err = cq.LatestLedger(&coreLatest)
	if err != nil {
		return 0, errors.Wrap(err, "load core elder ledger failed")
	}

	log.
		WithField("start", coreElder).
		WithField("end", coreLatest).
		Info("reingest: all")

	return i.ReingestRange(coreElder, coreLatest)
}

// ReingestOutdated finds old ledgers and reimports them.
func (i *System) ReingestOutdated() (n int, err error) {

	q := history.Q{Session: i.HorizonDB}

	err = i.trimAbandondedLedgers()
	if err != nil {
		return
	}

	// NOTE: this loop will never terminate if some bug were cause a ledger
	// reingestion to silently fail.
	for {
		outdated := []int32{}
		err = q.OldestOutdatedLedgers(&outdated, CurrentVersion)
		if err != nil {
			return
		}

		if len(outdated) == 0 {
			return
		}

		log.
			WithField("lowest_sequence", outdated[0]).
			WithField("batch_size", len(outdated)).
			Info("reingest: outdated")

		var start, end int32
		flush := func() error {
			ingested, ferr := i.ReingestRange(start, end)

			if ferr != nil {
				return ferr
			}
			n += ingested
			return nil
		}

		for idx := range outdated {
			seq := outdated[idx]

			if start == 0 {
				start = seq
				end = seq
				continue
			}

			if seq == end+1 {
				end = seq
				continue
			}

			err = flush()
			if err != nil {
				return
			}

			start = seq
			end = seq
		}

		err = flush()
		if err != nil {
			return
		}
	}
}

// ReingestRange reingests a range of ledgers, from `start` to `end`, inclusive.
func (i *System) ReingestRange(start, end int32) (int, error) {
	is := NewSession(i)
	is.Cursor = NewCursor(start, end, i)
	is.ClearExisting = true

	is.Run()
	log.WithField("start", start).
		WithField("end", end).
		WithField("err", is.Err).
		WithField("ingested", is.Ingested).
		Info("ingest: range complete")
	return is.Ingested, is.Err
}

// ReingestSingle re-ingests a single ledger
func (i *System) ReingestSingle(sequence int32) error {
	_, err := i.ReingestRange(sequence, sequence)
	return err
}

// Tick triggers the ingestion system to ingest any new ledger data, provided
// that there currently is not an import session in progress.
func (i *System) Tick() *Session {
	i.lock.Lock()
	if i.current != nil {
		log.Info("ingest: already in progress")
		i.lock.Unlock()
		return nil
	}

	is := NewSession(i)
	i.current = is
	i.lock.Unlock()

	i.runOnce()
	return is
}

// newCursor creates a new ingestion cursor that reflects the
// current cached ledger state.
func (i *System) newCursor() (*Cursor, error) {
	ls := ledger.CurrentState()

	// If we already have ingested data, start from the next ingestable ledger,
	// end with the newest closed ledger.
	if ls.HistoryLatest != 0 {
		return NewCursor(ls.HistoryLatest+1, ls.CoreLatest, i), nil
	}

	// Since we've found out the history db is empty (i.e. ls.HistoryLatest == 0),
	// we now find what the earliest ledger we can import is.

	var start int32

	// If horizon is configured to only retain a certain number of ledgers, use
	// that retention number to guess at the new start point.
	if i.HistoryRetentionCount > 0 {
		err := i.CoreDB.GetRaw(&start, `
			SELECT ledgerseq FROM ledgerheaders WHERE ledgerseq > ?
		`, ls.CoreLatest-int32(i.HistoryRetentionCount))

		if err != nil {
			return nil, errors.Wrap(err, "failed to find session start")
		}
	} else {
		err := i.CoreDB.GetRaw(&start, `
			SELECT ledgerseq FROM ledgerheaders WHERE ledgerseq > ? LIMIT 1 
		`, 1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load first ledger after genesis")
		}
	}

	// HACK (scott): we just start with ledger 2 because we can't performantly
	// detect the contiguous range of history that stellar-core has available when
	// the database is large. Until the ingestion system has support for
	// backfilling history, it is advised that you manually establish your
	// ingestion point by using the `horizon db rebase` command with a single
	// specified ledger number to start from.  If a horizon database is empty and
	// the connected stellar-core is not configured for CATCHUP_COMPLETE, trying
	// to import from ledger 1 will cause an ingestion deadlock when the system
	// hits the first gap.
	//
	// HACK (scott):  we don't start with ledger 1 because stellar-core will
	// always write the genesis ledger, even prior to attempting to sync with the
	// network.  While this behavior is can be considered a bug, the genesis
	// ledger is usually pretty boring.  By adopting this behavior for the near
	// term we can get a quick performance win to buy us some time to implement
	// ingestion backfill.
	return NewCursor(start, ls.CoreLatest, i), nil
}

// run causes the importer to check stellar-core to see if we can import new
// data.
func (i *System) runOnce() {
	defer func() {
		if rec := recover(); rec != nil {
			err := herr.FromPanic(rec)
			log.Errorf("import session panicked: %s", err)
			errors.ReportToSentry(err, nil)
		}
	}()

	ls := ledger.CurrentState()

	// 1. stash a copy of the current ingestion session (assigned from the tick)
	// 2. output "initial ingestion" message if the db is empty, otherwise
	//    validate the ledger chain.
	// 3. import until none available

	// 1.
	i.lock.Lock()
	is := i.current
	i.lock.Unlock()

	defer func() {
		i.lock.Lock()
		i.current = nil
		i.lock.Unlock()
	}()

	if is == nil {
		log.Warn("ingest: runOnce ran with a nil current session")
		return
	}

	cursor, err := i.newCursor()
	if err != nil {
		log.Errorf("failed to create ingestion cursor: %s", err)
		return
	}
	is.Cursor = cursor

	if is.Cursor.FirstLedger > is.Cursor.LastLedger {
		// NOTE: this occurs when horizon is synced with the connected stellar-core
		return
	}

	// 2.
	if ls.HistoryLatest == 0 {
		log.Infof(
			"history db is empty, starting ingestion from ledger %d",
			is.Cursor.FirstLedger,
		)
	} else {

		err := i.validateLedgerChain(is.Cursor.FirstLedger)
		if err != nil {
			log.
				WithField("start", is.Cursor.FirstLedger).
				Errorf("ledger gap detected (possible db corruption): %s", err)
			return
		}
	}

	// 3.
	is.Run()

	if is.Err != nil {
		log.Errorf("import session failed: %s", is.Err)
	}

	return
}

// trimAbandondedLedgers deletes all "abandonded" ledgers from the history
// database. An abandonded ledger, in this context, means a ledger known to
// horizon but is no longer present in the stellar-core database source.  The
// usual cause for this situation is a stellar-core that uses the CATCHUP_RECENT
// mode.
func (i *System) trimAbandondedLedgers() error {
	var coreElder int32
	cq := core.Q{Session: i.CoreDB}

	err := cq.ElderLedger(&coreElder)
	if err != nil {
		return errors.Wrap(err, "load core elder ledger failed")
	}

	hdb := i.HorizonDB.Clone()
	ingestion := &Ingestion{DB: hdb}

	err = ingestion.Start()
	if err != nil {
		return errors.Wrap(err, "failed to begin ingestion")
	}

	end := toid.New(coreElder, 0, 0)

	err = ingestion.Clear(0, end.ToInt64())
	if err != nil {
		return errors.Wrap(err, "failed to clear ingestion")
	}

	err = ingestion.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close ingestion")
	}

	log.
		WithField("new_elder_ledger", coreElder).
		Infof("reingest: abandonded ledgers trimmed")

	return nil
}

// validateLedgerChain helps to ensure the chain of ledger entries is contiguous
// within horizon.  It ensures the ledger at `seq` is a child of `seq - 1`.
func (i *System) validateLedgerChain(seq int32) error {
	var (
		cur  core.LedgerHeader
		prev core.LedgerHeader
	)

	q := &core.Q{Session: i.CoreDB}

	err := q.LedgerHeaderBySequence(&cur, seq)
	if err != nil {
		return err2.Wrap(err, "validateLedgerChain: failed to load cur ledger")
	}

	err = q.LedgerHeaderBySequence(&prev, seq-1)
	if err != nil {
		return err2.Wrap(err, "validateLedgerChain: failed to load prev ledger")
	}

	if cur.PrevHash != prev.LedgerHash {
		return err2.New("cur and prev ledger hashes don't match")
	}

	return nil
}
