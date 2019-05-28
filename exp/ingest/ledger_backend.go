package ingest

import (
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// CoreSession provides helper methods for making queries against Stellar Core's DB.
type CoreSession struct {
	db.Session
}

const latestLedgerSeqQuery = "select ledgerseq, closetime from ledgerheaders order by ledgerseq desc limit 1"
const txHistoryQuery = "select * from txhistory limit 10;"
const ledgerHeaderQuery = "select ledgerhash, data from ledgerheaders where ledgerseq = "

type LedgerBackend interface {
	GetLatestLedgerSequence() (sequence uint32, err error)
	// The first returned value is false when the ledger does not exist in a backend
	GetLedger(sequence uint32) (bool, LedgerCloseMeta, error)
}

// Ensure DatabaseBackend implements LedgerBackend
// var _ LedgerBackend = &DatabaseBackend{}
var _ LedgerBackend = (*DatabaseBackend)(nil)

type DatabaseBackend struct {
	session    CoreSession
	lastLedger LedgerHeader
}

func (dbb *DatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	if dbb.session == (CoreSession{}) {
		return 0, errors.New("no session configured - call CreateSesion() first")
	}

	var ledger []LedgerHeader
	err := dbb.session.SelectRaw(&ledger, latestLedgerSeqQuery)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't select ledger sequence")
	}

	log.Infof("latest ledger is %d, closed at %s (%d)", ledger[0].LedgerSeq, time.Unix(ledger[0].CloseTime, 0), ledger[0].CloseTime)

	return ledger[0].LedgerSeq, nil
}

func (dbb *DatabaseBackend) GetLedger(sequence uint32) (bool, LedgerCloseMeta, error) {
	if dbb.session == (CoreSession{}) {
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

	// Query
	// lcm := LedgerCloseMeta{}
	// Get LedgerHeader
	// TODO: Try this out. Probably will need custom deserialising into XDR
	// TODO: Append to create full query
	rows := []LedgerHeaderHistory
	err = dbb.session.SelectRaw(&rows, ledgerHeaderQ)


	// Return errors, otherwise data

	return true, LedgerCloseMeta{}, nil
}

func (dbb *DatabaseBackend) GetTXHistory() (rows []TXHistory, err error) {
	err = dbb.session.SelectRaw(&rows, txHistoryQuery)

	return rows, err
}

// CreateSession returns a new CoreSession that connects to the given DB settings.
func (dbb *DatabaseBackend) CreateSession(driverName, dataSourceName string) error {
	var session CoreSession

	dbconn, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return err
	}

	session.DB = dbconn
	dbb.session = session

	return nil
}

func (dbb *DatabaseBackend) Close() error {
	return dbb.session.DB.Close()
}

type LedgerCloseMeta struct {
	LedgerHeader          xdr.LedgerHeaderHistoryEntry
	Transaction           xdr.Transaction
	TransactionResult     xdr.TransactionResultPair
	TransactionMeta       xdr.TransactionMeta
	TransactionFeeChanges xdr.LedgerEntryChanges
}

type LedgerHeaderHistory struct {
	Hash xdr.Hash           `db:"ledgerhash"`
	Header xdr.LedgerHeader `db:"data"`
}

// LedgerHeader holds a row of data from the `ledgerheaders` table
// TODO: Could use horizon/internal/db2/core/main core.LedgerHeader after refactoring
type LedgerHeader struct {
	LedgerHash     string           `db:"ledgerhash"`
	PrevHash       string           `db:"prevhash"`
	BucketListHash string           `db:"bucketlisthash"`
	CloseTime      int64            `db:"closetime"`
	LedgerSeq      uint32           `db:"ledgerseq"`
	Data           xdr.LedgerHeader `db:"data"`
}

type TXHistory struct {
	TXID      string `db:"txid"`
	LedgerSeq uint32 `db:"ledgerseq"`
	TXIndex   uint32 `db:"txindex"`
	TXBody    string `db:"txbody"`
	TXResult  string `db:"txresult"`
	TXMeta    string `db:"txmeta"`
}

type TXFeeHistory struct {
	TXID      string `db:"txid"`
	LedgerSeq uint32 `db:"ledgerseq"`
	TXIndex   uint32 `db:"txindex"`
	TXChanges string `db:"txchanges"`
}

type SCPHistory struct {
	NodeID    string `db:"nodeid"`
	LedgerSeq uint32 `db:"ledgerseq"`
	Envelope  string `db:"envelope"`
}

type UpgradeHistory struct {
	LedgerSeq    uint32 `db:"ledgerseq"`
	UpgradeIndex uint32 `db:"upgradeindex"`
	Upgrade      string `db:"upgrade"`
	Changes      string `db:"changes"`
}
