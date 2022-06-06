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
	TxMetaSourceUrl, TargetUrl string
}

const (
	batchSizeEnv       = "BATCH_SIZE"
	jobIndexEnv        = "AWS_BATCH_JOB_ARRAY_INDEX"
	firstCheckpointEnv = "FIRST_CHECKPOINT"
	txmetaSourceUrlEnv = "TXMETA_SOURCE"
	indexTargetUrlEnv  = "INDEX_TARGET"

	s3BucketName = "sdf-txmeta-pubnet"
)

func NewS3BatchConfig() (*BatchConfig, error) {
	jobIndex, err := strconv.ParseUint(os.Getenv(jobIndexEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+jobIndexEnv)
	}

	url := fmt.Sprintf("s3://%s/job_%d?region=%s", s3BucketName, jobIndex, "us-east-1")
	if err := os.Setenv(indexTargetUrlEnv, url); err != nil {
		return nil, err
	}

	return NewBatchConfig()
}

func NewBatchConfig() (*BatchConfig, error) {
	targetUrl := os.Getenv(indexTargetUrlEnv)
	if targetUrl == "" {
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

	sourceUrl := os.Getenv(txmetaSourceUrlEnv)
	if sourceUrl == "" {
		return nil, errors.New("required parameter " + txmetaSourceUrlEnv)
	}

	log.Debugf("%s: %d", batchSizeEnv, batchSize)
	log.Debugf("%s: %d", jobIndexEnv, jobIndex)
	log.Debugf("%s: %d", firstCheckpointEnv, firstCheckpoint)
	log.Debugf("%s: %v", txmetaSourceUrlEnv, sourceUrl)

	startCheckpoint := uint32(firstCheckpoint + batchSize*jobIndex)
	endCheckpoint := startCheckpoint + uint32(batchSize) - 1
	return &BatchConfig{
		Range:           historyarchive.Range{Low: startCheckpoint, High: endCheckpoint},
		TxMetaSourceUrl: sourceUrl,
		TargetUrl:       targetUrl,
	}, nil
}

func main() {
	// log.SetLevel(log.DebugLevel)
	log.SetLevel(log.InfoLevel)

	batch, err := NewBatchConfig()
	if err != nil {
		panic(err)
	}

	log.Infof("Uploading ledger range [%d, %d] to %s",
		batch.Range.Low, batch.Range.High, batch.TargetUrl)

	if err := index.BuildIndices(
		context.Background(),
		batch.TxMetaSourceUrl,
		batch.TargetUrl,
		network.TestNetworkPassphrase,
		batch.Low, batch.High,
		[]string{"transactions", "accounts_unbacked"},
		1,
	); err != nil {
		panic(err)
	}
}
