package ledgerbackend

import (
	"fmt"
	"sync"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const (
	latestLedgerSeqQuery = "select ledgerseq, closetime from ledgerheaders order by ledgerseq desc limit 1"
	txHistoryQuery       = "select txbody, txresult, txmeta, txindex from txhistory where ledgerseq = "
	ledgerHeaderQuery    = "select ledgerhash, data from ledgerheaders where ledgerseq = "
	txFeeHistoryQuery    = "select txchanges from txfeehistory where ledgerseq = "
	orderBy              = " order by txindex asc"
)

// Ensure DatabaseBackend implements LedgerBackend
var _ LedgerBackend = (*DatabaseBackend)(nil)

// DatabaseBackend implements a database data store.
type DatabaseBackend struct {
	DataSourceName string
	DriverName     string
	session        *db.Session
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

	// Check whether ledger is available
	latest, err := dbb.GetLatestLedgerSequence()
	if err != nil {
		return false, LedgerCloseMeta{}, err
	}
	if latest < sequence {
		return false, LedgerCloseMeta{}, nil
	}

	lcm := LedgerCloseMeta{}

	// Query - ledgerheader
	var lRows []ledgerHeaderHistory

	ledgerHeaderQ := ledgerHeaderQuery + fmt.Sprintf("%d", sequence)
	err = dbb.session.SelectRaw(&lRows, ledgerHeaderQ)
	// Return errors...
	if err != nil {
		return false, LedgerCloseMeta{}, errors.Wrap(err, "Error getting ledger header")
	}

	// ...otherwise store the header
	lcm.LedgerHeader = xdr.LedgerHeaderHistoryEntry{
		Hash:   lRows[0].Hash,
		Header: lRows[0].Header,
		Ext:    xdr.LedgerHeaderHistoryEntryExt{},
	}

	// Query - txhistory
	var txhRows []txHistory
	txHistoryQ := txHistoryQuery + fmt.Sprintf("%d", sequence) + orderBy
	err = dbb.session.SelectRaw(&txhRows, txHistoryQ)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting txHistory")
	}

	// ...otherwise store the data
	for _, tx := range txhRows {
		lcm.TransactionEnvelope = append(lcm.TransactionEnvelope, tx.TXBody)
		lcm.TransactionResult = append(lcm.TransactionResult, tx.TXResult)
		lcm.TransactionMeta = append(lcm.TransactionMeta, tx.TXMeta)
		lcm.TransactionIndex = append(lcm.TransactionIndex, tx.TXIndex)
	}

	// Query - txfeehistory
	var txfhRows []txFeeHistory
	txFeeHistoryQ := txFeeHistoryQuery + fmt.Sprintf("%d", sequence) + orderBy
	err = dbb.session.SelectRaw(&txfhRows, txFeeHistoryQ)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting txFeeHistory")
	}

	// ...otherwise store the data
	for _, tx := range txfhRows {
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
	if dbb.DriverName == "" {
		return errors.New("missing DatabaseBackend.DriverName (e.g. \"postgres\")")
	}
	if dbb.DataSourceName == "" {
		return errors.New("missing DatabaseBackend.DataSourceName (e.g. \"postgres://stellar:postgres@localhost:8002/core\")")
	}

	session, err := db.Open(dbb.DriverName, dbb.DataSourceName)
	if err != nil {
		return err
	}

	dbb.session = session

	return nil
}

// Close disconnects an active database session.
func (dbb *DatabaseBackend) Close() error {
	return dbb.session.DB.Close()
}
