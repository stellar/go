package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

type BatchConfig struct {
	historyarchive.Range
	TxMetaSourceUrl   string
	IndexTargetUrl    string
	NetworkPassphrase string
}

const (
	batchSizeEnv         = "BATCH_SIZE"
	jobIndexEnv          = "AWS_BATCH_JOB_ARRAY_INDEX"
	firstCheckpointEnv   = "FIRST_CHECKPOINT"
	txmetaSourceUrlEnv   = "TXMETA_SOURCE"
	indexTargetUrlEnv    = "INDEX_TARGET"
	workerCountEnv       = "WORKER_COUNT"
	networkPassphraseEnv = "NETWORK_PASSPHRASE"
)

func NewBatchConfig() (*BatchConfig, error) {
	indexTargetRootUrl := os.Getenv(indexTargetUrlEnv)
	if indexTargetRootUrl == "" {
		return nil, errors.New("required parameter: " + indexTargetUrlEnv)
	}

	jobIndex, err := strconv.ParseUint(os.Getenv(jobIndexEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+jobIndexEnv)
	}

	firstCheckpoint, err := strconv.ParseUint(os.Getenv(firstCheckpointEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+firstCheckpointEnv)
	}

	checkpoints := historyarchive.NewCheckpointManager(0)
	if !checkpoints.IsCheckpoint(uint32(firstCheckpoint - 1)) {
		return nil, fmt.Errorf(
			"%s (%d) must be the first ledger in a checkpoint range",
			firstCheckpointEnv, firstCheckpoint)
	}

	batchSize, err := strconv.ParseUint(os.Getenv(batchSizeEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+batchSizeEnv)
	} else if batchSize%uint64(checkpoints.GetCheckpointFrequency()) != 0 {
		return nil, fmt.Errorf(
			"%s (%d) must be a multiple of checkpoint frequency (%d)",
			batchSizeEnv, batchSize, checkpoints.GetCheckpointFrequency())
	}

	txmetaSourceUrl := os.Getenv(txmetaSourceUrlEnv)
	if txmetaSourceUrl == "" {
		return nil, errors.New("required parameter " + txmetaSourceUrlEnv)
	}

	networkPassphrase := os.Getenv(networkPassphraseEnv)
	switch networkPassphrase {
	case "":
		log.Warnf("%s not specified, defaulting to 'testnet'", networkPassphraseEnv)
		fallthrough
	case "testnet":
		networkPassphrase = network.TestNetworkPassphrase
	case "pubnet":
		networkPassphrase = network.PublicNetworkPassphrase
	default:
		log.Warnf("%s is not a recognized shortcut ('pubnet' or 'testnet')",
			networkPassphraseEnv)
	}
	log.Infof("Using network passphrase '%s'", networkPassphrase)

	firstLedger := uint32(firstCheckpoint + batchSize*jobIndex)
	lastLedger := firstLedger + uint32(batchSize) - 1
	return &BatchConfig{
		Range:             historyarchive.Range{Low: firstLedger, High: lastLedger},
		TxMetaSourceUrl:   txmetaSourceUrl,
		IndexTargetUrl:    fmt.Sprintf("%s%cjob_%d", indexTargetRootUrl, os.PathSeparator, jobIndex),
		NetworkPassphrase: networkPassphrase,
	}, nil
}

func main() {
	log.SetLevel(log.InfoLevel)
	// log.SetLevel(log.DebugLevel)

	batch, err := NewBatchConfig()
	if err != nil {
		panic(err)
	}

	var workerCount int
	workerCountStr := os.Getenv(workerCountEnv)
	if workerCountStr == "" {
		workerCount = runtime.NumCPU()
	} else {
		workerCountParsed, innerErr := strconv.ParseUint(workerCountStr, 10, 8)
		if innerErr != nil {
			panic(errors.Wrapf(innerErr,
				"invalid worker count parameter (%s)", workerCountStr))
		}
		workerCount = int(workerCountParsed)
	}

	log.Infof("Uploading ledger range [%d, %d] to %s",
		batch.Range.Low, batch.Range.High, batch.IndexTargetUrl)

	if _, err := index.BuildIndices(
		context.Background(),
		batch.TxMetaSourceUrl,
		batch.IndexTargetUrl,
		batch.NetworkPassphrase,
		batch.Range,
		[]string{
			"accounts_unbacked",
			"transactions",
		},
		workerCount,
	); err != nil {
		panic(err)
	}
}
