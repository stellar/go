package main

import (
	"bytes"
	"context"
	"github.com/stellar/go/xdr"
	"io"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/storage"

	"github.com/stellar/go/ingest/ledgerbackend"
	supportlog "github.com/stellar/go/support/log"
)

const (
	bucket = "horizon-archive-poc"
)
var logger = supportlog.New()

func main() {
	logger.SetLevel(supportlog.InfoLevel)
	binaryPath := os.Getenv("STELLAR_CORE_BINARY_PATH")

	params := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  "Test SDF Network ; September 2015",
		HistoryArchiveURLs: []string{"https://history.stellar.org/prd/core-testnet/core_testnet_001"},
	}
	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromFile(os.Getenv("CAPTIVE_CORE_TOML_PATH"),params)
	if err != nil {
		logger.WithError(err).Fatal("Invalid captive core toml")
	}

	captiveConfig := ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          binaryPath,
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

	client, err := storage.NewClient(context.Background())
	if err != nil {
		logger.WithError(err).Fatal("Could not create GCS client")
	}
	defer client.Close()

	var latestLedger uint32
	gcsBucket := client.Bucket(bucket)
	latestLedger = readLatestLedger(gcsBucket)

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

		if err = writeLedger(gcsBucket, leddger); err != nil {
			continue
		}

		if err = writeLatestLedger(gcsBucket, nextLedger); err != nil {
			logger.WithError(err).Warnf("could not write latest ledger %v", nextLedger)
		}

		nextLedger++
	}

}

func readLatestLedger(gcsBucket *storage.BucketHandle) uint32 {
	r, err := gcsBucket.Object("latest").NewReader(context.Background())
	if err == storage.ErrObjectNotExist {
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

func writeLedger(gcsBucket *storage.BucketHandle, leddger xdr.LedgerCloseMeta) error {
	writer := gcsBucket.Object("ledgers/" + strconv.FormatUint(uint64(leddger.LedgerSequence()), 10)).NewWriter(context.Background())
	blob, err := leddger.MarshalBinary()
	if err != nil {
		logger.WithError(err).Fatalf("could not serialize ledger %v", uint64(leddger.LedgerSequence()))
	}
	if _, err = io.Copy(writer, bytes.NewReader(blob)); err != nil {
		logger.WithError(err).Warnf("could not write ledger object %v, will retry", uint64(leddger.LedgerSequence()))
		return err
	}
	if err = writer.Close(); err != nil {
		logger.WithError(err).Warnf("could not close ledger object %v, will retry", uint64(leddger.LedgerSequence()))
		return err
	}
	return nil
}

func writeLatestLedger(gcsBucket *storage.BucketHandle, ledger uint32) error {
	w := gcsBucket.Object("latest").NewWriter(context.Background())
	if _, err := io.Copy(w, bytes.NewBufferString(strconv.FormatUint(uint64(ledger), 10))); err != nil {
		return nil
	}
	return w.Close()
}
