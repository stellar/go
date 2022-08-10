package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/stellar/go/metaarchive"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"
)

const (
	defaultCacheCount = (60 * 60 * 24) / 5 // ~24hrs worth of ledgers
)

func main() {
	log.SetLevel(log.InfoLevel)

	cmd := &cobra.Command{
		Use:  "cache",
		Long: "Manages the on-disk cache of ledgers.",
		Example: `
cache build --start 1234 --count 1000 s3://txmeta /tmp/example
cache purge /tmp/example 1234 1300
cache show /tmp/example`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// require a subcommand for now - eventually this will live under
			// the lighthorizon command
			return cmd.Help()
		},
	}
	purge := &cobra.Command{
		Use:  "purge [flags] path <start> <end>",
		Long: "Purges individual ledgers (or ranges) from the cache, or the entire cache.",
		Example: `
purge /tmp/example              # empty the whole cache
purge /tmp/example 1000         # purge one ledger
purge /tmp/example 1000 1005    # purge a ledger range`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// The first parameter must be a valid cache directory.
			// You can then pass nothing, a single ledger, or a ledger range.
			if len(args) < 1 || len(args) > 3 {
				return cmd.Usage()
			}

			var err error
			var start, end uint64
			if len(args) > 1 {
				start, err = strconv.ParseUint(args[1], 10, 32)
				if err != nil {
					cmd.Printf("Error: '%s' not a ledger sequence: %v\n", args[1], err)
					return cmd.Usage()
				}
			}
			end = start // fallback

			if len(args) == 3 {
				end, err = strconv.ParseUint(args[2], 10, 32)
				if err != nil {
					cmd.Printf("Error: '%s' not a ledger sequence: %v\n", args[2], err)
					return cmd.Usage()
				} else if end < start {
					cmd.Printf("Error: end precedes start (%d < %d)\n", end, start)
					return cmd.Usage()
				}
			}

			path := args[0]
			if start > 0 {
				return PurgeLedgers(path, uint32(start), uint32(end))
			}
			return PurgeCache(path)
		},
	}
	show := &cobra.Command{
		Use:  "show <cache path>",
		Long: "Traverses the on-disk cache and prints out cached ledger ranges.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Usage()
			}
			return ShowCache(args[0])
		},
	}
	build := &cobra.Command{
		Use:     "build [flags] <ledger source> <cache path>",
		Example: "See cache --help text",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				cmd.Println("Error: 2 positional arguments are required")
				return cmd.Usage()
			}

			start, err := cmd.Flags().GetUint32("start")
			if err != nil || start < 2 {
				cmd.Println("--start is required to be a ledger sequence")
				return cmd.Usage()
			}

			count, err := cmd.Flags().GetUint("count")
			if err != nil || count <= 0 {
				cmd.Println("--count should be a positive integer")
				return cmd.Usage()
			}
			repair, _ := cmd.Flags().GetBool("repair")
			return BuildCache(args[0], args[1], start, count, repair)
		},
	}

	build.Flags().Bool("repair", false, "attempt to purge the cache and retry ledgers that error")
	build.Flags().Uint32("start", 0, "first ledger to cache (required)")
	build.Flags().Uint("count", defaultCacheCount, "number of ledgers to cache")

	cmd.AddCommand(build, purge, show)
	cmd.Execute()
}

func BuildCache(ledgerSource, cacheDir string, start uint32, count uint, repair bool) error {
	fullStart := time.Now()
	L := log.DefaultLogger
	L.SetLevel(log.InfoLevel)
	log := L

	ctx := context.Background()
	store, err := storage.ConnectBackend(ledgerSource, storage.ConnectOptions{
		Context: ctx,
		Wrap: func(store storage.Storage) (storage.Storage, error) {
			return storage.MakeOnDiskCache(store, cacheDir, count)
		},
	})
	if err != nil {
		log.Errorf("Couldn't create local cache for '%s' at '%s': %v",
			ledgerSource, cacheDir, err)
		return err
	}

	log.Infof("Connected to ledger source at %s", ledgerSource)
	log.Infof("Connected to ledger cache at %s", cacheDir)

	source := metaarchive.NewMetaArchive(store)
	log.Infof("Filling local cache of ledgers at %s...", cacheDir)
	log.Infof("Ledger range: [%d, %d] (%d ledgers)",
		start, uint(start)+count-1, count)

	successful := uint(0)
	for i := uint(0); i < count; i++ {
		ledgerSeq := start + uint32(i)

		// do "best effort" caching, skipping if too slow
		dlCtx, dlCancel := context.WithTimeout(ctx, 10*time.Second)
		start := time.Now()

		_, err := source.GetLedger(dlCtx, ledgerSeq) // this caches
		dlCancel()

		if err != nil {
			if repair && strings.Contains(err.Error(), "xdr") {
				log.Warnf("Caching ledger %d failed, purging & retrying: %v", ledgerSeq, err)
				store.(*storage.OnDiskCache).Evict(fmt.Sprintf("ledgers/%d", ledgerSeq))
				i-- // retry
			} else {
				log.Warnf("Caching ledger %d failed, skipping: %v", ledgerSeq, err)
				log.Warn("If you see an XDR decoding error, the cache may be corrupted.")
				log.Warnf("Run '%s purge %d' and try again, or pass --repair",
					filepath.Base(os.Args[0]), ledgerSeq)
			}
			continue
		} else {
			successful++
		}

		duration := time.Since(start)
		if duration > 2*time.Second {
			log.WithField("duration", duration).
				Warnf("Downloading ledger %d took a while.", ledgerSeq)
		}

		log = log.WithField("failures", 1+i-successful)
		if successful%97 == 0 {
			log.Infof("Cached %d/%d ledgers (%0.1f%%)", successful, count,
				100*float64(successful)/float64(count))
		}
	}

	duration := time.Since(fullStart)
	log.WithField("duration", duration).
		Infof("Cached %d ledgers into %s", successful, cacheDir)

	return nil
}

func PurgeLedgers(cacheDir string, start, end uint32) error {
	base := filepath.Join(cacheDir, "ledgers")

	successful := 0
	for i := start; i <= end; i++ {
		ledgerPath := filepath.Join(base, strconv.FormatUint(uint64(i), 10))
		if err := os.Remove(ledgerPath); err != nil {
			log.Warnf("Failed to remove cached ledger %d: %v", i, err)
			continue
		} else {
			log.Debugf("Purged ledger from %s", ledgerPath)
			successful++
		}
	}

	log.Infof("Purged %d cached ledgers from %s", successful, cacheDir)
	return nil
}

func PurgeCache(cacheDir string) error {
	if err := os.RemoveAll(cacheDir); err != nil {
		log.Warnf("Failed to remove cache directory (%s): %v", cacheDir, err)
		return err
	}

	log.Infof("Purged cache at %s", cacheDir)
	return nil
}

func ShowCache(cacheDir string) error {
	files, err := ioutil.ReadDir(filepath.Join(cacheDir, "ledgers"))
	if err != nil {
		log.Errorf("Failed to read cache: %v", err)
		return err
	}

	ledgers := make([]uint32, 0, len(files))

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		// If the name can be converted to a ledger sequence, track it.
		if seq, errr := strconv.ParseUint(f.Name(), 10, 32); errr == nil {
			ledgers = append(ledgers, uint32(seq))
		}
	}

	log.Infof("Analyzed cache at %s: %d cached ledgers.", cacheDir, len(ledgers))
	if len(ledgers) == 0 {
		return nil
	}

	// Find consecutive ranges of ledgers in the cache
	log.Infof("Cached ranges:")
	firstSeq, lastSeq := ledgers[0], ledgers[0]
	for i := 1; i < len(ledgers); i++ {
		if ledgers[i]-1 != lastSeq {
			log.Infof(" - [%d, %d]", firstSeq, lastSeq)
			firstSeq = ledgers[i]
		}
		lastSeq = ledgers[i]
	}

	log.Infof(" - [%d, %d]", firstSeq, lastSeq)
	return nil
}
