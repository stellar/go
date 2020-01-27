package expingest

import (
	"context"
	"time"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
)

func ingestionChangeProcessor(historyQ *history.Q) ChangeProcessor {
	return GroupChangeProcessors(
		&processors.AccountDataProcessor{DataQ: historyQ},
		// TODO add more processors
	)
}

func ingestionLedgerTransactionProcessor(historyQ *history.Q) LedgerTransactionProcessor {
	return GroupLedgerTransactionProcessors(
		&processors.NewTransactionProcessor{TransactionsQ: historyQ},
		// TODO add more processors
	)
}

func ingestHistoryRange(
	ctx context.Context,
	historyQ *history.Q,
	ledgerBackend ledgerbackend.LedgerBackend,
	start, end uint32,
) error {
	ltProcessor := ingestionLedgerTransactionProcessor(historyQ)
	if start > end {
		return errors.Errorf("invalid history range [%v, %v]", start, end)
	}

	// TODO include context in ledger backend / ledger adapter
	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: ledgerBackend,
	}

	for cur := start; cur <= end; cur++ {
		err := RunTransactionProcessorOnLedger(ledgerAdapter, ltProcessor, cur)
		if err != nil {
			return err
		}
	}

	return nil
}

func ingestHAS(
	ctx context.Context,
	historyQ *history.Q,
	archive historyarchive.ArchiveInterface,
	tempSet io.TempSet,
	maxStreamRetries int,
) error {
	changeProcessor := ingestionChangeProcessor(historyQ)
	reader, err := io.NewStateReader(ctx, archive, tempSet, maxStreamRetries)
	if err != nil {
		return errors.Wrap(err, "could not create state reader")
	}

	// TODO setup transaction

	// TODO validate HAS
	err = RunChangeProcessorOnHAS(changeProcessor, reader)
	if err != nil {
		return err
	}

	// TODO commit transaction
	// update ingestion values in DB

	return err
}

func resumeIngestion(
	historyQ *history.Q,
	ledgerBackend ledgerbackend.LedgerBackend,
) error {
	ltProcessor := ingestionLedgerTransactionProcessor(historyQ)
	changeProcessor := ingestionChangeProcessor(historyQ)

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: ledgerBackend,
	}

	for {
		// TODO setup transaction
		lastIngestedLedger, err := historyQ.GetLastLedgerExpIngest()
		if err != nil {
			return errors.Wrap(err, "Error getting last ledger")
		}

		if lastIngestedLedger == 0 {
			return errors.New("expected last ingested ledger to be > 0")
		}

		next := lastIngestedLedger + 1

		err = RunTransactionProcessorOnLedger(ledgerAdapter, ltProcessor, next)
		if err != nil && errors.Cause(err) == io.ErrNotFound {
			// Ensure that there are no gaps. This is "just in case". There shouldn't
			// be any gaps if CURSOR in core is updated and core version is v11.2.0+.
			var latestLedger uint32
			latestLedger, err = ledgerAdapter.GetLatestLedgerSequence()
			if err != nil {
				return err
			}

			if latestLedger > lastIngestedLedger {
				return errors.Errorf(
					"Gap detected (ledger %d does not exist but %d is latest)",
					next,
					latestLedger,
				)
			}

			select {
			case <-time.After(time.Second):
				// TODO make the idle time smaller
			}

			continue
		} else if err != nil {
			return err
		}

		err = RunChangeProcessorOnLedger(ledgerAdapter, changeProcessor, next)
		if err != nil {
			return err
		}

		// TODO commit transaction
	}
}
