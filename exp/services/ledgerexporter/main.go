package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"

	"github.com/stellar/go/ingest/ledgerbackend"
	supportlog "github.com/stellar/go/support/log"
)

var logger = supportlog.New()

func main() {
	targetUrl := flag.String("target", "gcs://horizon-archive-poc", "history archive url to write txmeta files")
	stellarCoreBinaryPath := flag.String("stellar-core-binary-path", os.Getenv("STELLAR_CORE_BINARY_PATH"), "path to the stellar core binary")
	networkPassphrase := flag.String("network-passphrase", "Test SDF Network ; September 2015", "network passphrase")
	historyArchiveUrls := flag.String("history-archive-urls", "https://history.stellar.org/prd/core-testnet/core_testnet_001", "comma-separated list of history archive urls to read from")
	flag.Parse()

	logger.SetLevel(supportlog.InfoLevel)

	params := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  *networkPassphrase,
		HistoryArchiveURLs: strings.Split(*historyArchiveUrls, ","),
	}
	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromFile(os.Getenv("CAPTIVE_CORE_TOML_PATH"), params)
	if err != nil {
		logger.WithError(err).Fatal("Invalid captive core toml")
	}

	captiveConfig := ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          *stellarCoreBinaryPath,
		NetworkPassphrase:   params.NetworkPassphrase,
		HistoryArchiveURLs:  params.HistoryArchiveURLs,
		CheckpointFrequency: 64,
		Log:                 logger.WithField("subservice", "stellar-core"),
		Toml:                captiveCoreToml,
	}
	core, err := ledgerbackend.NewCaptive(captiveConfig)
	if err != nil {
		logger.WithError(err).Fatal("Could not create captive core instance")
	}

	target, err := historyarchive.ConnectBackend(
		*targetUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: params.NetworkPassphrase,
		},
	)
	defer target.Close()

	var latestLedger uint32
	latestLedger = readLatestLedger(target)

	nextLedger := latestLedger + 1
	if err := core.PrepareRange(context.Background(), ledgerbackend.UnboundedRange(latestLedger)); err != nil {
		logger.WithError(err).Fatalf("could not prepare unbounded range %v", nextLedger)
	}

	for {
		leddger, err := core.GetLedger(context.Background(), nextLedger)
		if err != nil {
			logger.WithError(err).Warnf("could not fetch ledger %v, will retry", nextLedger)
			time.Sleep(time.Second)
			continue
		}

		if err = writeLedger(target, leddger); err != nil {
			continue
		}

		if err = writeLatestLedger(target, nextLedger); err != nil {
			logger.WithError(err).Warnf("could not write latest ledger %v", nextLedger)
		}

		nextLedger++
	}

}

func readLatestLedger(backend historyarchive.ArchiveBackend) uint32 {
	r, err := backend.GetFile("latest")
	if err == os.ErrNotExist {
		return 2
	} else if err != nil {
		logger.WithError(err).Fatal("could not open latest ledger bucket")
	} else {
		defer r.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			logger.WithError(err).Fatal("could not read latest ledger")
		}
		if parsed, err := strconv.ParseUint(buf.String(), 10, 32); err != nil {
			logger.WithError(err).Fatalf("could not parse latest ledger: %s", buf.String())
		} else {
			return uint32(parsed)
		}
	}
	return 0
}

func writeLedger(backend historyarchive.ArchiveBackend, leddger xdr.LedgerCloseMeta) error {
	blob, err := leddger.MarshalBinary()
	if err != nil {
		logger.WithError(err).Fatalf("could not serialize ledger %v", uint64(leddger.LedgerSequence()))
	}
	err = backend.PutFile(
		"ledgers/"+strconv.FormatUint(uint64(leddger.LedgerSequence()), 10),
		ioutil.NopCloser(bytes.NewReader(blob)),
	)
	if err != nil {
		logger.WithError(err).Warnf("could not write ledger object %v, will retry", uint64(leddger.LedgerSequence()))
	}
	return err
}

func writeLatestLedger(backend historyarchive.ArchiveBackend, ledger uint32) error {
	return backend.PutFile(
		"latest",
		ioutil.NopCloser(
			bytes.NewBufferString(
				strconv.FormatUint(uint64(ledger), 10),
			),
		),
	)
}
