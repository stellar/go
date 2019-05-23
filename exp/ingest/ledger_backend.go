package ingest

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/support/db"
)

// CoreSession provides helper methods for making queries against Stellar Core's DB.
type CoreSession struct {
	db.Session
}

const latestLedgerQuery = "select "

type LedgerBackend interface {
	GetLatestLedgerSequence() (uint32, error)
	GetLedger(uint32) (bool, []byte, error)
}

type DatabaseBackend struct {
	uri     string
	session CoreSession
}

func (dbb DatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	// Call DB and find latest sequence number
	// assume connection is already set up
	// create the query
	// execute the query
	// handle errors
	// return the answer
	return 0, fmt.Errorf("Not implemented yet")
}

// CreateSession returns a new CoreSession that connects to the given DB settings.
func CreateSession(driverName, dataSourceName string) (session CoreSession, err error) {
	dbconn, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return
	}

	session.DB = dbconn
	// dbb.session = session
	return
}
