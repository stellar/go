package database

import (
	"time"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

type Chain string

// Scan implements database/sql.Scanner interface
func (s *Chain) Scan(src interface{}) error {
	value, ok := src.([]byte)
	if !ok {
		return errors.New("Cannot convert value to Chain")
	}
	*s = Chain(value)
	return nil
}

const (
	ChainEthereum Chain = "ethereum"
)

type Database interface {
	// CreateEthereumAddressAssociation creates Ethereum-Stellar association. `addressIndex`
	// is the ethereum address derivation index (BIP-32).
	CreateEthereumAddressAssociation(stellarAddress, ethereumAddress string, addressIndex uint32) error
	// GetAssociationByEthereumAddress searches for previously saved Ethereum-Stellar association.
	// Should return nil if not found.
	GetAssociationByEthereumAddress(ethereumAddress string) (*AddressAssociation, error)
	// GetAssociationByStellarPublicKey searches for previously saved Ethereum-Stellar association.
	// Should return nil if not found.
	GetAssociationByStellarPublicKey(stellarPublicKey string) (*AddressAssociation, error)

	// AddProcessedTransaction adds a transaction to database as processed. This
	// should return `nil` if transaction is already added.
	AddProcessedTransaction(chain Chain, transactionID string) error
	// IsTransactionProcessed returns `true` if transaction has been already processed.
	IsTransactionProcessed(chain Chain, transactionID string) (bool, error)

	// IncrementEthereumAddressIndex returns the current value of index used for ethereum key
	// derivation and then increments it. This operation must be atomic so this function
	// should never return the same value more than once.
	IncrementEthereumAddressIndex() (uint32, error)
}

type PostgresDatabase struct {
	session *db.Session
}

type AddressAssociation struct {
	// Chain is the name of the payment origin chain: currently `ethereum` only
	Chain Chain `db:"chain"`
	// BIP-44
	AddressIndex     uint32    `db:"address_index"`
	Address          string    `db:"address"`
	StellarPublicKey string    `db:"stellar_public_key"`
	CreatedAt        time.Time `db:"created_at"`
}
