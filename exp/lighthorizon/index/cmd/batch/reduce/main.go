package main

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/support/log"
)

var (
	// Should we use runtime.NumCPU() for a reasonable default?
	parallel = uint32(16)
)

func main() {
	log.SetLevel(log.InfoLevel)
	jobIndexString := os.Getenv("AWS_BATCH_JOB_ARRAY_INDEX")
	if jobIndexString == "" {
		panic("AWS_BATCH_JOB_ARRAY_INDEX env required")
	}

	mapJobsString := os.Getenv("MAP_JOBS")
	if mapJobsString == "" {
		panic("MAP_JOBS env required")
	}

	reduceJobsString := os.Getenv("REDUCE_JOBS")
	if mapJobsString == "" {
		panic("REDUCE_JOBS env required")
	}

	jobIndex, err := strconv.ParseUint(jobIndexString, 10, 64)
	if err != nil {
		panic(err)
	}

	mapJobs, err := strconv.ParseUint(mapJobsString, 10, 64)
	if err != nil {
		panic(err)
	}

	reduceJobs, err := strconv.ParseUint(reduceJobsString, 10, 64)
	if err != nil {
		panic(err)
	}

	var (
		mutex        sync.Mutex
		doneAccounts map[string]struct{} = map[string]struct{}{}
	)

	indexStore, err := index.NewS3Store(&aws.Config{Region: aws.String("us-east-1")}, "", parallel)
	if err != nil {
		panic(err)
	}

	for i := uint64(0); i < mapJobs; i++ {
		outerJobStore, err := index.NewS3Store(
			&aws.Config{Region: aws.String("us-east-1")},
			fmt.Sprintf("job_%d", i),
			parallel,
		)
		if err != nil {
			panic(err)
		}

		accounts, err := outerJobStore.ReadAccounts()
		if err != nil {
			// TODO: in final version this should be critical error, now just skip it
			if err == os.ErrNotExist {
				log.Errorf("Job %d is unavailable - TODO fix", i)
				continue
			}
			panic(err)
		}

		log.Info("Outer job ", i, " accounts ", len(accounts))

		ch := make(chan string, parallel)
		go func() {
			for _, account := range accounts {
				mutex.Lock()
				_, ok := doneAccounts[account]
				mutex.Unlock()
				if ok {
					// Account index already merged in the previous outer job
					continue
				}
				ch <- account
			}
			close(ch)
		}()

		var wg sync.WaitGroup
		wg.Add(int(parallel))
		for j := uint32(0); j < parallel; j++ {
			go func(routine uint32) {
				defer wg.Done()
				var skipped, processed uint64
				for account := range ch {
					if (processed+skipped)%1000 == 0 {
						log.Infof(
							"outer: %d, routine: %d, processed: %d, skipped: %d, all account in outer job: %d\n",
							i, routine, processed, skipped, len(accounts),
						)
					}

					hash := fnv.New64a()
					_, err = hash.Write([]byte(account))
					if err != nil {
						panic(err)
					}

					hashSum := hash.Sum64()
					hashLeft := uint32(hashSum >> 4)
					hashRight := uint32(0x0000ffff & hashSum)

					if hashRight%uint32(reduceJobs) != uint32(jobIndex) {
						// This job is not merging this account
						skipped++
						continue
					}

					if hashLeft%uint32(parallel) != uint32(routine) {
						// This go routine is not merging this account
						skipped++
						continue
					}

					outerAccountIndexes, err := outerJobStore.Read(account)
					if err != nil {
						// TODO: in final version this should be critical error, now just skip it
						if err == os.ErrNotExist {
							log.Errorf("Account %d is unavailable - TODO fix", account)
							continue
						}
						panic(err)
					}

					for k := uint64(i + 1); k < mapJobs; k++ {
						innerJobStore, err := index.NewS3Store(
							&aws.Config{Region: aws.String("us-east-1")},
							fmt.Sprintf("job_%d", k),
							parallel,
						)
						if err != nil {
							panic(err)
						}

						innerAccountIndexes, err := innerJobStore.Read(account)
						if err != nil {
							if err == os.ErrNotExist {
								continue
							}
							panic(err)
						}

						for name, index := range outerAccountIndexes {
							if innerAccountIndexes[name] == nil {
								continue
							}
							err := index.Merge(innerAccountIndexes[name])
							if err != nil {
								panic(err)
							}
						}
					}

					// Save merged index
					indexStore.AddParticipantToIndexesNoBackend(account, outerAccountIndexes)

					// Mark account as done
					mutex.Lock()
					doneAccounts[account] = struct{}{}
					mutex.Unlock()
					processed++

					if processed%200 == 0 {
						log.Infof("Flushing %d, processed %d", routine, processed)
						err = indexStore.Flush()
						if err != nil {
							panic(err)
						}
					}
				}

				log.Infof("Flushing Accounts %d, processed %d", routine, processed)
				err = indexStore.Flush()
				if err != nil {
					panic(err)
				}

				// Merge the transaction indexes
				// There's 256 files, (one for each first byte of the txn hash)
				processed = 0
				for i := byte(0x00); i < 0xff; i++ {
					hashLeft := uint32(i >> 4)
					hashRight := uint32(0x0f & i)
					if hashRight%uint32(reduceJobs) != uint32(jobIndex) {
						// This job is not merging this prefix
						skipped++
						continue
					}

					if hashLeft%uint32(parallel) != uint32(routine) {
						// This go routine is not merging this prefix
						skipped++
						continue
					}
					processed++

					prefix := hex.EncodeToString([]byte{i})

					for k := uint64(0); k < mapJobs; k++ {
						innerJobStore, err := index.NewS3Store(
							&aws.Config{Region: aws.String("us-east-1")},
							fmt.Sprintf("job_%d", k),
							parallel,
						)
						if err != nil {
							panic(err)
						}

						innerTxnIndexes, err := innerJobStore.ReadTransactions(prefix)
						if err != nil {
							if err == os.ErrNotExist {
								continue
							}
							panic(err)
						}

						if err := indexStore.MergeTransactions(prefix, innerTxnIndexes); err != nil {
							panic(err)
						}
					}
				}

				log.Infof("Flushing Transactions %d, processed %d", routine, processed)
				err = indexStore.Flush()
				if err != nil {
					panic(err)
				}
			}(j)
		}

		wg.Wait()
	}
}
