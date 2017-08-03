package ethereum

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

func (l *Listener) Start(rpcServer string) error {
	l.log = common.CreateLogger("EthereumListener")
	l.log.Info("EthereumListener starting")

	rpcClient, err := rpc.Dial(rpcServer)
	if err != nil {
		err = errors.Wrap(err, "Error dialing geth")
		l.log.Error(err)
		return err
	}

	l.client = ethclient.NewClient(rpcClient)
	blockNumber, err := l.Storage.GetEthereumBlockToProcess()
	if err != nil {
		err = errors.Wrap(err, "Error getting ethereum block to process from DB")
		l.log.Error(err)
		return err
	}

	go l.processBlocks(blockNumber)
	return nil
}

func (l *Listener) processBlocks(blockNumber uint64) {
	if blockNumber == 0 {
		l.log.Info("Starting from the latest block")
	} else {
		l.log.Infof("Starting from block %d", blockNumber)
	}

	for {
		block, err := l.getBlock(blockNumber)
		if err != nil {
			l.log.WithFields(log.F{"err": err, "blockNumber": blockNumber}).Error("Error getting block")
			time.Sleep(1 * time.Second)
			continue
		}

		// Block doesn't exist yet
		if block == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		err = l.processBlock(block)
		if err != nil {
			l.log.WithFields(log.F{"err": err, "blockNumber": block.NumberU64()}).Error("Error processing block")
			time.Sleep(1 * time.Second)
			continue
		}

		// Persist block number
		err = l.Storage.SaveLastProcessedEthereumBlock(blockNumber)
		if err != nil {
			l.log.WithField("err", err).Error("Error saving last processed block")
			time.Sleep(1 * time.Second)
			// We continue to the next block
		}

		blockNumber = block.NumberU64() + 1
	}
}

// getBlock returns (nil, nil) if block has not been found (not exists yet)
func (l *Listener) getBlock(blockNumber uint64) (*types.Block, error) {
	var blockNumberInt *big.Int
	if blockNumber > 0 {
		blockNumberInt = big.NewInt(int64(blockNumber))
	}

	d := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	block, err := l.client.BlockByNumber(ctx, blockNumberInt)
	if err != nil {
		if err.Error() == "not found" {
			return nil, nil
		}
		err = errors.Wrap(err, "Error getting block from geth")
		l.log.WithField("block", blockNumberInt.String()).Error(err)
		return nil, err
	}

	return block, nil
}

func (l *Listener) processBlock(block *types.Block) error {
	transactions := block.Transactions()
	blockTime := time.Unix(block.Time().Int64(), 0)

	localLog := l.log.WithFields(log.F{
		"blockNumber":  block.NumberU64(),
		"blockTime":    blockTime,
		"transactions": len(transactions),
	})
	localLog.Info("Processing block")

	for _, transaction := range transactions {
		err := l.TransactionHandler(transaction)
		if err != nil {
			return errors.Wrap(err, "Error processing transaction")
		}
	}

	localLog.Info("Processed block")

	return nil
}
