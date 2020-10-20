package ledgerbackend

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/xdr"
)

const ledgersPerCheckpoint = 64

// Range represents a range of ledger sequence numbers.
type Range struct {
	from    uint32
	to      uint32
	bounded bool
}

type jsonRange struct {
	From    uint32 `json:"from"`
	To      uint32 `json:"to"`
	Bounded bool   `json:"bounded"`
}

func (r *Range) UnmarshalJSON(b []byte) error {
	var s jsonRange
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	r.from = s.From
	r.to = s.To
	r.bounded = s.Bounded

	return nil
}

func (r Range) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonRange{
		From:    r.from,
		To:      r.to,
		Bounded: r.bounded,
	})
}

func (r Range) String() string {
	if r.bounded {
		return fmt.Sprintf("[%d,%d]", r.from, r.to)
	}
	return fmt.Sprintf("[%d,latest)", r.from)
}

func (r Range) Contains(other Range) bool {
	if r.bounded && !other.bounded {
		return false
	}
	if r.bounded && other.bounded {
		return r.from <= other.from && r.to >= other.to
	}
	return r.from <= other.from
}

// SingleLedgerRange constructs a bounded range containing a single ledger.
func SingleLedgerRange(ledger uint32) Range {
	return Range{from: ledger, to: ledger, bounded: true}
}

// BoundedRange constructs a bounded range of ledgers with a fixed starting ledger and ending ledger.
func BoundedRange(from uint32, to uint32) Range {
	return Range{from: from, to: to, bounded: true}
}

// BoundedRange constructs a unbounded range of ledgers with a fixed starting ledger.
func UnboundedRange(from uint32) Range {
	return Range{from: from, bounded: false}
}

// LedgerBackend represents the interface to a ledger data store.
type LedgerBackend interface {
	// GetLatestLedgerSequence returns the sequence of the latest ledger available
	// in the backend.
	GetLatestLedgerSequence() (sequence uint32, err error)
	// The first returned value is false when the ledger does not exist in a backend.
	GetLedger(sequence uint32) (bool, xdr.LedgerCloseMeta, error)
	// PrepareRange prepares the given range (including from and to) to be loaded.
	// Some backends (like captive stellar-core) need to initalize data to be
	// able to stream ledgers.
	PrepareRange(ledgerRange Range) error
	// IsPrepared returns true if a given ledgerRange is prepared.
	IsPrepared(ledgerRange Range) (bool, error)
	Close() error
}

// session is the interface needed to access a persistent database session.
// TODO can't use this until we add Close() to the existing db.Session object
type session interface {
	GetRaw(dest interface{}, query string, args ...interface{}) error
	SelectRaw(dest interface{}, query string, args ...interface{}) error
	Close() error
}

// ledgerHeaderHistory is a helper struct used to unmarshall header fields from a stellar-core DB.
type ledgerHeaderHistory struct {
	Hash   xdr.Hash         `db:"ledgerhash"`
	Header xdr.LedgerHeader `db:"data"`
}

// ledgerHeader holds a row of data from the stellar-core `ledgerheaders` table.
type ledgerHeader struct {
	LedgerHash     string           `db:"ledgerhash"`
	PrevHash       string           `db:"prevhash"`
	BucketListHash string           `db:"bucketlisthash"`
	CloseTime      int64            `db:"closetime"`
	LedgerSeq      uint32           `db:"ledgerseq"`
	Data           xdr.LedgerHeader `db:"data"`
}

// txHistory holds a row of data from the stellar-core `txhistory` table.
type txHistory struct {
	TXID      string                    `db:"txid"`
	LedgerSeq uint32                    `db:"ledgerseq"`
	TXIndex   uint32                    `db:"txindex"`
	TXBody    xdr.TransactionEnvelope   `db:"txbody"`
	TXResult  xdr.TransactionResultPair `db:"txresult"`
	TXMeta    xdr.TransactionMeta       `db:"txmeta"`
}

// txFeeHistory holds a row of data from the stellar-core `txfeehistory` table.
type txFeeHistory struct {
	TXID      string                 `db:"txid"`
	LedgerSeq uint32                 `db:"ledgerseq"`
	TXIndex   uint32                 `db:"txindex"`
	TXChanges xdr.LedgerEntryChanges `db:"txchanges"`
}

// scpHistory holds a row of data from the stellar-core `scphistory` table.
// type scpHistory struct {
// 	NodeID    string `db:"nodeid"`
// 	LedgerSeq uint32 `db:"ledgerseq"`
// 	Envelope  string `db:"envelope"`
// }

// upgradeHistory holds a row of data from the stellar-core `upgradehistory` table.
type upgradeHistory struct {
	LedgerSeq    uint32                 `db:"ledgerseq"`
	UpgradeIndex uint32                 `db:"upgradeindex"`
	Upgrade      xdr.LedgerUpgrade      `db:"upgrade"`
	Changes      xdr.LedgerEntryChanges `db:"changes"`
}
