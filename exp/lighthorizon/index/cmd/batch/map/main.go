package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

type BatchConfig struct {
	historyarchive.Range
	TxMetaSourceUrl, IndexTargetUrl string
}

const (
	batchSizeEnv       = "BATCH_SIZE"
	jobIndexEnv        = "AWS_BATCH_JOB_ARRAY_INDEX"
	firstCheckpointEnv = "FIRST_CHECKPOINT"
	txmetaSourceUrlEnv = "TXMETA_SOURCE"
	indexTargetUrlEnv  = "INDEX_TARGET"

	s3BucketName = "sdf-txmeta-pubnet"
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
	if (firstCheckpoint+1)%64 != 0 {
		return nil, fmt.Errorf("invalid checkpoint: %d", firstCheckpoint)
	}

	batchSize, err := strconv.ParseUint(os.Getenv(batchSizeEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+batchSizeEnv)
	}

	txmetaSourceUrl := os.Getenv(txmetaSourceUrlEnv)
	if txmetaSourceUrl == "" {
		return nil, errors.New("required parameter " + txmetaSourceUrlEnv)
	}

	startCheckpoint := uint32(firstCheckpoint + batchSize*jobIndex)
	endLedger := startCheckpoint + uint32(batchSize) - 1
	return &BatchConfig{
		Range:           historyarchive.Range{Low: startCheckpoint, High: endLedger},
		TxMetaSourceUrl: txmetaSourceUrl,
		IndexTargetUrl:  fmt.Sprintf("%s%cjob_%d", indexTargetRootUrl, os.PathSeparator, jobIndex),
	}, nil
}

func main() {
	log.SetLevel(log.InfoLevel)
	// log.SetLevel(log.DebugLevel)

	batch, err := NewBatchConfig()
	if err != nil {
		panic(err)
	}

	log.Infof("Uploading ledger range [%d, %d] to %s",
		batch.Range.Low, batch.Range.High, batch.IndexTargetUrl)

	if _, err := index.BuildIndices(
		context.Background(),
		batch.TxMetaSourceUrl,
		batch.IndexTargetUrl,
		network.TestNetworkPassphrase,
		batch.Low, batch.High,
		[]string{"transactions", "accounts_unbacked"},
		1,
	); err != nil {
		panic(err)
	}
}
