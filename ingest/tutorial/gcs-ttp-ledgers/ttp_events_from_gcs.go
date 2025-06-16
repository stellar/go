package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"os"
	"strconv"
	"strings"
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
	var ledgerNumbers, outputFile string
	var startLedger, endLedger uint

	flag.StringVar(&ledgerNumbers, "ledgers", "", "Comma-separated list of ledger numbers (e.g., '123,456,789')")
	flag.UintVar(&startLedger, "start-ledger", 0, "Starting ledger sequence number (alternative to -ledgers)")
	flag.UintVar(&endLedger, "end-ledger", 0, "Ending ledger sequence number (used with -start-ledger)")
	flag.StringVar(&outputFile, "output", "", "Output file path (if not specified, outputs to console)")
	flag.Parse()

	// Parse ledger numbers
	var ledgers []uint32
	var err error

	if ledgerNumbers != "" {
		// Parse comma-separated ledger numbers
		ledgers, err = parseLedgerNumbers(ledgerNumbers)
		if err != nil {
			log.Fatalf("Error parsing ledger numbers: %v", err)
		}
	} else if startLedger != 0 && endLedger != 0 {
		// Generate range of ledgers
		for i := startLedger; i <= endLedger; i++ {
			ledgers = append(ledgers, uint32(i))
		}
	} else {
		log.Fatal("Either -ledgers or both -start-ledger and -end-ledger are required")
	}

	if len(ledgers) == 0 {
		log.Fatal("No ledgers specified")
	}

	// Set up output writer
	var output io.Writer = os.Stdout
	var outputFileHandle *os.File

	if outputFile != "" {
		outputFileHandle, err = os.Create(outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer outputFileHandle.Close()
		output = outputFileHandle
		log.Printf("Output will be written to: %s", outputFile)
	} else {
		log.Println("Output will be written to console")
	}

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
		log.Fatal(err)
	}
	defer backend.Close()

	// Determine the range for preparation
	minLedger := ledgers[0]
	maxLedger := ledgers[0]
	for _, ledger := range ledgers {
		if ledger < minLedger {
			minLedger = ledger
		}
		if ledger > maxLedger {
			maxLedger = ledger
		}
	}

	ledgerRange := ledgerbackend.BoundedRange(minLedger, maxLedger)
	err = backend.PrepareRange(ctx, ledgerRange)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to prepare range: %v", ledgerRange))
	}

	log.Printf("Starting ledger retrieval for %d ledgers", len(ledgers))

	// Set up progress tracking
	processedCount := 0
	results := make([]LedgerResult, 0, len(ledgers))

	// Process each ledger
	for _, ledgerSeq := range ledgers {
		ledger, err := backend.GetLedger(ctx, ledgerSeq)
		if err != nil {
			log.Printf("Failed to retrieve ledger %d: %v", ledgerSeq, err)
			results = append(results, LedgerResult{
				Ledger:    ledgerSeq,
				Status:    "RETRIEVAL_ERROR",
				Error:     err,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			continue
		}

		processedCount++

		// Verify the ledger
		res := token_transfer.VerifyEvents(ledger, network.PublicNetworkPassphrase, true)

		var result LedgerResult
		if res != nil {
			result = LedgerResult{
				Ledger:    ledgerSeq,
				Status:    "VERIFICATION_FAILED",
				Error:     res,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			log.Printf("Verification error in ledger %d", ledgerSeq)
			//printProtoEvents(res.Events)
		} else {
			result = LedgerResult{
				Ledger:    ledgerSeq,
				Status:    "SUCCESS",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			//printProtoEvents(res.Events)
		}

		results = append(results, result)

	}

	// Output results
	outputResults(output, results)

}

// LedgerResult represents the result of processing a single ledger
type LedgerResult struct {
	Ledger    uint32
	Status    string
	Error     error
	Timestamp string
}

// parseLedgerNumbers parses a comma-separated string of ledger numbers
func parseLedgerNumbers(input string) ([]uint32, error) {
	parts := strings.Split(input, ",")
	ledgers := make([]uint32, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		num, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ledger number '%s': %v", part, err)
		}

		ledgers = append(ledgers, uint32(num))
	}

	return ledgers, nil
}

// outputResults writes the results to the specified output
func outputResults(output io.Writer, results []LedgerResult) {
	// Write summary
	fmt.Fprintf(output, "=== LEDGER PROCESSING SUMMARY ===\n")
	fmt.Fprintf(output, "Total ledgers processed: %d\n", len(results))

	// Count statuses
	statusCounts := make(map[string]int)
	for _, result := range results {
		statusCounts[result.Status]++
	}

	for status, count := range statusCounts {
		fmt.Fprintf(output, "%s: %d\n", status, count)
	}

	fmt.Fprintf(output, "\n=== DETAILED RESULTS ===\n")

	// Output detailed results as JSON
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")

	for _, result := range results {
		fmt.Printf("Ledger:%d, Status: %s, TimeStamp: %v\n", result.Ledger, result.Status, result.Timestamp)
		if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
		}
		fmt.Println("------")

	}
}

// Retaining this function to help with debugging
func printProtoEvents(events []*token_transfer.TokenTransferEvent) {
	for _, event := range events {
		jsonBytes, _ := protojson.MarshalOptions{
			Multiline:         true,
			EmitDefaultValues: true,
			Indent:            "  ",
		}.Marshal(event)
		fmt.Printf("### Event Type : %v\n", event.GetEventType())
		fmt.Println(string(jsonBytes))
		fmt.Println("###")
	}
}
