package bitcoin

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/stellar/go/support/log"
	"github.com/tyler-smith/go-bip32"
)

// Listener listens for transactions using bitcoin-core RPC. It calls TransactionHandler for each new
// transactions. It will reprocess the block if TransactionHandler returns error. It will
// start from the block number returned from Storage.GetBitcoinBlockToProcess or the latest block
// if it returned 0. Transactions can be processed more than once, it's TransactionHandler
// responsibility to ignore duplicates.
// Listener tracks only P2PKH payments.
// You can run multiple Listeners if Storage is implemented correctly.
type Listener struct {
	Storage            Storage `inject:""`
	TransactionHandler TransactionHandler
	Testnet            bool

	client      *rpcclient.Client
	chainParams *chaincfg.Params
	log         *log.Entry
}

// Storage is an interface that must be implemented by an object using
// persistent storage.
type Storage interface {
	// GetBitcoinBlockToProcess gets the number of Bitcoin block to process. `0` means the
	// processing should start from the current block.
	GetBitcoinBlockToProcess() (uint64, error)
	// SaveLastProcessedBitcoinBlock should update the number of the last processed Bitcoin
	// block. It should only update the block if block > current block in atomic transaction.
	SaveLastProcessedBitcoinBlock(block uint64) error
}

type TransactionHandler func(transaction Transaction) error

type Transaction struct {
	Hash       string
	TxOutIndex int
	Value      int64
	To         string
}

type AddressGenerator struct {
	masterPublicKey *bip32.Key
}
