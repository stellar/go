package ethereum

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stellar/go/support/log"
	"github.com/tyler-smith/go-bip32"
)

// Listener listens for transactions using geth RPC. It calls TransactionHandler for each new
// transactions. It will reproces the block if TransactionHandler returns error. It will
// start from the block number returned from Storage.GetEthereumBlockToProcess or the latest block
// if it returned 0. Transactions can be processed more than once, it's TransactionHandler
// responsibility to ignore duplicates.
// You can run multiple Listeners if Storage is implemented correctly.
// Listener requires geth 1.7.0.
type Listener struct {
	Storage            Storage `inject:""`
	TransactionHandler TransactionHandler

	client *ethclient.Client
	log    *log.Entry
}

// Storage is an interface that must be implemented by an object using
// persistent storage.
type Storage interface {
	// GetEthereumBlockToProcess gets the number of Ethereum block to process. `0`means the
	// processing should start from the current block.
	GetEthereumBlockToProcess() (uint64, error)
	// SaveLastProcessedEthereumBlock should update the number of the last processed Ethereum
	// block. It should only update the block if block > current block in atomic transaction.
	SaveLastProcessedEthereumBlock(block uint64) error
}

type TransactionHandler func(transaction *types.Transaction) error

type AddressGenerator struct {
	masterPublicKey *bip32.Key
}
