package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/util"
	"github.com/stellar/go/support/log"
)

var (
	from, to, size int32
	workers        int
)

type ledgerRange struct {
	from, to int32
}

var dbReingestRangeCmd = &cobra.Command{
	Use:   "range",
	Short: "reingests range of ledgers",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()

		if from == 0 || to == 0 || workers == 0 || size == 0 {
			cmd.Help()
			os.Exit(0)
		}

		if to < from {
			log.Error("Invalid range")
			os.Exit(1)
		}

		i := ingestSystem(ingest.Config{
			IngestFailedTransactions: config.IngestFailedTransactions,
		})

		var pool util.WorkersPool
		log.Info("Creating work...")
		for current := from; current <= to; current += size {
			lr := ledgerRange{from: current, to: current + size - 1}
			if lr.to > to {
				lr.to = to
			}
			pool.AddWork(lr)
		}

		allJobs := pool.WorkSize()

		pool.SetWorker(func(workerID int, job interface{}) {
			lr, ok := job.(ledgerRange)
			if !ok {
				log.Error("job is not a ledgerRange")
				os.Exit(1)
			}

			localLog := log.WithFields(log.F{
				"id":   workerID,
				"from": lr.from,
				"to":   lr.to,
			})

			localLog.Info("Worker starting range...")

			_, err := i.ReingestRange(lr.from, lr.to)
			if err != nil {
				localLog.WithField("err", err).Error("Worker failed range, work will processed again")
				// Add the work again
				pool.AddWork(lr)
				return
			}
			localLog.Info("Worker finished range")
		})

		log.Infof("Starting %d workers...", workers)
		go func() {
			c := time.Tick(10 * time.Second)
			for range c {
				log.WithField("progress", float32(allJobs-pool.WorkSize())/float32(allJobs)*100).Info("Work status")
			}
		}()
		pool.Start(workers)
		log.Info("Done")
	},
}

func init() {
	dbReingestCmd.AddCommand(dbReingestRangeCmd)
	dbReingestRangeCmd.Flags().Int32VarP(&from, "from", "f", 0, "Start of the ledger range")
	dbReingestRangeCmd.Flags().Int32VarP(&to, "to", "t", 0, "End of the ledger range (included)")
	dbReingestRangeCmd.Flags().Int32VarP(&size, "size", "s", 10000, "Size of the work (how many ledgers a single worker should reingest at once)")
	dbReingestRangeCmd.Flags().IntVarP(&workers, "workers", "w", 10, "Number of workers")
}
