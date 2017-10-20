package bitcoin

import (
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

func (l *Listener) Start() error {
	l.log = common.CreateLogger("BitcoinListener")
	l.log.Info("BitcoinListener starting")

	genesisBlockHash, err := l.Client.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "Error getting genesis block")
	}

	if l.Testnet {
		l.chainParams = &chaincfg.TestNet3Params
		if !genesisBlockHash.IsEqual(chaincfg.TestNet3Params.GenesisHash) {
			return errors.New("Invalid genesis hash")
		}
	} else {
		l.chainParams = &chaincfg.MainNetParams
		if !genesisBlockHash.IsEqual(chaincfg.MainNetParams.GenesisHash) {
			return errors.New("Invalid genesis hash")
		}
	}

	blockNumber, err := l.Storage.GetBitcoinBlockToProcess()
	if err != nil {
		err = errors.Wrap(err, "Error getting bitcoin block to process from DB")
		l.log.Error(err)
		return err
	}

	if blockNumber == 0 {
		blockNumberTmp, err := l.Client.GetBlockCount()
		if err != nil {
			err = errors.Wrap(err, "Error getting the block count from bitcoin-core")
			l.log.Error(err)
			return err
		}
		blockNumber = uint64(blockNumberTmp)
	}

	go l.processBlocks(blockNumber)
	return nil
}

func (l *Listener) processBlocks(blockNumber uint64) {
	l.log.Infof("Starting from block %d", blockNumber)

	// Time when last new block has been seen
	lastBlockSeen := time.Now()
	noBlockWarningLogged := false

	for {
		block, err := l.getBlock(blockNumber)
		if err != nil {
			l.log.WithFields(log.F{"err": err, "blockNumber": blockNumber}).Error("Error getting block")
			time.Sleep(1 * time.Second)
			continue
		}

		// Block doesn't exist yet
		if block == nil {
			if time.Since(lastBlockSeen) > 20*time.Minute && !noBlockWarningLogged {
				l.log.Warn("No new block in more than 20 minutes")
				noBlockWarningLogged = true
			}

			time.Sleep(1 * time.Second)
			continue
		}

		// Reset counter when new block appears
		lastBlockSeen = time.Now()
		noBlockWarningLogged = false

		err = l.processBlock(block)
		if err != nil {
			l.log.WithFields(log.F{"err": err, "blockHash": block.Header.BlockHash().String()}).Error("Error processing block")
			time.Sleep(1 * time.Second)
			continue
		}

		// Persist block number
		err = l.Storage.SaveLastProcessedBitcoinBlock(blockNumber)
		if err != nil {
			l.log.WithField("err", err).Error("Error saving last processed block")
			time.Sleep(1 * time.Second)
			// We continue to the next block
		}

		blockNumber++
	}
}

// getBlock returns (nil, nil) if block has not been found (not exists yet)
func (l *Listener) getBlock(blockNumber uint64) (*wire.MsgBlock, error) {
	blockHeight := int64(blockNumber)
	blockHash, err := l.Client.GetBlockHash(blockHeight)
	if err != nil {
		if strings.Contains(err.Error(), "Block height out of range") {
			// Block does not exist yet
			return nil, nil
		}
		err = errors.Wrap(err, "Error getting block hash from bitcoin-core")
		l.log.WithField("blockHeight", blockHeight).Error(err)
		return nil, err
	}

	block, err := l.Client.GetBlock(blockHash)
	if err != nil {
		err = errors.Wrap(err, "Error getting block from bitcoin-core")
		l.log.WithField("blockHash", blockHash.String()).Error(err)
		return nil, err
	}

	return block, nil
}

func (l *Listener) processBlock(block *wire.MsgBlock) error {
	transactions := block.Transactions

	localLog := l.log.WithFields(log.F{
		"blockHash":    block.Header.BlockHash().String(),
		"blockTime":    block.Header.Timestamp,
		"transactions": len(transactions),
	})
	localLog.Info("Processing block")

	for _, transaction := range transactions {
		transactionLog := localLog.WithField("transactionHash", transaction.TxHash().String())

		for index, output := range transaction.TxOut {
			class, addresses, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, l.chainParams)
			if err != nil {
				// txscript.ExtractPkScriptAddrs returns error on non-standard scripts
				// so this can be Warn.
				transactionLog.WithField("err", err).Warn("Error extracting addresses")
				continue
			}

			// We only support P2PK and P2PKH addresses
			if class != txscript.PubKeyTy && class != txscript.PubKeyHashTy {
				transactionLog.WithField("class", class).Debug("Invalid addresses class")
				continue
			}

			// Paranoid. We access address[0] later.
			if len(addresses) != 1 {
				transactionLog.WithField("addresses", addresses).Error("Invalid addresses length")
				continue
			}

			handlerTransaction := Transaction{
				Hash:       transaction.TxHash().String(),
				TxOutIndex: index,
				ValueSat:   output.Value,
				To:         addresses[0].EncodeAddress(),
			}

			err = l.TransactionHandler(handlerTransaction)
			if err != nil {
				return errors.Wrap(err, "Error processing transaction")
			}
		}
	}

	localLog.Info("Processed block")

	return nil
}
