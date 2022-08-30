package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/collections/maps"
	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/ordered"
)

var (
	checkpointMgr = historyarchive.NewCheckpointManager(0)
)

func addIndexCommands(parent *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "index",
		Long: "Lets you view details about an index source.",
		Example: `
index view file:///tmp/indices
index view file:///tmp/indices GAGJZWQ5QT34VK3U6W6YKRYFIK6YSAXQC6BHIIYLG6X3CE5QW2KAYNJR
index stats file:///tmp/indices`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// require a subcommand - this is just a "category"
			return cmd.Help()
		},
	}

	stats := &cobra.Command{
		Use: "stats <index path>",
		Long: "Summarize the statistics (like the # of active checkpoints " +
			"or accounts). Note that this is a very read-heavy operation and " +
			"will incur download bandwidth costs if reading from remote, " +
			"billable sources.",
		Example: `stats s3://indices`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Usage()
			}

			path := args[0]
			log.Infof("Analyzing indices at %s", path)

			allCheckpoints := set.Set[uint32]{}
			allIndexNames := set.Set[string]{}
			accounts := showAccounts(path, 0)
			log.Infof("Analyzing indices for %d accounts.", len(accounts))

			// We want to summarize as much as possible on a Ctrl+C event, so
			// this handles that by setting up a context that gets cancelled on
			// SIGINT.
			//
			// https://pace.dev/blog/2020/02/17/repond-to-ctrl-c-interrupt-signals-gracefully-with-context-in-golang-by-mat-ryer.html
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				select {
				case <-c:
					cancel()
				case <-ctx.Done():
				}

				<-c
				os.Exit(1)
			}()

			mostActiveAccountChk := 0
			mostActiveAccount := ""
			for _, account := range accounts {
				if ctx.Err() != nil {
					break
				}

				activity := getIndex(path, account, "", 0)
				allCheckpoints.AddSlice(maps.Keys(activity))
				for _, names := range activity {
					allIndexNames.AddSlice(names)
				}

				if len(activity) > mostActiveAccountChk {
					mostActiveAccount = account
					mostActiveAccountChk = len(activity)
				}
			}

			log.WithField("error", ctx.Err()).
				Info("Done analyzing indices, summarizing...")

			ledgerCount := len(allCheckpoints) * int(checkpointMgr.GetCheckpointFrequency())

			log.Infof("")
			log.Infof("=== Final Summary ===")
			log.Infof("Path:     %s", path)
			log.Infof("Accounts: %d", len(accounts))
			log.Infof("Smallest checkpoint: %d", ordered.MinSlice(allCheckpoints.Slice()))
			log.Infof("Largest  checkpoint: %d", ordered.MaxSlice(allCheckpoints.Slice()))
			log.Infof("Checkpoint count:    %d (%d possible ledgers, ~%0.2f days)",
				len(allCheckpoints), ledgerCount,
				float64(ledgerCount)/(float64(60*60*24)/6.0) /* approx. ledgers per day */)
			log.Infof("Index names: %s", strings.Join(allIndexNames.Slice(), ", "))
			log.Infof("Most active account: %s (%d checkpoints)",
				mostActiveAccount, mostActiveAccountChk)

			return nil
		},
	}

	view := &cobra.Command{
		Use: "view <index path> [accounts?]",
		Long: "View the accounts in an index source or view the " +
			"checkpoints specific account(s) are active in.",
		Example: `view s3://indices
view s3://indices GAXLQGKIUAIIUHAX4GJO3J7HFGLBCNF6ZCZSTLJE7EKO5IUHGLQLMXZO
view file:///tmp/indices --limit=0 GAXLQGKIUAIIUHAX4GJO3J7HFGLBCNF6ZCZSTLJE7EKO5IUHGLQLMXZO
view gcs://indices GAXLQGKIUAIIUHAX4GJO3J7HFGLBCNF6ZCZSTLJE7EKO5IUHGLQLMXZO,GBUUWQDVEEXBJCUF5UL24YGXKJIP5EMM7KFWIAR33KQRJR34GN6HEDPV,GBYETUYNBK2ZO5MSYBJKSLDEA2ZHIXLCFL3MMWU6RHFVAUBKEWQORYKS`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 2 {
				return cmd.Usage()
			}

			path := args[0]
			log.Infof("Analyzing indices at %s", path)

			accounts := []string{}
			if len(args) == 2 {
				accounts = strings.Split(args[1], ",")
			}

			limit, err := cmd.Flags().GetUint("limit")
			if err != nil {
				return cmd.Usage()
			}

			if len(accounts) > 0 {
				indexName, err := cmd.Flags().GetString("index-name")
				if err != nil {
					return cmd.Usage()
				}

				for _, account := range accounts {
					if !strkey.IsValidEd25519PublicKey(account) &&
						!strkey.IsValidMuxedAccountEd25519PublicKey(account) {
						log.Errorf("Invalid account ID: '%s'", account)
						continue
					}

					getIndex(path, account, indexName, limit)
				}
			} else {
				showAccounts(path, limit)
			}

			return nil
		},
	}

	view.Flags().Uint("limit", 10, "a maximum number of accounts or checkpoints to show")
	view.Flags().String("index-name", "", "filter for a particular index")
	cmd.AddCommand(stats, view)

	if parent == nil {
		return cmd
	}
	parent.AddCommand(cmd)
	return parent
}

func getIndex(path, account, indexName string, limit uint) map[uint32][]string {
	freq := checkpointMgr.GetCheckpointFrequency()

	store, err := index.Connect(path)
	if err != nil {
		log.Fatal(err)
	}

	indices, err := store.Read(account)
	if err != nil {
		log.Fatal(err)
	}

	// It's better to summarize activity and then group it by index rather than
	// just show activity in each index, because there's likely a ton of overlap
	// across indices.
	activity := map[uint32][]string{}
	indexNames := []string{}

	for name, idx := range indices {
		log.Infof("Index found: '%s'", name)
		if indexName != "" && name != indexName {
			continue
		}

		indexNames = append(indexNames, name)

		checkpoint, err := idx.NextActiveBit(0)
		for err != io.EOF {
			activity[checkpoint] = append(activity[checkpoint], name)
			checkpoint, err = idx.NextActiveBit(checkpoint + 1)

			if limit > 0 && limit <= uint(len(activity)) {
				break
			}
		}
	}

	log.WithField("account", account).WithField("limit", limit).
		Infof("Activity for account:")

	for checkpoint, names := range activity {
		first := (checkpoint - 1) * freq
		last := first + freq

		nameStr := strings.Join(names, ", ")
		log.WithField("indices", nameStr).
			Infof("  - checkpoint %d, ledgers [%d, %d)", checkpoint, first, last)
	}

	log.Infof("Summary: %d active checkpoints, %d possible active ledgers",
		len(activity), len(activity)*int(freq))
	log.Infof("Checkpoint range: [%d, %d]",
		ordered.MinSlice(maps.Keys(activity)),
		ordered.MaxSlice(maps.Keys(activity)))

	return activity
}

func showAccounts(path string, limit uint) []string {
	store, err := index.Connect(path)
	if err != nil {
		log.Fatal(err)
	}

	accounts, err := store.ReadAccounts()
	if err != nil {
		log.Fatal(err)
	}

	if limit == 0 {
		limit = uint(len(accounts))
	}

	for i := uint(0); i < limit; i++ {
		log.Info(accounts[i])
	}

	return accounts
}

func slicesHaveSameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	copyA, copyB := make([]string, len(a)), make([]string, len(b))
	copy(copyA, a)
	copy(copyB, b)
	slices.Sort(copyA)
	slices.Sort(copyB)

	return slices.Equal(copyA, copyB)
}
