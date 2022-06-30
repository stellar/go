package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

var logger = supportlog.New()

func main() {
	targetUrl := flag.String("target", "gcs://horizon-archive-poc", "history archive url to write txmeta files")
	stellarCoreBinaryPath := flag.String("stellar-core-binary-path", os.Getenv("STELLAR_CORE_BINARY_PATH"), "path to the stellar core binary")
	networkPassphrase := flag.String("network-passphrase", network.TestNetworkPassphrase, "network passphrase")
	historyArchiveUrls := flag.String("history-archive-urls", "https://history.stellar.org/prd/core-testnet/core_testnet_001", "comma-separated list of history archive urls to read from")
	captiveCoreTomlPath := flag.String("captive-core-toml-path", os.Getenv("CAPTIVE_CORE_TOML_PATH"), "path to load captive core toml file from")
	startingLedger := flag.Uint("start-ledger", 0, "ledger to start export from")
	endingLedger := flag.Uint("end-ledger", math.MaxUint32, "ledger at which to stop the export")
	writeLatestPath := flag.Bool("write-latest-path", true, "update the value of the /latest path on the target")
	flag.Parse()

	logger.SetLevel(supportlog.InfoLevel)

	params := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  *networkPassphrase,
		HistoryArchiveURLs: strings.Split(*historyArchiveUrls, ","),
	}
	if *captiveCoreTomlPath == "" {
		logger.Fatal("Missing -captive-core-toml-path flag")
	}
	startLedger := uint32(*startingLedger)
	endLedger := uint32(*endingLedger)
	if endLedger < startLedger {
		logger.Fatalf("--end-ledger must be >= --start-ledger")
	}

	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromFile(*captiveCoreTomlPath, params)
	logFatalIf(err, "Invalid captive core toml")

	captiveConfig := ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          *stellarCoreBinaryPath,
		NetworkPassphrase:   params.NetworkPassphrase,
		HistoryArchiveURLs:  params.HistoryArchiveURLs,
		CheckpointFrequency: 64,
		Log:                 logger.WithField("subservice", "stellar-core"),
		Toml:                captiveCoreToml,
	}
	core, err := ledgerbackend.NewCaptive(captiveConfig)
	logFatalIf(err, "Could not create captive core instance")

	target, err := historyarchive.ConnectBackend(
		*targetUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: params.NetworkPassphrase,
		},
	)
	logFatalIf(err, "Could not connect to target")
	defer target.Close()

	latestLedger := readLatestLedger(target)

	// Build the appropriate range for the given backend state.
	var ledgerRange ledgerbackend.Range
	if startLedger == 0 {
		ledgerRange = ledgerbackend.UnboundedRange(latestLedger)
	} else if startLedger > 0 && latestLedger == 2 {
		// Special case: if the starting ledger is set but there's no ledger in
		// the backend (i.e. it's 2), the starting ledger becomes an unbounded
		// the lower bound
		ledgerRange = ledgerbackend.UnboundedRange(startLedger)
		latestLedger = startLedger
	} else {
		if startLedger >= latestLedger {
			logger.Fatalf("Invalid ledger range: %d >= %d",
				startLedger, latestLedger)
		}
		ledgerRange = ledgerbackend.BoundedRange(startLedger, latestLedger)
	}

	err = core.PrepareRange(context.Background(), ledgerRange)
	logFatalIf(err, "could not prepare unbounded range %v", latestLedger)

	logger.Infof("Unpacking ledger range [%d, %d]", *startingLedger, latestLedger)

	for nextLedger := latestLedger; nextLedger < endLedger; {
		ledger, err := core.GetLedger(context.Background(), nextLedger)
		if err != nil {
			logger.WithError(err).Warnf("could not fetch ledger %v, retrying", nextLedger)
			time.Sleep(time.Second)
			continue
		}

		if err = writeLedger(target, ledger); err != nil {
			logger.WithError(err).Warnf(
				"could not write ledger object %v, retrying",
				uint64(ledger.LedgerSequence()))
			continue
		}

		if *writeLatestPath {
			if err = writeLatestLedger(target, nextLedger); err != nil {
				logger.WithError(err).Warnf("could not write latest ledger %v", nextLedger)
			}
		}

		nextLedger++
	}

}

// readLatestLedger determines the latest ledger in the given backend (at the
// /latest path), defaulting to Ledger #2 if one doesn't exist
func readLatestLedger(backend historyarchive.ArchiveBackend) uint32 {
	r, err := backend.GetFile("latest")
	if os.IsNotExist(err) {
		return 2
	}

	logFatalIf(err, "could not open latest ledger bucket")
	defer r.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	logFatalIf(err, "could not read latest ledger")

	parsed, err := strconv.ParseUint(buf.String(), 10, 32)
	logFatalIf(err, "could not parse latest ledger: %s", buf.String())
	return uint32(parsed)
}

// writeLedger stores the given LedgerCloseMeta instance as a raw binary at the
// /ledgers/<seqNum> path. If an error is returned, it may be transient so you
// should attempt to retry.
func writeLedger(backend historyarchive.ArchiveBackend, ledger xdr.LedgerCloseMeta) error {
	blob, err := ledger.MarshalBinary()
	logFatalIf(err, "could not serialize ledger %v", ledger.LedgerSequence())

	return backend.PutFile(
		"ledgers/"+strconv.FormatUint(uint64(ledger.LedgerSequence()), 10),
		io.NopCloser(bytes.NewReader(blob)),
	)
}

func writeLatestLedger(backend historyarchive.ArchiveBackend, ledger uint32) error {
	return backend.PutFile(
		"latest",
		io.NopCloser(
			bytes.NewBufferString(
				strconv.FormatUint(uint64(ledger), 10),
			),
		),
	)
}

func logFatalIf(err error, message string, args ...interface{}) {
	if err != nil {
		logger.WithError(err).Fatalf(message, args...)
	}
}
