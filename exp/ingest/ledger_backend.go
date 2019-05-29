package ingest

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const latestLedgerSeqQuery = "select ledgerseq, closetime from ledgerheaders order by ledgerseq desc limit 1"
const txHistoryQuery = "select txbody, txresult, txmeta from txhistory where ledgerseq = "
const ledgerHeaderQuery = "select ledgerhash, data from ledgerheaders where ledgerseq = "
const txFeeHistoryQuery = "select txchanges from txfeehistory where ledgerseq = "

// LedgerBackend represents the interface to a ledger data store.
type LedgerBackend interface {
	GetLatestLedgerSequence() (sequence uint32, err error)
	// The first returned value is false when the ledger does not exist in a backend.
	GetLedger(sequence uint32) (bool, LedgerCloseMeta, error)
}

// Ensure DatabaseBackend implements LedgerBackend
var _ LedgerBackend = (*DatabaseBackend)(nil)

// DatabaseBackend implements a database data store.
type DatabaseBackend struct {
	session db.Session
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the database.
func (dbb *DatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	if dbb.session == (db.Session{}) {
		return 0, errors.New("no session configured - call CreateSesion() first")
	}

	var ledger []LedgerHeader
	err := dbb.session.SelectRaw(&ledger, latestLedgerSeqQuery)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't select ledger sequence")
	}

	log.Infof("latest ledger is %d, closed at %s (%d)", ledger[0].LedgerSeq,
		time.Unix(ledger[0].CloseTime, 0), ledger[0].CloseTime)

	return ledger[0].LedgerSeq, nil
}

// GetLedger returns the LedgerCloseMeta for the given ledger sequence number.
// The first returned value is false when the ledger does not exist in the database.
func (dbb *DatabaseBackend) GetLedger(sequence uint32) (bool, LedgerCloseMeta, error) {
	if dbb.session == (db.Session{}) {
		return false, LedgerCloseMeta{}, errors.New("no session configured - call CreateSesion() first")
	}

	// Check whether ledger is available
	latest, err := dbb.GetLatestLedgerSequence()
	if err != nil {
		return false, LedgerCloseMeta{}, err
	}
	if latest < sequence {
		return false, LedgerCloseMeta{}, nil
	}

	// Query - ledgerheader
	var lRows []LedgerHeaderHistory

	ledgerHeaderQ := ledgerHeaderQuery + fmt.Sprintf("%d", sequence)
	err = dbb.session.SelectRaw(&lRows, ledgerHeaderQ)
	// Return errors, otherwise data
	if err != nil {
		return false, LedgerCloseMeta{}, err
	}

	lcm := LedgerCloseMeta{}

	lcm.LedgerHeader = xdr.LedgerHeaderHistoryEntry{
		Hash:   lRows[0].Hash,
		Header: lRows[0].Header,
		Ext:    xdr.LedgerHeaderHistoryEntryExt{},
	}

	// Query - txhistory
	var txhRows []TXHistory
	txHistoryQ := txHistoryQuery + fmt.Sprintf("%d", sequence)
	err = dbb.session.SelectRaw(&txhRows, txHistoryQ)
	// Return errors, otherwise data
	if err != nil {
		return false, lcm, err
	}

	for _, tx := range txhRows {
		lcm.TransactionEnvelope = append(lcm.TransactionEnvelope, tx.TXBody)
		lcm.TransactionResult = append(lcm.TransactionResult, tx.TXResult)
		lcm.TransactionMeta = append(lcm.TransactionMeta, tx.TXMeta)
	}

	// Query - txfeehistory
	var txfhRows []TXFeeHistory
	txFeeHistoryQ := txFeeHistoryQuery + fmt.Sprintf("%d", sequence)
	err = dbb.session.SelectRaw(&txfhRows, txFeeHistoryQ)
	// Return errors, otherwise data
	if err != nil {
		return false, lcm, err
	}

	for _, tx := range txfhRows {
		lcm.TransactionFeeChanges = append(lcm.TransactionFeeChanges, tx.TXChanges)
	}

	return true, lcm, nil
}

// CreateSession returns a new db.Session that connects to the given DB settings.
func (dbb *DatabaseBackend) CreateSession(driverName, dataSourceName string) error {
	var session db.Session

	dbconn, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return err
	}

	session.DB = dbconn
	dbb.session = session

	return nil
}

// Close disconnects an active database session.
func (dbb *DatabaseBackend) Close() error {
	return dbb.session.DB.Close()
}

// LedgerCloseMeta is the information needed to reconstruct the history of transactions in a given ledger.
type LedgerCloseMeta struct {
	LedgerHeader          xdr.LedgerHeaderHistoryEntry
	TransactionEnvelope   []xdr.TransactionEnvelope
	TransactionResult     []xdr.TransactionResultPair
	TransactionMeta       []xdr.TransactionMeta
	TransactionFeeChanges []xdr.LedgerEntryChanges
}

// LedgerHeaderHistory is a helper struct used to unmarshall header fields from a stellar-core DB.
type LedgerHeaderHistory struct {
	Hash   xdr.Hash         `db:"ledgerhash"`
	Header xdr.LedgerHeader `db:"data"`
}

// TODO: Could use horizon/internal/db2/core/main core.LedgerHeader etc. after refactoring

// LedgerHeader holds a row of data from the stellar-core `ledgerheaders` table.
type LedgerHeader struct {
	LedgerHash     string           `db:"ledgerhash"`
	PrevHash       string           `db:"prevhash"`
	BucketListHash string           `db:"bucketlisthash"`
	CloseTime      int64            `db:"closetime"`
	LedgerSeq      uint32           `db:"ledgerseq"`
	Data           xdr.LedgerHeader `db:"data"`
}

// TXHistory holds a row of data from the stellar-core `txhistory` table.
type TXHistory struct {
	TXID      string                    `db:"txid"`
	LedgerSeq uint32                    `db:"ledgerseq"`
	TXIndex   uint32                    `db:"txindex"`
	TXBody    xdr.TransactionEnvelope   `db:"txbody"`
	TXResult  xdr.TransactionResultPair `db:"txresult"`
	TXMeta    xdr.TransactionMeta       `db:"txmeta"`
}

// TXFeeHistory holds a row of data from the stellar-core `txfeehistory` table.
type TXFeeHistory struct {
	TXID      string                 `db:"txid"`
	LedgerSeq uint32                 `db:"ledgerseq"`
	TXIndex   uint32                 `db:"txindex"`
	TXChanges xdr.LedgerEntryChanges `db:"txchanges"`
}

// SCPHistory holds a row of data from the stellar-core `scphistory` table.
type SCPHistory struct {
	NodeID    string `db:"nodeid"`
	LedgerSeq uint32 `db:"ledgerseq"`
	Envelope  string `db:"envelope"`
}

// UpgradeHistory holds a row of data from the stellar-core `upgradehistory` table.
type UpgradeHistory struct {
	LedgerSeq    uint32 `db:"ledgerseq"`
	UpgradeIndex uint32 `db:"upgradeindex"`
	Upgrade      string `db:"upgrade"`
	Changes      string `db:"changes"`
}
