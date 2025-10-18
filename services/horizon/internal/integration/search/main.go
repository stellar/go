package main

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/sac"
	"github.com/stellar/go/network"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

type balance struct {
	assetAddress  string
	holderAddress string
	amount        string
}

func findEvictedBalances(ctx context.Context, arch historyarchive.ArchiveInterface, seq uint32, targetAssets map[string]bool) []balance {
	reader, err := ingest.NewEvictedEntriesReader(ctx, arch, seq)
	if err != nil {
		log.Fatalf("failed to create evicted checkpoint change reader: %v", err)
	}
	defer reader.Close()
	var balances []balance
	for {
		change, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error while reading checkpoint changes: %v", err)
		}

		if change.Type != xdr.LedgerEntryTypeContractData || change.Post == nil {
			continue
		}

		le := *change.Post
		contractID, ok := le.Data.MustContractData().Contract.GetContractId()
		if !ok {
			continue
		}
		sacAddress := strkey.MustEncode(strkey.VersionByteContract, contractID[:])
		if !targetAssets[sacAddress] {
			continue
		}

		holder, amt, ok := sac.ContractBalanceFromContractData(le, network.PublicNetworkPassphrase)
		if !ok {
			continue
		}

		balances = append(balances, balance{
			assetAddress:  sacAddress,
			holderAddress: strkey.MustEncode(strkey.VersionByteContract, holder[:]),
			amount:        amt.String(),
		})
	}
	return balances
}

func main() {
	arch, err := historyarchive.NewArchivePool(
		[]string{
			"https://history.stellar.org/prd/core-live/core_live_001",
			"https://history.stellar.org/prd/core-live/core_live_002",
			"https://history.stellar.org/prd/core-live/core_live_003",
		},
		historyarchive.ArchiveOptions{},
	)
	if err != nil {
		log.Fatalf("failed to connect to pubnet archives: %v", err)
	}

	// Determine the latest published checkpoint ledger sequence
	seq, err := arch.GetLatestLedgerSequence()
	if err != nil {
		log.Fatalf("failed to get latest checkpoint sequence: %v", err)
	}

	// Initialize checkpoint change reader
	ctx := context.Background()
	reader, err := ingest.NewCheckpointChangeReader(ctx, arch, seq)
	if err != nil {
		log.Fatalf("failed to create checkpoint change reader: %v", err)
	}
	defer reader.Close()

	targetAssets := map[string]bool{
		"CDTKPWPLOURQA2SGTKTUQOWRCBZEORB4BWBOMJ3D3ZTQQSGE5F6JBQLV": true, // EURC
		"CCW67TSZV3SSS2HXMBQ5JFGCKJNXKZM7UQUWUZPUTHXSTZLEO7SJMI75": true, // USDC
	}
	var cutoffLedger uint32 = 60739011

	// Always write to stdout
	csvw := csv.NewWriter(os.Stdout)
	defer csvw.Flush()
	// header
	_ = csvw.Write([]string{"contract_id", "holder_contract_id", "amount"})

	byKeyHash := map[xdr.Hash]balance{}
	ttls := map[xdr.Hash]uint32{}

	records := 0
	for {
		change, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error while reading checkpoint changes: %v", err)
		}

		if change.Type == xdr.LedgerEntryTypeTtl {
			ttlEntry := change.Post.Data.MustTtl()
			ttls[ttlEntry.KeyHash] = uint32(ttlEntry.LiveUntilLedgerSeq)
		}

		if change.Type != xdr.LedgerEntryTypeContractData || change.Post == nil {
			continue
		}

		le := *change.Post
		contractID, ok := le.Data.MustContractData().Contract.GetContractId()
		if !ok {
			continue
		}
		sacAddress := strkey.MustEncode(strkey.VersionByteContract, contractID[:])
		if !targetAssets[sacAddress] {
			continue
		}

		holder, amt, ok := sac.ContractBalanceFromContractData(le, network.PublicNetworkPassphrase)
		if !ok {
			continue
		}

		ledgerKey, err := le.LedgerKey()
		if err != nil {
			log.Fatalf("error while extracting ledger key: %v", err)
		}
		bin, err := ledgerKey.MarshalBinary()
		if err != nil {
			log.Fatalf("error while marshaling ledger key: %v", err)
		}
		keyHash := xdr.Hash(sha256.Sum256(bin))
		if _, ok := byKeyHash[keyHash]; ok {
			log.Fatalf("duplicate key hash: %v", keyHash)
		}

		byKeyHash[keyHash] = balance{
			assetAddress:  sacAddress,
			holderAddress: strkey.MustEncode(strkey.VersionByteContract, holder[:]),
			amount:        amt.String(),
		}
	}

	for keyHash, balance := range byKeyHash {
		ttl, ok := ttls[keyHash]
		if !ok {
			log.Fatalf("missing ttl for key hash: %v", keyHash)
		}
		if ttl > cutoffLedger {
			continue
		}
		csvw.Write([]string{
			balance.assetAddress,
			balance.holderAddress,
			balance.amount,
			strconv.FormatUint(uint64(ttl), 10),
		})
	}

	for _, balance := range findEvictedBalances(ctx, arch, seq, targetAssets) {
		csvw.Write([]string{
			balance.assetAddress,
			balance.holderAddress,
			balance.amount,
			"evicted",
		})
	}

	csvw.Flush()
	if err := csvw.Error(); err != nil {
		log.Fatalf("failed writing CSV: %v", err)
	}
	log.Printf("done. wrote %d records", records)
}
