package ingest

import (
	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// CoreSession provides helper methods for making queries against Stellar Core's DB.
type CoreSession struct {
	db.Session
}

const latestLedgerQuery = "select ledgerseq, closetime from ledgerheaders order by ledgerseq desc limit 1"
const txHistoryQuery = "select * from txhistory limit 10;"

type LedgerBackend interface {
	GetLatestLedgerSequence() (uint32, error)
	GetLedger(uint32) (bool, []byte, error)
}

type DatabaseBackend struct {
	session CoreSession
}

func (dbb *DatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	// TODO: Assumes connection is already set up
	// Call DB and find latest sequence number
	var ledger []Ledger
	err := dbb.session.SelectRaw(&ledger, latestLedgerQuery)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't select ledger sequence")
	}

	return ledger[0].LedgerSeq, nil
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

type TXHistory struct {
	// TODO: Check sizes of ints
	TXID      string `db:"txid"`
	LedgerSeq int    `db:"ledgerseq"`
	TXIndex   int    `db:"txindex"`
	TXBody    string `db:"txbody"`
	TXResult  string `db:"txresult"`
	TXMeta    string `db:"txmeta"`
}

type Ledger struct {
	// TODO: Check sizes of ints
	LedgerSeq uint32 `db:"ledgerseq"`
	CloseTime int    `db:"closetime"`
}
