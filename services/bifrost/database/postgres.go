package database

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/services/bifrost/sse"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

const (
	ethereumAddressIndexKey = "ethereum_address_index"
	ethereumLastBlockKey    = "ethereum_last_block"

	bitcoinAddressIndexKey = "bitcoin_address_index"
	bitcoinLastBlockKey    = "bitcoin_last_block"

	addressAssociationTableName     = "address_association"
	broadcastedEventTableName       = "broadcasted_event"
	keyValueStoreTableName          = "key_value_store"
	processedTransactionTableName   = "processed_transaction"
	transactionsQueueTableName      = "transactions_queue"
	recoveryTransactionTableName    = "recovery_transaction"
	transactionSubmissionsTableName = "transaction_submissions"

	maxProcessingFailureCount = 15
)

type keyValueStoreRow struct {
	Key   string `db:"key"`
	Value string `db:"value"`
}

type broadcastedEventRow struct {
	ID      int64  `db:"id"`
	Address string `db:"address"`
	Event   string `db:"event"`
	Data    string `db:"data"`
}

func (b *broadcastedEventRow) toSSE() sse.Event {
	return sse.Event{
		Address: b.Address,
		Event:   sse.AddressEvent(b.Event),
		Data:    b.Data,
	}
}

type transactionsQueueRow struct {
	TransactionID    string          `db:"transaction_id"`
	AssetCode        queue.AssetCode `db:"asset_code"`
	Amount           string          `db:"amount"`
	StellarPublicKey string          `db:"stellar_public_key"`
	FailureCount     int             `db:"failure_count"`
}

type processedTransactionRow struct {
	Chain            Chain     `db:"chain"`
	TransactionID    string    `db:"transaction_id"`
	ReceivingAddress string    `db:"receiving_address"`
	CreatedAt        time.Time `db:"created_at"`
}

type recoveryTransactionRow struct {
	Source      string `db:"source"`
	EnvelopeXDR string `db:"envelope_xdr"`
}

func fromQueueTransaction(tx queue.Transaction) *transactionsQueueRow {
	return &transactionsQueueRow{
		TransactionID:    tx.TransactionID,
		AssetCode:        tx.AssetCode,
		Amount:           tx.Amount,
		StellarPublicKey: tx.StellarPublicKey,
	}
}

func isDuplicateError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

func (r *transactionsQueueRow) toQueueTransaction() queue.Transaction {
	return queue.Transaction{
		TransactionID:    r.TransactionID,
		AssetCode:        r.AssetCode,
		Amount:           r.Amount,
		StellarPublicKey: r.StellarPublicKey,
	}
}

func (d *PostgresDatabase) Open(dsn string) error {
	var err error
	d.session, err = db.Open("postgres", dsn)
	if err != nil {
		return err
	}

	return nil
}

func (d *PostgresDatabase) getTable(name string, session *db.Session) *db.Table {
	if session == nil {
		session = d.session
	}

	return &db.Table{
		Name:    name,
		Session: session,
	}
}

func (d *PostgresDatabase) CreateAddressAssociation(chain Chain, stellarAddress, address string, addressIndex uint32) error {
	addressAssociationTable := d.getTable(addressAssociationTableName, nil)

	association := &AddressAssociation{
		Chain:            chain,
		AddressIndex:     addressIndex,
		Address:          address,
		StellarPublicKey: stellarAddress,
		CreatedAt:        time.Now(),
	}

	_, err := addressAssociationTable.Insert(association).Exec()
	return err
}

func (d *PostgresDatabase) GetAssociationByChainAddress(chain Chain, address string) (*AddressAssociation, error) {
	addressAssociationTable := d.getTable(addressAssociationTableName, nil)
	row := &AddressAssociation{}
	where := map[string]interface{}{"address": address, "chain": chain}
	err := addressAssociationTable.Get(row, where).Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "Error getting addressAssociation from DB")
		}
	}

	return row, nil
}

func (d *PostgresDatabase) GetAssociationByStellarPublicKey(stellarPublicKey string) (*AddressAssociation, error) {
	addressAssociationTable := d.getTable(addressAssociationTableName, nil)
	row := &AddressAssociation{}
	where := map[string]interface{}{"stellar_public_key": stellarPublicKey}
	err := addressAssociationTable.Get(row, where).Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "Error getting addressAssociation from DB")
		}
	}

	return row, nil
}

func (d *PostgresDatabase) AddProcessedTransaction(chain Chain, transactionID, receivingAddress string) (bool, error) {
	processedTransactionTable := d.getTable(processedTransactionTableName, nil)
	processedTransaction := processedTransactionRow{chain, transactionID, receivingAddress, time.Now()}
	_, err := processedTransactionTable.Insert(processedTransaction).Exec()
	if err != nil && isDuplicateError(err) {
		return true, nil
	}
	return false, err
}

func (d *PostgresDatabase) IncrementAddressIndex(chain Chain) (uint32, error) {
	var key string
	switch chain {
	case ChainBitcoin:
		key = bitcoinAddressIndexKey
	case ChainEthereum:
		key = ethereumAddressIndexKey
	default:
		return 0, errors.New("Invalid chain")
	}

	row := keyValueStoreRow{}

	session := d.session.Clone()
	keyValueStore := d.getTable(keyValueStoreTableName, session)

	err := session.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "Error starting a new transaction")
	}
	defer session.Rollback()

	err = keyValueStore.Get(&row, map[string]interface{}{"key": key}).Suffix("FOR UPDATE").Exec()
	if err != nil {
		return 0, errors.Wrap(err, "Error getting `"+key+"` from DB")
	}

	// TODO check for overflows - should we create a new account 1'?
	index, err := strconv.ParseUint(row.Value, 10, 32)
	if err != nil {
		return 0, errors.Wrap(err, "Error converting `"+key+"` value to uint32")
	}

	index++

	// TODO: something's wrong with db.Table.Update(). Setting the first argument does not work as expected.
	_, err = keyValueStore.Update(nil, map[string]interface{}{"key": key}).Set("value", index).Exec()
	if err != nil {
		return 0, errors.Wrap(err, "Error updating `"+key+"`")
	}

	err = session.Commit()
	if err != nil {
		return 0, errors.Wrap(err, "Error commiting a transaction")
	}

	return uint32(index), nil
}

func (d *PostgresDatabase) ResetBlockCounters() error {
	keyValueStore := d.getTable(keyValueStoreTableName, nil)

	_, err := keyValueStore.Update(nil, map[string]interface{}{"key": bitcoinLastBlockKey}).Set("value", 0).Exec()
	if err != nil {
		return errors.Wrap(err, "Error reseting `bitcoinLastBlockKey`")
	}

	_, err = keyValueStore.Update(nil, map[string]interface{}{"key": ethereumLastBlockKey}).Set("value", 0).Exec()
	if err != nil {
		return errors.Wrap(err, "Error reseting `ethereumLastBlockKey`")
	}

	return nil
}

func (d *PostgresDatabase) GetEthereumBlockToProcess() (uint64, error) {
	return d.getBlockToProcess(ethereumLastBlockKey)
}

func (d *PostgresDatabase) SaveLastProcessedEthereumBlock(block uint64) error {
	return d.saveLastProcessedBlock(ethereumLastBlockKey, block)
}

func (d *PostgresDatabase) GetBitcoinBlockToProcess() (uint64, error) {
	return d.getBlockToProcess(bitcoinLastBlockKey)
}

func (d *PostgresDatabase) SaveLastProcessedBitcoinBlock(block uint64) error {
	return d.saveLastProcessedBlock(bitcoinLastBlockKey, block)
}

func (d *PostgresDatabase) getBlockToProcess(key string) (uint64, error) {
	keyValueStore := d.getTable(keyValueStoreTableName, nil)
	row := keyValueStoreRow{}

	err := keyValueStore.Get(&row, map[string]interface{}{"key": key}).Exec()
	if err != nil {
		return 0, errors.Wrap(err, "Error getting `"+key+"` from DB")
	}

	block, err := strconv.ParseUint(row.Value, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "Error converting `"+key+"` value to uint64")
	}

	// If set, `block` is the last processed block so we need to start processing from the next one.
	if block > 0 {
		block++
	}
	return block, nil
}

func (d *PostgresDatabase) saveLastProcessedBlock(key string, block uint64) error {
	row := keyValueStoreRow{}

	session := d.session.Clone()
	keyValueStore := d.getTable(keyValueStoreTableName, session)

	err := session.Begin()
	if err != nil {
		return errors.Wrap(err, "Error starting a new transaction")
	}
	defer session.Rollback()

	err = keyValueStore.Get(&row, map[string]interface{}{"key": key}).Suffix("FOR UPDATE").Exec()
	if err != nil {
		return errors.Wrap(err, "Error getting `"+key+"` from DB")
	}

	lastBlock, err := strconv.ParseUint(row.Value, 10, 64)
	if err != nil {
		return errors.Wrap(err, "Error converting `"+key+"` value to uint32")
	}

	if block > lastBlock {
		// TODO: something's wrong with db.Table.Update(). Setting the first argument does not work as expected.
		_, err = keyValueStore.Update(nil, map[string]interface{}{"key": key}).Set("value", block).Exec()
		if err != nil {
			return errors.Wrap(err, "Error updating `"+key+"`")
		}
	}

	err = session.Commit()
	if err != nil {
		return errors.Wrap(err, "Error commiting a transaction")
	}

	return nil
}

// AddEvent adds a new server-sent event to the storage.
func (d *PostgresDatabase) AddEvent(event sse.Event) error {
	broadcastedEventTable := d.getTable(broadcastedEventTableName, nil)
	_, err := broadcastedEventTable.Insert(event).Exec()
	if err != nil {
		if isDuplicateError(err) {
			return nil
		}
	}
	return err
}

// GetEventsSinceID returns all events since `id`. Used to load and publish
// all broadcasted events.
// It returns the last event ID, list of events or error.
// If `id` is equal `-1`:
//    * it should return the last event ID and empty list if at least one
//      event has been broadcasted.
//    * it should return `0` if no events have been broadcasted.
func (d *PostgresDatabase) GetEventsSinceID(id int64) (int64, []sse.Event, error) {
	if id == -1 {
		lastID, err := d.getEventsLastID()
		return lastID, nil, err
	}

	broadcastedEventTable := d.getTable(broadcastedEventTableName, nil)
	rows := []broadcastedEventRow{}
	err := broadcastedEventTable.Select(&rows, "id > ?", id).Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return 0, nil, nil
		default:
			return 0, nil, errors.Wrap(err, "Error getting broadcastedEvent's from DB")
		}
	}

	var lastID int64
	returnRows := make([]sse.Event, len(rows))
	if len(rows) > 0 {
		lastID = rows[len(rows)-1].ID
		for i, event := range rows {
			returnRows[i] = event.toSSE()
		}
	}

	return lastID, returnRows, nil
}

// Returns the last event ID from broadcasted_event table.
func (d *PostgresDatabase) getEventsLastID() (int64, error) {
	row := struct {
		ID int64 `db:"id"`
	}{}
	broadcastedEventTable := d.getTable(broadcastedEventTableName, nil)
	// TODO: `1=1`. We should be able to get row without WHERE clause.
	// When it's set to nil: `pq: syntax error at or near "ORDER""`
	err := broadcastedEventTable.Get(&row, "1=1").OrderBy("id DESC").Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return 0, nil
		default:
			return 0, errors.Wrap(err, "Error getting events last ID")
		}
	}

	return row.ID, nil
}

func (d *PostgresDatabase) AddRecoveryTransaction(sourceAccount string, txEnvelope string) error {
	recoveryTransactionTable := d.getTable(recoveryTransactionTableName, nil)
	recoveryTransaction := recoveryTransactionRow{Source: sourceAccount, EnvelopeXDR: txEnvelope}

	_, err := recoveryTransactionTable.Insert(recoveryTransaction).Exec()
	return err
}
