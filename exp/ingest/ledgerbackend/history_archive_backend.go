package ledgerbackend

import (
	"context"
	"io"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// HistoryArchiveBackend is using history archives to get ledger data.
// Please remember that:
// * history archives do not contain meta changes so meta fields in
//   LedgerCloseMeta will be empty,
// * history archives should not be trusted: ledger data is not signed!
type HistoryArchiveBackend struct {
	archive historyarchive.ArchiveInterface

	rangeFrom uint32
	rangeTo   uint32
	cache     map[uint32]*LedgerCloseMeta
}

func NewHistoryArchiveBackendFromURL(archiveURL string) (*HistoryArchiveBackend, error) {
	archive, err := historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{Context: context.Background()},
	)
	if err != nil {
		return nil, err
	}

	return &HistoryArchiveBackend{
		archive: archive,
		cache:   make(map[uint32]*LedgerCloseMeta),
	}, nil
}

func NewHistoryArchiveBackendFromArchive(archive historyarchive.ArchiveInterface) *HistoryArchiveBackend {
	return &HistoryArchiveBackend{
		archive: archive,
		cache:   make(map[uint32]*LedgerCloseMeta),
	}
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present
// in the history archives.
func (hab *HistoryArchiveBackend) GetLatestLedgerSequence() (uint32, error) {
	has, err := hab.archive.GetRootHAS()
	if err != nil {
		return 0, errors.Wrap(err, "could not get root HAS")
	}

	return has.CurrentLedger, nil
}

// GetLedger returns the LedgerCloseMeta for the given ledger sequence number.
// The first returned value is false when the ledger does not exist in the history archives.
// Due to the history archives architecture the first request to get a ledger is slow because
// it downloads the data for all the ledgers for a given checkpoint. The following requests
// for other ledgers within the same checkpoint are fast, requesting another checkpoint is
// slow again.
func (hab *HistoryArchiveBackend) GetLedger(sequence uint32) (bool, LedgerCloseMeta, error) {
	if !(sequence >= hab.rangeFrom && sequence <= hab.rangeTo) {
		checkpointSequence := (sequence/64)*64 + 64 - 1
		found, err := hab.loadTransactionsFromCheckpoint(checkpointSequence)
		if err != nil {
			return false, LedgerCloseMeta{}, err
		}
		if !found {
			return false, LedgerCloseMeta{}, nil
		}
	}

	meta := hab.cache[sequence]
	if meta == nil {
		return false, LedgerCloseMeta{}, errors.New("checkpoint loaded but ledger not found")
	}
	return true, *meta, nil
}

func (hab *HistoryArchiveBackend) loadTransactionsFromCheckpoint(checkpointSequence uint32) (bool, error) {
	hab.rangeFrom = 0
	hab.rangeTo = 0

	ledgerExists, err := hab.archive.CategoryCheckpointExists("ledger", checkpointSequence)
	if err != nil {
		return false, errors.Wrap(err, "error checking if ledger category exists")
	}

	transactionsExists, err := hab.archive.CategoryCheckpointExists("transactions", checkpointSequence)
	if err != nil {
		return false, errors.Wrap(err, "error checking if transactions category exists")
	}

	resultsExists, err := hab.archive.CategoryCheckpointExists("results", checkpointSequence)
	if err != nil {
		return false, errors.Wrap(err, "error checking if results category exists")
	}

	if !ledgerExists && !transactionsExists && !resultsExists {
		return false, nil
	} else if !(ledgerExists && transactionsExists && resultsExists) {
		return false, errors.New("history archive broken, some categories does not exist")
	}

	// Ledger
	ledgerPath := historyarchive.CategoryCheckpointPath("ledger", checkpointSequence)
	xdrStream, err := hab.archive.GetXdrStream(ledgerPath)
	if err != nil {
		return false, errors.Wrap(err, "error opening ledger stream")
	}
	defer xdrStream.Close()

	for {
		var header xdr.LedgerHeaderHistoryEntry
		err = xdrStream.ReadOne(&header)
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, errors.Wrap(err, "error reading from ledger stream")
		}
		hab.cache[uint32(header.Header.LedgerSeq)] = &LedgerCloseMeta{LedgerHeader: header}
	}

	// Transactions
	transactionsPath := historyarchive.CategoryCheckpointPath("transactions", checkpointSequence)
	xdrStream, err = hab.archive.GetXdrStream(transactionsPath)
	if err != nil {
		return false, errors.Wrap(err, "error opening transactions stream")
	}
	defer xdrStream.Close()

	for {
		var transactionHistoryEntry xdr.TransactionHistoryEntry
		err = xdrStream.ReadOne(&transactionHistoryEntry)
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, errors.Wrap(err, "error reading from transactions stream")
		}
		hab.cache[uint32(transactionHistoryEntry.LedgerSeq)].TransactionEnvelope = transactionHistoryEntry.TxSet.Txs
	}

	// Results
	resultsPath := historyarchive.CategoryCheckpointPath("results", checkpointSequence)
	xdrStream, err = hab.archive.GetXdrStream(resultsPath)
	if err != nil {
		return false, errors.Wrap(err, "error opening results stream")
	}
	defer xdrStream.Close()

	for {
		var transactionHistoryResultEntry xdr.TransactionHistoryResultEntry
		err = xdrStream.ReadOne(&transactionHistoryResultEntry)
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, errors.Wrap(err, "error reading from results stream")
		}
		hab.cache[uint32(transactionHistoryResultEntry.LedgerSeq)].TransactionResult = transactionHistoryResultEntry.TxResultSet.Results
	}

	hab.rangeFrom = checkpointSequence - 64
	hab.rangeTo = checkpointSequence

	return true, nil
}

// Close clears and resets internal state.
func (hab *HistoryArchiveBackend) Close() error {
	hab.rangeFrom = 0
	hab.rangeTo = 0
	hab.cache = make(map[uint32]*LedgerCloseMeta)
	return nil
}
