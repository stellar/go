package ledgerbackend

import (
	"database/sql"
	"sync"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const (
	latestLedgerSeqQuery = "select ledgerseq, closetime from ledgerheaders order by ledgerseq desc limit 1"
	txHistoryQuery       = "select txbody, txresult, txmeta, txindex from txhistory where ledgerseq = ? "
	ledgerHeaderQuery    = "select ledgerhash, data from ledgerheaders where ledgerseq = ? "
	txFeeHistoryQuery    = "select txchanges, txindex from txfeehistory where ledgerseq = ? "
	orderBy              = "order by txindex asc"
	dbDriver             = "postgres"
)

// Ensure DatabaseBackend implements LedgerBackend
var _ LedgerBackend = (*DatabaseBackend)(nil)

// DatabaseBackend implements a database data store.
type DatabaseBackend struct {
	DataSourceName string
	session        session
	initOnce       sync.Once
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the database.
func (dbb *DatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	var err error
	dbb.initOnce.Do(func() { err = dbb.init() })
	if err != nil {
		return 0, err
	}

	var ledger []ledgerHeader
	err = dbb.session.SelectRaw(&ledger, latestLedgerSeqQuery)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't select ledger sequence")
	}

	return ledger[0].LedgerSeq, nil
}

// GetLedger returns the LedgerCloseMeta for the given ledger sequence number.
// The first returned value is false when the ledger does not exist in the database.
func (dbb *DatabaseBackend) GetLedger(sequence uint32) (bool, LedgerCloseMeta, error) {
	var err error
	dbb.initOnce.Do(func() { err = dbb.init() })
	if err != nil {
		return false, LedgerCloseMeta{}, err
	}

	lcm := LedgerCloseMeta{}

	// Query - ledgerheader
	var lRow ledgerHeaderHistory

	err = dbb.session.GetRaw(&lRow, ledgerHeaderQuery, sequence)
	// Return errors...
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			// Ledger was not found
			return false, LedgerCloseMeta{}, nil
		default:
			return false, LedgerCloseMeta{}, errors.Wrap(err, "Error getting ledger header")
		}
	}

	// ...otherwise store the header
	lcm.LedgerHeader = xdr.LedgerHeaderHistoryEntry{
		Hash:   lRow.Hash,
		Header: lRow.Header,
		Ext:    xdr.LedgerHeaderHistoryEntryExt{},
	}

	// Query - txhistory
	var txhRows []txHistory
	err = dbb.session.SelectRaw(&txhRows, txHistoryQuery+orderBy, sequence)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting txHistory")
	}

	// ...otherwise store the data
	for i, tx := range txhRows {
		// Sanity check index. Note that first TXIndex in a ledger is 1
		if i != int(tx.TXIndex)-1 {
			return false, LedgerCloseMeta{}, errors.New("transactions read from DB history table are misordered")
		}

		lcm.TransactionEnvelope = append(lcm.TransactionEnvelope, tx.TXBody)
		lcm.TransactionResult = append(lcm.TransactionResult, tx.TXResult)
		lcm.TransactionMeta = append(lcm.TransactionMeta, tx.TXMeta)
	}

	// Query - txfeehistory
	var txfhRows []txFeeHistory
	err = dbb.session.SelectRaw(&txfhRows, txFeeHistoryQuery+orderBy, sequence)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting txFeeHistory")
	}

	// ...otherwise store the data
	for i, tx := range txfhRows {
		// Sanity check index. Note that first TXIndex in a ledger is 1
		if i != int(tx.TXIndex)-1 {
			return false, LedgerCloseMeta{}, errors.New("transactions read from DB fee history table are misordered")
		}
		lcm.TransactionFeeChanges = append(lcm.TransactionFeeChanges, tx.TXChanges)
	}

	return true, lcm, nil
}

// init sets up the backend for use. It delegates to the specific backend implementation.
func (dbb *DatabaseBackend) init() error {
	return dbb.createSession()
}

// CreateSession returns a new db.Session that connects to the given DB settings.
func (dbb *DatabaseBackend) createSession() error {
	if dbb.DataSourceName == "" {
		return errors.New("missing DatabaseBackend.DataSourceName (e.g. \"postgres://stellar:postgres@localhost:8002/core\")")
	}

	session, err := db.Open(dbDriver, dbb.DataSourceName)
	if err != nil {
		return err
	}

	dbb.session = session

	return nil
}

// Close disconnects an active database session.
func (dbb *DatabaseBackend) Close() error {
	return dbb.session.Close()
}
