package ledgerbackend

import "github.com/stellar/go/xdr"

// LedgerBackend represents the interface to a ledger data store.
type LedgerBackend interface {
	GetLatestLedgerSequence() (sequence uint32, err error)
	// The first returned value is false when the ledger does not exist in a backend.
	GetLedger(sequence uint32) (bool, LedgerCloseMeta, error)
	Close() error
}

// LedgerCloseMeta is the information needed to reconstruct the history of transactions in a given ledger.
type LedgerCloseMeta struct {
	LedgerHeader          xdr.LedgerHeaderHistoryEntry
	TransactionEnvelope   []xdr.TransactionEnvelope
	TransactionResult     []xdr.TransactionResultPair
	TransactionMeta       []xdr.TransactionMeta
	TransactionIndex      []uint32
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
