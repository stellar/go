package database

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

const (
	ethereumAddressIndexKey = "ethereum_address_index"
	ethereumLastBlockKey    = "ethereum_last_block"

	addressAssociationTableName   = "address_association"
	keyValueStoreTableName        = "key_value_store"
	processedTransactionTableName = "processed_transaction"
	transactionsQueueTableName    = "transactions_queue"
)

type keyValueStoreRow struct {
	Key   string `db:"key"`
	Value string `db:"value"`
}

type transactionsQueueRow struct {
	TransactionID    string          `db:"transaction_id"`
	AssetCode        queue.AssetCode `db:"asset_code"`
	Amount           string          `db:"amount"`
	StellarPublicKey string          `db:"stellar_public_key"`
}

type processedTransactionRow struct {
	Chain         Chain  `db:"chain"`
	TransactionID string `db:"transaction_id"`
}

func fromQueueTransaction(tx queue.Transaction) *transactionsQueueRow {
	return &transactionsQueueRow{
		TransactionID:    tx.TransactionID,
		AssetCode:        tx.AssetCode,
		Amount:           tx.Amount,
		StellarPublicKey: tx.StellarPublicKey,
	}
}

func (r *transactionsQueueRow) toQueueTransaction() *queue.Transaction {
	return &queue.Transaction{
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

func (d *PostgresDatabase) CreateEthereumAddressAssociation(stellarAddress, ethereumAddress string, addressIndex uint32) error {
	addressAssociationTable := d.getTable(addressAssociationTableName, nil)

	association := &AddressAssociation{
		Chain:            ChainEthereum,
		AddressIndex:     addressIndex,
		Address:          ethereumAddress,
		StellarPublicKey: stellarAddress,
		CreatedAt:        time.Now(),
	}

	_, err := addressAssociationTable.Insert(association).Exec()
	return err
}

func (d *PostgresDatabase) GetAssociationByEthereumAddress(ethereumAddress string) (*AddressAssociation, error) {
	addressAssociationTable := d.getTable(addressAssociationTableName, nil)
	row := &AddressAssociation{}
	where := map[string]interface{}{"address": ethereumAddress, "chain": ChainEthereum}
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

func (d *PostgresDatabase) AddProcessedTransaction(chain Chain, transactionID string) error {
	processedTransactionTable := d.getTable(processedTransactionTableName, nil)
	processedTransaction := processedTransactionRow{chain, transactionID}
	_, err := processedTransactionTable.Insert(processedTransaction).Exec()
	return err
}

func (d *PostgresDatabase) IsTransactionProcessed(chain Chain, transactionID string) (bool, error) {
	processedTransactionTable := d.getTable(processedTransactionTableName, nil)

	row := processedTransactionRow{}
	where := map[string]interface{}{"chain": chain, "transaction_id": transactionID}
	err := processedTransactionTable.Get(&row, where).Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return false, nil
		default:
			return false, errors.Wrap(err, "Error getting processedTransaction from DB")
		}
	}

	return true, nil
}

func (d *PostgresDatabase) IncrementEthereumAddressIndex() (uint32, error) {
	row := keyValueStoreRow{}

	session := d.session.Clone()
	keyValueStore := d.getTable(keyValueStoreTableName, session)

	err := session.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "Error starting a new transaction")
	}
	defer session.Rollback()

	err = keyValueStore.Get(&row, map[string]interface{}{"key": ethereumAddressIndexKey}).Suffix("FOR UPDATE").Exec()
	if err != nil {
		return 0, errors.Wrap(err, "Error getting `ethereumAddressIndexKey` from DB")
	}

	// TODO check for overflows - should we create a new account 1'?
	index, err := strconv.ParseUint(row.Value, 10, 32)
	if err != nil {
		return 0, errors.Wrap(err, "Error converting `ethereumAddressIndexKey` value to uint32")
	}

	index++

	// TODO: something's wrong with db.Table.Update(). Setting the first argument does not work as expected.
	_, err = keyValueStore.Update(nil, map[string]interface{}{"key": ethereumAddressIndexKey}).Set("value", index).Exec()
	if err != nil {
		return 0, errors.Wrap(err, "Error updating `ethereumAddressIndexKey`")
	}

	err = session.Commit()
	if err != nil {
		return 0, errors.Wrap(err, "Error commiting a transaction")
	}

	return uint32(index), nil
}

func (d *PostgresDatabase) GetEthereumBlockToProcess() (uint64, error) {
	keyValueStore := d.getTable(keyValueStoreTableName, nil)
	row := keyValueStoreRow{}

	err := keyValueStore.Get(&row, map[string]interface{}{"key": ethereumLastBlockKey}).Exec()
	if err != nil {
		return 0, errors.Wrap(err, "Error getting `ethereumLastBlockKey` from DB")
	}

	block, err := strconv.ParseUint(row.Value, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "Error converting `ethereumLastBlockKey` value to uint64")
	}

	// If set, `block` is the last processed block so we need to start processing from the next one.
	if block > 0 {
		block++
	}
	return block, nil
}

func (d *PostgresDatabase) SaveLastProcessedEthereumBlock(block uint64) error {
	row := keyValueStoreRow{}

	session := d.session.Clone()
	keyValueStore := d.getTable(keyValueStoreTableName, session)

	err := session.Begin()
	if err != nil {
		return errors.Wrap(err, "Error starting a new transaction")
	}
	defer session.Rollback()

	err = keyValueStore.Get(&row, map[string]interface{}{"key": ethereumLastBlockKey}).Suffix("FOR UPDATE").Exec()
	if err != nil {
		return errors.Wrap(err, "Error getting `ethereumLastBlockKey` from DB")
	}

	lastBlock, err := strconv.ParseUint(row.Value, 10, 64)
	if err != nil {
		return errors.Wrap(err, "Error converting `ethereumLastBlockKey` value to uint32")
	}

	if block > lastBlock {
		// TODO: something's wrong with db.Table.Update(). Setting the first argument does not work as expected.
		_, err = keyValueStore.Update(nil, map[string]interface{}{"key": ethereumLastBlockKey}).Set("value", block).Exec()
		if err != nil {
			return errors.Wrap(err, "Error updating `ethereumLastBlockKey`")
		}
	}

	err = session.Commit()
	if err != nil {
		return errors.Wrap(err, "Error commiting a transaction")
	}

	return nil
}

// Add implements queue.Queue interface
func (d *PostgresDatabase) Add(tx queue.Transaction) error {
	transactionsQueueTable := d.getTable(transactionsQueueTableName, nil)
	transactionQueue := fromQueueTransaction(tx)
	_, err := transactionsQueueTable.Insert(transactionQueue).Exec()
	return err
}

// Pool receives and removes the head of this queue. Returns nil if no elements found.
// Pool implements queue.Queue interface.
func (d *PostgresDatabase) Pool() (*queue.Transaction, error) {
	row := transactionsQueueRow{}

	session := d.session.Clone()
	transactionsQueueTable := d.getTable(transactionsQueueTableName, session)

	err := session.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "Error starting a new transaction")
	}
	defer session.Rollback()

	err = transactionsQueueTable.Get(&row, map[string]interface{}{"pooled": false}).Suffix("FOR UPDATE").Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "Error getting transaction from a queue")
		}
	}

	// TODO: something's wrong with db.Table.Update(). Setting the first argument does not work as expected.
	where := map[string]interface{}{"transaction_id": row.TransactionID, "asset_code": row.AssetCode}
	_, err = transactionsQueueTable.Update(nil, where).Set("pooled", true).Exec()
	if err != nil {
		return nil, errors.Wrap(err, "Error setting transaction as pooled in a queue")
	}

	err = session.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "Error commiting a transaction")
	}

	return row.toQueueTransaction(), nil
}
