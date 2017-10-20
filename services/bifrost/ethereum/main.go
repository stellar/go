package ethereum

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stellar/go/support/log"
	"github.com/tyler-smith/go-bip32"
)

const stellarAmountPrecision = 7

var (
	ten      = big.NewInt(10)
	eighteen = big.NewInt(18)
	// weiInEth = 10^18
	weiInEth = new(big.Rat).SetInt(new(big.Int).Exp(ten, eighteen, nil))
)

// Listener listens for transactions using geth RPC. It calls TransactionHandler for each new
// transactions. It will reprocess the block if TransactionHandler returns error. It will
// start from the block number returned from Storage.GetEthereumBlockToProcess or the latest block
// if it returned 0. Transactions can be processed more than once, it's TransactionHandler
// responsibility to ignore duplicates.
// You can run multiple Listeners if Storage is implemented correctly.
// Listener ignores contract creation transactions.
// Listener requires geth 1.7.0.
type Listener struct {
	Client             Client  `inject:""`
	Storage            Storage `inject:""`
	NetworkID          string
	TransactionHandler TransactionHandler

	log *log.Entry
}

type Client interface {
	NetworkID(ctx context.Context) (*big.Int, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

// Storage is an interface that must be implemented by an object using
// persistent storage.
type Storage interface {
	// GetEthereumBlockToProcess gets the number of Ethereum block to process. `0` means the
	// processing should start from the current block.
	GetEthereumBlockToProcess() (uint64, error)
	// SaveLastProcessedEthereumBlock should update the number of the last processed Ethereum
	// block. It should only update the block if block > current block in atomic transaction.
	SaveLastProcessedEthereumBlock(block uint64) error
}

type TransactionHandler func(transaction Transaction) error

type Transaction struct {
	Hash string
	// Value in Wei
	ValueWei *big.Int
	To       string
}

type AddressGenerator struct {
	masterPublicKey *bip32.Key
}
