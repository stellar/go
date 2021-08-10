package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

// This program will dump all the offers from a history archive checkpoint.
// The offers dump can then be fed in to the path finding benchmarks in exp/orderbook/graph_benchmark_test.go
func main() {
	testnet := flag.Bool("testnet", false, "connect to the Stellar test network")
	checkpointLedger := flag.Int(
		"checkpoint-ledger",
		0,
		"checkpoint ledger sequence to ingest, if omitted will use latest checkpoint ledger.",
	)
	output := flag.String("output", "offers.dump", "output file which will be populated with offerss")
	flag.Parse()

	archive, err := archive(*testnet)
	if err != nil {
		panic(err)
	}
	log.SetLevel(log.InfoLevel)

	sequence := uint32(*checkpointLedger)
	if sequence == 0 {
		root, err := archive.GetRootHAS()
		if err != nil {
			log.WithField("err", err).Fatal("could not fetch root has")
		}
		sequence = root.CurrentLedger
	}
	log.WithField("ledger", sequence).
		Info("Processing entries from History Archive Snapshot")

	changeReader, err := ingest.NewCheckpointChangeReader(
		context.Background(),
		archive,
		sequence,
	)
	if err != nil {
		log.WithField("err", err).Fatal("cannot construct change reader")
	}
	defer changeReader.Close()
	file, err := os.Create(*output)
	if err != nil {
		log.WithField("err", err).Fatal("could not create offers file")
	}

	var offerXDRs []string

	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.WithField("err", err).Fatal("could not read change")
		}

		if change.Type != xdr.LedgerEntryTypeOffer {
			continue
		}
		serialized, err := xdr.MarshalBase64(change.Post.Data.MustOffer())
		if err != nil {
			log.WithField("err", err).Fatal("could not marshall offer")
		}
		offerXDRs = append(offerXDRs, serialized)
	}

	if _, err := io.Copy(file, bytes.NewBufferString(strings.Join(offerXDRs, "\n"))); err != nil {
		log.WithField("err", err).Fatal("could not write dump file")
	}
}

func archive(testnet bool) (*historyarchive.Archive, error) {
	if testnet {
		return historyarchive.Connect(
			"https://history.stellar.org/prd/core-testnet/core_testnet_001",
			historyarchive.ConnectOptions{},
		)
	}

	return historyarchive.Connect(
		fmt.Sprintf("https://history.stellar.org/prd/core-live/core_live_001/"),
		historyarchive.ConnectOptions{},
	)
}
