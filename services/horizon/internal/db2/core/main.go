// Package core contains database record definitions useable for
// reading rows from a Stellar Core db
package core

import (
	"github.com/stellar/go/support/db"
)

// Q is a helper struct on which to hang common queries against a stellar
// core database.
type Q struct {
	*db.Session
}

// ElderLedger represents the oldest "ingestable" ledger known to the
// stellar-core database this ingestion system is communicating with.  Horizon,
// which wants to operate on a contiguous range of ledger data (i.e. free from
// gaps) uses the elder ledger to start importing in the case of an empty
// database.  NOTE:  This current query used is correct, but slow.  Please keep
// this query out of latency sensitive or frequently trafficked code paths.
func (q *Q) ElderLedger(dest *int32) error {
	err := q.GetRaw(dest, `
		SELECT COALESCE(ledgerseq, 0)
		FROM (
			SELECT 
				ledgerseq,
				LAG(ledgerseq, 1) OVER ( ORDER BY ledgerseq) as prev
			FROM ledgerheaders
		) seqs
		WHERE COALESCE(prev, -1) < ledgerseq - 1
		ORDER BY ledgerseq DESC
		LIMIT 1;
	`)

	return err
}

// LatestLedger loads the latest known ledger
func (q *Q) LatestLedger(dest interface{}) error {
	return q.GetRaw(dest, `SELECT COALESCE(MAX(ledgerseq), 0) FROM ledgerheaders`)
}
