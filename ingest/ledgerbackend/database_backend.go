package ledgerbackend

import (
	"database/sql"
	"sort"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const (
	latestLedgerSeqQuery = "select ledgerseq, closetime from ledgerheaders order by ledgerseq desc limit 1"
	txHistoryQuery       = "select txbody, txresult, txmeta, txindex from txhistory where ledgerseq = ? "
	ledgerHeaderQuery    = "select ledgerhash, data from ledgerheaders where ledgerseq = ? "
	txFeeHistoryQuery    = "select txchanges, txindex from txfeehistory where ledgerseq = ? "
	upgradeHistoryQuery  = "select ledgerseq, upgradeindex, upgrade, changes from upgradehistory where ledgerseq = ? order by upgradeindex asc"
	orderBy              = "order by txindex asc"
	dbDriver             = "postgres"
)

// Ensure DatabaseBackend implements LedgerBackend
var _ LedgerBackend = (*DatabaseBackend)(nil)

// DatabaseBackend implements a database data store.
type DatabaseBackend struct {
	networkPassphrase string
	session           session
}

func NewDatabaseBackend(dataSourceName, networkPassphrase string) (*DatabaseBackend, error) {
	session, err := createSession(dataSourceName)
	if err != nil {
		return nil, err
	}

	return &DatabaseBackend{session: session, networkPassphrase: networkPassphrase}, nil
}

func NewDatabaseBackendFromSession(session *db.Session, networkPassphrase string) (*DatabaseBackend, error) {
	return &DatabaseBackend{session: session, networkPassphrase: networkPassphrase}, nil
}

func (dbb *DatabaseBackend) PrepareRange(ledgerRange Range) error {
	fromExists, _, err := dbb.GetLedger(ledgerRange.from)
	if err != nil {
		return errors.Wrap(err, "error getting ledger")
	}

	if !fromExists {
		return errors.New("`from` ledger does not exist")
	}

	if ledgerRange.bounded {
		toExists, _, err := dbb.GetLedger(ledgerRange.to)
		if err != nil {
			return errors.Wrap(err, "error getting ledger")
		}

		if !toExists {
			return errors.New("`to` ledger does not exist")
		}
	}

	return nil
}

// IsPrepared returns true if a given ledgerRange is prepared.
func (*DatabaseBackend) IsPrepared(ledgerRange Range) (bool, error) {
	return true, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the database.
func (dbb *DatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	var ledger []ledgerHeader
	err := dbb.session.SelectRaw(&ledger, latestLedgerSeqQuery)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't select ledger sequence")
	}
	if len(ledger) == 0 {
		return 0, errors.New("no ledgers exist in ledgerheaders table")
	}

	return ledger[0].LedgerSeq, nil
}

func sortByHash(transactions []xdr.TransactionEnvelope, passphrase string) error {
	hashes := make([]xdr.Hash, len(transactions))
	txByHash := map[xdr.Hash]xdr.TransactionEnvelope{}
	for i, tx := range transactions {
		hash, err := network.HashTransactionInEnvelope(tx, passphrase)
		if err != nil {
			return errors.Wrap(err, "cannot hash transaction")
		}
		hashes[i] = hash
		txByHash[hash] = tx
	}

	sort.Slice(hashes, func(i, j int) bool {
		a := hashes[i]
		b := hashes[j]
		for k := range a {
			if a[k] < b[k] {
				return true
			}
			if a[k] > b[k] {
				return false
			}
		}
		return false
	})

	for i, hash := range hashes {
		transactions[i] = txByHash[hash]
	}
	return nil
}

// GetLedger returns the LedgerCloseMeta for the given ledger sequence number.
// The first returned value is false when the ledger does not exist in the database.
func (dbb *DatabaseBackend) GetLedger(sequence uint32) (bool, xdr.LedgerCloseMeta, error) {
	lcm := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{},
	}

	// Query - ledgerheader
	var lRow ledgerHeaderHistory

	err := dbb.session.GetRaw(&lRow, ledgerHeaderQuery, sequence)
	// Return errors...
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			// Ledger was not found
			return false, xdr.LedgerCloseMeta{}, nil
		default:
			return false, xdr.LedgerCloseMeta{}, errors.Wrap(err, "Error getting ledger header")
		}
	}

	// ...otherwise store the header
	lcm.V0.LedgerHeader = xdr.LedgerHeaderHistoryEntry{
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
			return false, xdr.LedgerCloseMeta{}, errors.New("transactions read from DB history table are misordered")
		}

		lcm.V0.TxSet.Txs = append(lcm.V0.TxSet.Txs, tx.TXBody)
		lcm.V0.TxProcessing = append(lcm.V0.TxProcessing, xdr.TransactionResultMeta{
			Result:            tx.TXResult,
			TxApplyProcessing: tx.TXMeta,
		})
	}

	if err = sortByHash(lcm.V0.TxSet.Txs, dbb.networkPassphrase); err != nil {
		return false, xdr.LedgerCloseMeta{}, errors.Wrap(err, "could not sort txset")
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
			return false, xdr.LedgerCloseMeta{}, errors.New("transactions read from DB fee history table are misordered")
		}
		lcm.V0.TxProcessing[i].FeeProcessing = tx.TXChanges
	}

	// Query - upgradehistory
	var upgradeHistoryRows []upgradeHistory
	err = dbb.session.SelectRaw(&upgradeHistoryRows, upgradeHistoryQuery, sequence)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting upgradeHistoryRows")
	}

	// ...otherwise store the data
	lcm.V0.UpgradesProcessing = make([]xdr.UpgradeEntryMeta, len(upgradeHistoryRows))
	for i, upgradeHistoryRow := range upgradeHistoryRows {
		lcm.V0.UpgradesProcessing[i] = xdr.UpgradeEntryMeta{
			Upgrade: upgradeHistoryRow.Upgrade,
			Changes: upgradeHistoryRow.Changes,
		}
	}

	return true, lcm, nil
}

// CreateSession returns a new db.Session that connects to the given DB settings.
func createSession(dataSourceName string) (*db.Session, error) {
	if dataSourceName == "" {
		return nil, errors.New("missing DatabaseBackend.DataSourceName (e.g. \"postgres://stellar:postgres@localhost:8002/core\")")
	}

	return db.Open(dbDriver, dataSourceName)
}

// Close disconnects an active database session.
func (dbb *DatabaseBackend) Close() error {
	return dbb.session.Close()
}
