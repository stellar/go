package main

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

const (
	ACCOUNT_FLUSH_FREQUENCY = 200
	// arbitrary default, should we use runtime.NumCPU()?
	DEFAULT_WORKER_COUNT = 2
)

type ReduceConfig struct {
	JobIndex        uint32
	MapJobCount     uint32
	ReduceJobCount  uint32
	IndexTarget     string
	IndexRootSource string

	Workers uint32
}

func ReduceConfigFromEnvironment() (*ReduceConfig, error) {
	const (
		mapJobsEnv         = "MAP_JOB_COUNT"
		reduceJobsEnv      = "REDUCE_JOB_COUNT"
		workerCountEnv     = "WORKER_COUNT"
		jobIndexEnv        = "AWS_BATCH_JOB_ARRAY_INDEX"
		indexRootSourceEnv = "INDEX_SOURCE_ROOT"
		indexTargetEnv     = "INDEX_TARGET"
	)

	jobIndex, err := strconv.ParseUint(os.Getenv(jobIndexEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+jobIndexEnv)
	}
	mapJobCount, err := strconv.ParseUint(os.Getenv(mapJobsEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+mapJobsEnv)
	}
	reduceJobCount, err := strconv.ParseUint(os.Getenv(reduceJobsEnv), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+reduceJobsEnv)
	}

	workersStr := os.Getenv(workerCountEnv)
	if workersStr == "" {
		workersStr = strconv.FormatUint(DEFAULT_WORKER_COUNT, 10)
	}
	workers, err := strconv.ParseUint(workersStr, 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "invalid parameter "+workerCountEnv)
	}

	indexTarget := os.Getenv(indexTargetEnv)
	if indexTarget == "" {
		return nil, errors.New("required parameter missing " + indexTargetEnv)
	}

	indexRootSource := os.Getenv(indexRootSourceEnv)
	if indexRootSource == "" {
		return nil, errors.New("required parameter missing " + indexRootSourceEnv)
	}

	return &ReduceConfig{
		JobIndex:        uint32(jobIndex),
		MapJobCount:     uint32(mapJobCount),
		ReduceJobCount:  uint32(reduceJobCount),
		Workers:         uint32(workers),
		IndexTarget:     indexTarget,
		IndexRootSource: indexRootSource,
	}, nil
}

func main() {
	log.SetLevel(log.InfoLevel)

	config, err := ReduceConfigFromEnvironment()
	if err != nil {
		panic(err)
	}

	log.Infof("Connecting to %s", config.IndexTarget)
	finalIndexStore, err := index.Connect(config.IndexTarget)
	if err != nil {
		panic(errors.Wrapf(err, "failed to connect to indices at %s",
			config.IndexTarget))
	}

	if err := mergeAllIndices(finalIndexStore, config); err != nil {
		panic(errors.Wrap(err, "failed to merge indices"))
	}
}

func mergeAllIndices(finalIndexStore index.Store, config *ReduceConfig) error {
	doneAccounts := NewSafeStringSet()
	for i := uint32(0); i < config.MapJobCount; i++ {
		logger := log.WithField("job", i)

		url := filepath.Join(config.IndexRootSource, "job_"+strconv.FormatUint(uint64(i), 10))
		logger.Infof("Connecting to %s", url)

		outerJobStore, err := index.Connect(url)
		if err != nil {
			return errors.Wrapf(err, "failed to connect to indices at %s", url)
		}

		accounts, err := outerJobStore.ReadAccounts()
		// TODO: in final version this should be critical error, now just skip it
		if os.IsNotExist(err) {
			logger.Errorf("accounts file not found (TODO!)")
			continue
		} else if err != nil {
			return errors.Wrapf(err, "failed to read accounts for job %d", i)
		}

		logger.Infof("Processing %d accounts with %d workers",
			len(accounts), config.Workers)

		workQueues := make([]chan string, config.Workers)
		for i := range workQueues {
			workQueues[i] = make(chan string, 1)
		}

		for idx, queue := range workQueues {
			go (func(index uint32, queue chan string) {
				for _, account := range accounts {
					// Account index already merged in the previous outer job?
					if doneAccounts.Contains(account) {
						continue
					}

					// Account doesn't belong in this work queue?
					if !config.shouldProcessAccount(account, index) {
						continue
					}

					queue <- account
				}

				close(queue)
			})(uint32(idx), queue)
		}

		// TODO: errgroup.WithContext(ctx)
		var wg sync.WaitGroup
		wg.Add(int(config.Workers))
		for j := uint32(0); j < config.Workers; j++ {
			go func(routineIndex uint32) {
				defer wg.Done()
				logger := logger.
					WithField("worker", routineIndex).
					WithField("total", len(accounts))
				logger.Info("Started worker")

				var accountsProcessed, accountsSkipped uint64
				for account := range workQueues[routineIndex] {
					logger.Infof("Account: %s", account)
					if (accountsProcessed+accountsSkipped)%97 == 0 {
						logger.
							WithField("indexed", accountsProcessed).
							WithField("skipped", accountsSkipped).
							Infof("Processed %d/%d accounts",
								accountsProcessed+accountsSkipped, len(accounts))
					}

					logger.Infof("Reading index for account: %s", account)

					// First, open the "final merged indices" at the root level
					// for this account.
					mergedIndices, err := outerJobStore.Read(account)

					// TODO: in final version this should be critical error, now just skip it
					if os.IsNotExist(err) {
						logger.Errorf("Account %s is unavailable - TODO fix", account)
						continue
					} else if err != nil {
						panic(err)
					}

					// Then, iterate through all of the job folders and merge
					// indices from all jobs that touched this account.
					for k := uint32(0); k < config.MapJobCount; k++ {
						url := filepath.Join(config.IndexRootSource, fmt.Sprintf("job_%d", k))

						// FIXME: This could probably come from a pool. Every
						// worker needs to have a connection to every index
						// store, so there's no reason to re-open these for each
						// inner loop.
						innerJobStore, err := index.Connect(url)
						if err != nil {
							logger.WithError(err).
								Errorf("Failed to open index at %s", url)
							panic(err)
						}

						jobIndices, err := innerJobStore.Read(account)
						// This job never touched this account; skip.
						if os.IsNotExist(err) {
							continue
						} else if err != nil {
							logger.WithError(err).
								Errorf("Failed to read index for %s", account)
							panic(err)
						}

						if err := mergeIndices(mergedIndices, jobIndices); err != nil {
							logger.WithError(err).
								Errorf("Merge failure for index at %s", url)
							panic(err)
						}
					}

					// Finally, save the merged index.
					finalIndexStore.AddParticipantToIndexesNoBackend(account, mergedIndices)

					// Mark this account for other workers to ignore.
					doneAccounts.Add(account)
					accountsProcessed++
					logger = logger.WithField("processed", accountsProcessed)

					// Periodically flush to disk to save memory.
					if accountsProcessed%ACCOUNT_FLUSH_FREQUENCY == 0 {
						logger.Infof("Flushing indexed accounts.")
						if err = finalIndexStore.Flush(); err != nil {
							logger.WithError(err).Errorf("Flush error.")
							panic(err)
						}
					}
				}

				logger.Infof("Final account flush.")
				if err = finalIndexStore.Flush(); err != nil {
					logger.WithError(err).Errorf("Flush error.")
					panic(err)
				}

				// Merge the transaction indexes
				// There's 256 files, (one for each first byte of the txn hash)
				var transactionsProcessed, transactionsSkipped uint64
				logger = logger.
					WithField("indexed", transactionsProcessed).
					WithField("skipped", transactionsSkipped)

				for i := byte(0x00); i < 0xff; i++ {
					if i%97 == 0 {
						logger.Infof("%d transactions processed (%d skipped)",
							transactionsProcessed, transactionsSkipped)
					}

					if !config.shouldProcessTx(i, routineIndex) {
						transactionsSkipped++
						continue
					}
					transactionsProcessed++

					prefix := hex.EncodeToString([]byte{i})

					for k := uint32(0); k < config.MapJobCount; k++ {
						url := filepath.Join(config.IndexRootSource, fmt.Sprintf("job_%d", k))
						innerJobStore, err := index.Connect(url)
						if err != nil {
							logger.WithError(err).Errorf("Failed to open index at %s", url)
							panic(err)
						}

						innerTxnIndexes, err := innerJobStore.ReadTransactions(prefix)
						if os.IsNotExist(err) {
							continue
						} else if err != nil {
							logger.WithError(err).Error("Error reading tx prefix %s", prefix)
							panic(err)
						}

						if err := finalIndexStore.MergeTransactions(prefix, innerTxnIndexes); err != nil {
							logger.WithError(err).Errorf("Error merging txs at prefix %s", prefix)
							panic(err)
						}
					}
				}

				logger.Infof("Final transaction flush (%d processed)", transactionsProcessed)
				if err = finalIndexStore.Flush(); err != nil {
					logger.Errorf("Error flushing transactions: %v", err)
					panic(err)
				}
			}(j)
		}

		wg.Wait()
	}

	return nil
}

func (cfg *ReduceConfig) shouldProcessAccount(account string, routineIndex uint32) bool {
	hash := fnv.New64a()

	// Docs state (https://pkg.go.dev/hash#Hash) that Write will never error.
	hash.Write([]byte(account))
	digest := uint32(hash.Sum64()) // discard top 32 bits

	leftHalf := digest >> 16
	rightHalf := digest & 0x0000FFFF

	log.WithField("worker", routineIndex).
		WithField("account", account).
		Debugf("Hash: %d (left=%d, right=%d)", digest, leftHalf, rightHalf)

	// Because the digest is basically a random number (given a good hash
	// function), its remainders w.r.t. the indices will distribute the work
	// fairly (and deterministically).
	return leftHalf%cfg.ReduceJobCount == cfg.JobIndex &&
		rightHalf%cfg.Workers == routineIndex
}

func (cfg *ReduceConfig) shouldProcessTx(txPrefix byte, routineIndex uint32) bool {
	hashLeft := uint32(txPrefix >> 4)
	hashRight := uint32(txPrefix & 0x0F)

	// Because the transaction hash (and thus the first byte or "prefix") is a
	// random value, its remainders w.r.t. the indices will distribute the work
	// fairly (and deterministically).
	return hashRight%cfg.ReduceJobCount == cfg.JobIndex &&
		hashLeft%cfg.Workers == routineIndex
}

// For every index that exists in `dest`, finds the corresponding index in
// `source` and merges it into `dest`'s version.
func mergeIndices(dest, source map[string]*index.CheckpointIndex) error {
	for name, index := range dest {
		// The source doesn't contain this particular index.
		//
		// This probably shouldn't happen, since during the Map step, there's no
		// way to choose which indices you want, but, strictly-speaking, it's
		// not an error, so we can just move on.
		innerIndices, ok := source[name]
		if !ok || innerIndices == nil {
			continue
		}

		if err := index.Merge(innerIndices); err != nil {
			return errors.Wrapf(err, "failed to merge index for %s", name)
		}
	}

	return nil
}
