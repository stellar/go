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
	cache     map[uint32]*xdr.LedgerCloseMeta
}

var _ LedgerBackend = (*HistoryArchiveBackend)(nil)

const (
	ledgerCategory       = "ledger"
	transactionsCategory = "transactions"
	resultsCategory      = "results"
)

// NewHistoryArchiveBackendFromURL builds a new HistoryArchiveBackend using
// history archive URL.
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
		cache:   make(map[uint32]*xdr.LedgerCloseMeta),
	}, nil
}

// NewHistoryArchiveBackendFromArchive builds a new HistoryArchiveBackend using
// historyarchive.Archive.
func NewHistoryArchiveBackendFromArchive(archive historyarchive.ArchiveInterface) *HistoryArchiveBackend {
	return &HistoryArchiveBackend{
		archive: archive,
		cache:   make(map[uint32]*xdr.LedgerCloseMeta),
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

// PrepareRange does nothing because of how history archives are structured. Request any
// ledger by using `GetLedger` to download all ledger data within a given checkpoint
func (hab *HistoryArchiveBackend) PrepareRange(ledgerRange Range) error {
	return nil
}

// GetLedger returns the LedgerCloseMeta for the given ledger sequence number.
// The first returned value is false when the ledger does not exist in the history archives.
// Due to the history archives architecture the first request to get a ledger is slow because
// it downloads the data for all the ledgers for a given checkpoint. The following requests
// for other ledgers within the same checkpoint are fast, requesting another checkpoint is
// slow again.
func (hab *HistoryArchiveBackend) GetLedger(sequence uint32) (bool, xdr.LedgerCloseMeta, error) {
	if !(sequence >= hab.rangeFrom && sequence <= hab.rangeTo) {
		checkpointSequence := (sequence/ledgersPerCheckpoint)*ledgersPerCheckpoint + ledgersPerCheckpoint - 1
		found, err := hab.loadTransactionsFromCheckpoint(checkpointSequence)
		if err != nil {
			return false, xdr.LedgerCloseMeta{}, err
		}
		if !found {
			return false, xdr.LedgerCloseMeta{}, nil
		}
	}

	meta := hab.cache[sequence]
	if meta == nil {
		return false, xdr.LedgerCloseMeta{}, errors.New("checkpoint loaded but ledger not found")
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
		return false, errors.New("history archive broken, some categories do not exist")
	}

	// ledger must be fetched first because it initalizes LedgerCloseMeta for
	// a given sequence.
	categories := []string{ledgerCategory, transactionsCategory, resultsCategory}
	for _, category := range categories {
		err = hab.fetchCategory(category, checkpointSequence)
		if err != nil {
			return false, err
		}
	}

	if checkpointSequence >= ledgersPerCheckpoint {
		hab.rangeFrom = checkpointSequence - ledgersPerCheckpoint + 1
	} else {
		hab.rangeFrom = 1
	}
	hab.rangeTo = checkpointSequence

	return true, nil
}

func (hab *HistoryArchiveBackend) fetchCategory(category string, checkpointSequence uint32) error {
	path := historyarchive.CategoryCheckpointPath(category, checkpointSequence)
	xdrStream, err := hab.archive.GetXdrStream(path)
	if err != nil {
		return errors.Wrapf(err, "error opening %s stream", category)
	}
	defer xdrStream.Close()

	for {
		switch category {
		case ledgerCategory:
			var object xdr.LedgerHeaderHistoryEntry
			err = xdrStream.ReadOne(&object)
			hab.cache[uint32(object.Header.LedgerSeq)] = &xdr.LedgerCloseMeta{
				V: 0,
				V0: &xdr.LedgerCloseMetaV0{
					LedgerHeader: object,
				},
			}
		case transactionsCategory:
			var object xdr.TransactionHistoryEntry
			err = xdrStream.ReadOne(&object)
			hab.cache[uint32(object.LedgerSeq)].V0.TxSet = object.TxSet
		case resultsCategory:
			var object xdr.TransactionHistoryResultEntry
			err = xdrStream.ReadOne(&object)
			hab.cache[uint32(object.LedgerSeq)].V0.TxProcessing = make([]xdr.TransactionResultMeta, len(object.TxResultSet.Results))
			for i := range object.TxResultSet.Results {
				hab.cache[uint32(object.LedgerSeq)].V0.TxProcessing[i].Result = object.TxResultSet.Results[i]
			}
		default:
			panic("unknown category")
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrapf(err, "error reading from %s stream", category)
		}
	}

	return nil
}

// Close clears and resets internal state.
func (hab *HistoryArchiveBackend) Close() error {
	hab.rangeFrom = 0
	hab.rangeTo = 0
	hab.cache = make(map[uint32]*xdr.LedgerCloseMeta)
	return nil
}
