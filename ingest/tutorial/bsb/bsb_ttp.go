package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/processors/token_transfer"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/errors"
	"log"
)

func main() {
	// Command-line flags
	var startLedger, endLedger uint
	var outputDir, errorsDir, badLedgersFile, datePrefix string

	flag.UintVar(&startLedger, "start-ledger", 0, "Starting ledger sequence number")
	flag.UintVar(&endLedger, "end-ledger", 0, "Ending ledger sequence number")
	flag.StringVar(&outputDir, "output-dir", ".", "Directory to store output files")
	flag.StringVar(&errorsDir, "errors-dir", "", "Directory to store individual error files")
	flag.StringVar(&badLedgersFile, "bad-ledgers-file", "", "File to record bad ledgers (verification failures)")
	flag.StringVar(&datePrefix, "date-prefix", "", "Date prefix for file naming (YYYY-MM-DD)")
	flag.Parse()

	// Validate required args
	if startLedger == 0 || endLedger == 0 {
		log.Fatal("start-ledger and end-ledger are required")
	}

	// If errors directory not specified, create default location
	if errorsDir == "" {
		errorsDir = filepath.Join(outputDir, "ERRORS")
	}

	// If bad ledgers file not specified, create default name
	if badLedgersFile == "" && datePrefix != "" {
		badLedgersFile = filepath.Join(outputDir, fmt.Sprintf("%s-bad-ledgers.txt", datePrefix))
	} else if badLedgersFile == "" {
		badLedgersFile = filepath.Join(outputDir, "bad-ledgers.txt")
	}

	// Create directories if they don't exist
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	err = os.MkdirAll(errorsDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create errors directory: %v", err)
	}

	// Create bad ledgers file
	badLedgersF, err := os.Create(badLedgersFile)
	if err != nil {
		log.Fatalf("Failed to create bad ledgers file: %v", err)
	}
	defer badLedgersF.Close()

	ctx := context.Background()

	// Configure the datastore
	datastoreConfig := datastore.DataStoreConfig{
		Type: "GCS", // Google Cloud Storage as the backend
		Params: map[string]string{
			"destination_bucket_path": "sdf-ledger-close-meta-rc1/ledgers/pubnet", // Replace with actual GCS bucket path
		},
		Schema: datastore.DataStoreSchema{
			LedgersPerFile:    1,     // Defines how many ledgers per file
			FilesPerPartition: 64000, // Defines partition size
		},
	}

	// Initialize the datastore
	dataStore, err := datastore.NewDataStore(ctx, datastoreConfig)
	if err != nil {
		cleanupAndExit(outputDir, errorsDir, badLedgersFile)
		log.Fatal(errors.Wrap(err, "failed to create datastore"))
	}
	defer dataStore.Close()

	// Configure the BufferedStorageBackend
	backendConfig := ledgerbackend.BufferedStorageBackendConfig{
		BufferSize: 100,             // Number of files to buffer in memory
		NumWorkers: 10,              // Concurrent download workers
		RetryLimit: 3,               // Maximum retry attempts on failure
		RetryWait:  5 * time.Second, // Wait time between retries
	}

	// Initialize the backend
	backend, err := ledgerbackend.NewBufferedStorageBackend(backendConfig, dataStore)
	if err != nil {
		cleanupAndExit(outputDir, errorsDir, badLedgersFile)

		log.Fatal(errors.Wrap(err, "failed to create buffered storage backend"))
	}
	defer backend.Close()

	start, end := uint32(startLedger), uint32(endLedger)
	// Define the ledger range to process

	ledgerRange := ledgerbackend.BoundedRange(start, end)
	totalLedgers := int(ledgerRange.To() - ledgerRange.From() + 1)
	err = backend.PrepareRange(ctx, ledgerRange)
	if err != nil {
		cleanupAndExit(outputDir, errorsDir, badLedgersFile)
		log.Fatal(errors.Wrapf(err, "failed to prepare range:%v", ledgerRange))
	}

	log.Printf("Starting ledger retrieval for range: %d - %d", ledgerRange.From(), ledgerRange.To())

	// Set up progress tracking
	processedCount := 0
	lastReportedThousand := 0
	startTime := time.Now()

	// Iterate through the ledger sequence
	for ledgerSeq := ledgerRange.From(); ledgerSeq <= ledgerRange.To(); ledgerSeq++ {
		ledger, err := backend.GetLedger(ctx, ledgerSeq)
		if err != nil {
			cleanupAndExit(outputDir, errorsDir, badLedgersFile)
			log.Fatalf("Failed to retrieve ledger %d: %v", ledgerSeq, err)
		}

		processedCount++
		currentThousand := processedCount / 1000

		// Report progress after each ~1000 ledgers (approximating an hour of ledgers)
		if currentThousand > lastReportedThousand {
			elapsed := time.Since(startTime)
			progress := float64(processedCount) / float64(totalLedgers) * 100
			ledgersPerSec := float64(processedCount) / elapsed.Seconds()

			timeRemaining := time.Duration(float64(totalLedgers-processedCount)/ledgersPerSec) * time.Second

			log.Printf("Progress: %d/%d ledgers processed (%.2f%%) - %.2f ledgers/sec - ~%s remaining",
				processedCount, totalLedgers, progress, ledgersPerSec,
				formatDuration(timeRemaining))

			lastReportedThousand = currentThousand
		}

		ttpVerificationRes := token_transfer.VerifyEvents(ledger, network.PublicNetworkPassphrase, true)
		if ttpVerificationRes != nil {
			// This is a "bad ledger" - write to bad ledgers file
			badLedgerEntry := fmt.Sprintf("%d\n", ledgerSeq)
			_, err := badLedgersF.WriteString(badLedgerEntry)
			if err != nil {
				log.Printf("Warning: Failed to write to bad ledgers file: %v", err)
			}

			// Create individual error file for this ledger's error
			errorFileName := fmt.Sprintf("%s-error-ledger-%d.json", datePrefix, ledgerSeq)
			errorFilePath := filepath.Join(errorsDir, errorFileName)

			// Create a structured error record
			errorRecord := map[string]interface{}{
				"ledger":    ledgerSeq,
				"timestamp": time.Now().Format(time.RFC3339),
				"error":     fmt.Sprintf("%v", ttpVerificationRes),
			}

			// Write error to individual file as JSON
			errorFile, err := os.Create(errorFilePath)
			if err != nil {
				log.Printf("Failed to create error file for ledger %d: %v", ledgerSeq, err)
				continue
			}

			encoder := json.NewEncoder(errorFile)
			encoder.SetIndent("", "  ")
			err = encoder.Encode(errorRecord)
			if err != nil {
				log.Printf("Failed to write error data for ledger %d: %v", ledgerSeq, err)
			}

			errorFile.Close()

			log.Printf("Verification error in ledger %d: details written to %s", ledgerSeq, errorFilePath)
		}
	}

	log.Printf("Processing complete for ledger range: %d - %d", ledgerRange.From(), ledgerRange.To())
}

// Helper function to clean up and exit on fatal errors
func cleanupAndExit(outputDir, errorsDir, badLedgersFile string) {
	log.Println("Cleaning up files due to fatal error...")

	// Remove bad ledgers file
	if err := os.Remove(badLedgersFile); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Failed to remove bad ledgers file: %v", err)
	}

	// Remove errors directory and contents
	if err := os.RemoveAll(errorsDir); err != nil {
		log.Printf("Warning: Failed to remove errors directory: %v", err)
	}
}

// Helper function to format duration in a human-readable way
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
