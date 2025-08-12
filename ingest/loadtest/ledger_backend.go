package loadtest

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

// LedgerBackend is used to load test ingestion.
// LedgerBackend will take a file of synthetically generated ledgers (see
// services/horizon/internal/integration/generate_ledgers_test.go) and merge those ledgers
// with real ledgers from the Stellar network. The merged ledgers will then be replayed to
// the ingesting down stream system at a configurable rate.
type LedgerBackend struct {
	config                LedgerBackendConfig
	mergedLedgersFilePath string
	mergedLedgersStream   *xdr.Stream
	startTime             time.Time
	startLedgerSeq        uint32
	nextLedgerSeq         uint32
	latestLedgerSeq       uint32
	preparedRange         ledgerbackend.Range
	cachedLedger          xdr.LedgerCloseMeta
}

// LedgerBackendConfig configures LedgerBackend
type LedgerBackendConfig struct {
	// NetworkPassphrase is the passphrase of the Stellar network from where the real ledgers
	// will be obtained
	NetworkPassphrase string
	// LedgerBackend is the source of the real ledgers
	LedgerBackend ledgerbackend.LedgerBackend
	// LedgersFilePath is a file containing the synthetic ledgers that will be combined with the
	// real ledgers and then replayed by LedgerBackend
	LedgersFilePath string
	// LedgerEntriesFilePath is a file containing the ledger entry fixtures for the synthetic ledgers
	LedgerEntriesFilePath string
	// LedgerCloseDuration is the rate at which ledgers will be replayed from LedgerBackend
	LedgerCloseDuration time.Duration
}

// NewLedgerBackend constructs an LedgerBackend instance
func NewLedgerBackend(config LedgerBackendConfig) *LedgerBackend {
	return &LedgerBackend{
		config: config,
	}
}

func (r *LedgerBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	if r.nextLedgerSeq == 0 {
		return 0, fmt.Errorf("PrepareRange() must be called before GetLatestLedgerSequence()")
	}

	return r.latestLedgerSeq, nil
}

func readLedgerEntries(path string) ([]xdr.LedgerEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	stream, err := xdr.NewZstdStream(file)
	if err != nil {
		return nil, fmt.Errorf("could not open zstd read stream: %w", err)
	}

	var entries []xdr.LedgerEntry
	for {
		var entry xdr.LedgerEntry
		err = stream.ReadOne(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not read from zstd stream: %w", err)
		}
		entries = append(entries, entry)
	}

	if err = stream.Close(); err != nil {
		return nil, fmt.Errorf("could not close zstd stream: %w", err)
	}
	return entries, nil
}

func (r *LedgerBackend) PrepareRange(ctx context.Context, ledgerRange ledgerbackend.Range) error {
	if r.nextLedgerSeq != 0 {
		if r.isPrepared(ledgerRange) {
			return nil
		}
		return fmt.Errorf("PrepareRange() already called")
	}
	generatedLedgerEntries, err := readLedgerEntries(r.config.LedgerEntriesFilePath)
	if err != nil {
		return fmt.Errorf("could not parse ledger entries file: %w", err)
	}
	generatedLedgersFile, err := os.Open(r.config.LedgersFilePath)
	if err != nil {
		return fmt.Errorf("could not open ledgers file: %w", err)
	}
	generatedLedgers, err := xdr.NewZstdStream(generatedLedgersFile)
	if err != nil {
		return fmt.Errorf("could not open zstd stream for ledgers file: %w", err)
	}

	err = r.config.LedgerBackend.PrepareRange(ctx, ledgerRange)
	if err != nil {
		return fmt.Errorf("could not prepare range using real ledger backend: %w", err)
	}
	cur := ledgerRange.From()
	firstLedger, err := r.config.LedgerBackend.GetLedger(ctx, cur)
	if err != nil {
		return fmt.Errorf("could not get ledger %v from real ledger backend: %w", cur, err)
	}
	var changes xdr.LedgerEntryChanges
	// attach all ledger entry fixtures to the first ledger in the range
	for i := 0; i < len(generatedLedgerEntries); i++ {
		entry := generatedLedgerEntries[i]
		err = UpdateLedgerSeq(&entry, func(uint32) uint32 {
			return cur
		})
		if err != nil {
			return err
		}
		changes = append(changes, xdr.LedgerEntryChange{
			Type:    xdr.LedgerEntryChangeTypeLedgerEntryCreated,
			Created: &entry,
		})
	}
	var flag xdr.Uint32 = 1
	switch firstLedger.V {
	case 1:
		firstLedger.V1.UpgradesProcessing = append(firstLedger.V1.UpgradesProcessing, xdr.UpgradeEntryMeta{
			Upgrade: xdr.LedgerUpgrade{
				Type:     xdr.LedgerUpgradeTypeLedgerUpgradeFlags,
				NewFlags: &flag,
			},
			Changes: changes,
		})
	case 2:
		firstLedger.V2.UpgradesProcessing = append(firstLedger.V2.UpgradesProcessing, xdr.UpgradeEntryMeta{
			Upgrade: xdr.LedgerUpgrade{
				Type:     xdr.LedgerUpgradeTypeLedgerUpgradeFlags,
				NewFlags: &flag,
			},
			Changes: changes,
		})
	default:
		return fmt.Errorf("unsupported ledger version %d", firstLedger.V)
	}

	mergedLedgersFile, err := os.CreateTemp("", "merged-ledgers")
	if err != nil {
		return fmt.Errorf("could not create merged ledgers file: %w", err)
	}
	log.WithField("path", mergedLedgersFile.Name()).
		Info("creating temporary merged ledgers file")

	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(mergedLedgersFile.Name())
		}
	}()
	writer, err := zstd.NewWriter(mergedLedgersFile)
	if err != nil {
		return fmt.Errorf("could not create zstd writer for merged ledgers file: %w", err)
	}

	var latestLedgerSeq uint32
	checkNetworkPassphrase := true
	for cur = cur + 1; !ledgerRange.Bounded() || cur <= ledgerRange.To(); cur++ {
		var ledger xdr.LedgerCloseMeta
		ledger, err = r.config.LedgerBackend.GetLedger(ctx, cur)
		if err != nil {
			return fmt.Errorf("could not get ledger %v from real ledger backend: %w", cur, err)
		}
		var generatedLedger xdr.LedgerCloseMeta
		if err = generatedLedgers.ReadOne(&generatedLedger); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("could not get generated ledger: %w", err)
		}
		if checkNetworkPassphrase {
			// Here we validate that the generated ledgers have the same network passphrase as the
			// ledgers sourced from the real network. This check only needs to be done once because
			// we assume all the generated ledgers have the same network passphrase.
			if err = validateNetworkPassphrase(r.config.NetworkPassphrase, ledger); err != nil {
				return err
			}
			if err = validateNetworkPassphrase(r.config.NetworkPassphrase, generatedLedger); err != nil {
				return err
			}
			checkNetworkPassphrase = false
		}
		ledgerDiff := int64(ledger.LedgerSequence()) - int64(generatedLedger.LedgerSequence())
		if err = MergeLedgers(&ledger, generatedLedger, func(cur uint32) uint32 {
			newLedgerSeq := int64(cur) + ledgerDiff
			if newLedgerSeq > math.MaxUint32 {
				panic(fmt.Sprintf(
					"value %v overflows when applying ledger diff %v",
					cur, ledgerDiff,
				))
			}
			minLedger := ledgerRange.From()
			if newLedgerSeq <= int64(minLedger) {
				// All ledger entry fixtures are attached to the very first ledger in the range.
				// Any new or updated ledger entry will occur in a later ledger sequence.
				// So, the smallest possible ledger sequence associated with any ledger entry we merge is
				// ledgerRange.From()
				return minLedger
			}
			return uint32(newLedgerSeq)
		}); err != nil {
			return fmt.Errorf("could not merge ledgers: %w", err)
		}
		if err = xdr.MarshalFramed(writer, ledger); err != nil {
			return fmt.Errorf("could not marshal ledger to stream: %w", err)
		}
		latestLedgerSeq = cur
	}
	if err = generatedLedgers.Close(); err != nil {
		return fmt.Errorf("could not close generated ledgers xdr stream: %w", err)
	}
	if err = writer.Close(); err != nil {
		return fmt.Errorf("could not close zstd writer: %w", err)
	}
	if err = mergedLedgersFile.Sync(); err != nil {
		return fmt.Errorf("could not sync merged ledgers file: %w", err)
	}

	if _, err = mergedLedgersFile.Seek(0, 0); err != nil {
		return fmt.Errorf("could not seek to beginning of merged ledgers file: %w", err)
	}
	mergedLedgersStream, err := xdr.NewZstdStream(mergedLedgersFile)
	if err != nil {
		return fmt.Errorf("could not open zstd read stream for merged ledgers file: %w", err)
	}
	cleanup = false

	r.mergedLedgersFilePath = mergedLedgersFile.Name()
	r.mergedLedgersStream = mergedLedgersStream
	// from this point, ledgers will be available at a rate of once
	// every r.ledgerCloseDuration time has elapsed
	r.startTime = time.Now()
	r.startLedgerSeq = ledgerRange.From()
	r.nextLedgerSeq = r.startLedgerSeq + 1
	r.latestLedgerSeq = latestLedgerSeq
	r.cachedLedger = firstLedger
	r.preparedRange = ledgerRange
	log.WithField("start", r.startLedgerSeq).
		WithField("end", latestLedgerSeq).
		Info("ingesting ledgers from loadtest ledger backend")
	return nil
}

func validateNetworkPassphrase(networkPassphrase string, ledger xdr.LedgerCloseMeta) error {
	// If the network passphrase which is passed into ingest.NewLedgerChangeReaderFromLedgerCloseMeta()
	// is invalid, the reader will encounter an error at some point while streaming changes.
	reader, err := ingest.NewLedgerChangeReaderFromLedgerCloseMeta(networkPassphrase, ledger)
	if err != nil {
		return err
	}
	for {
		if _, err = reader.Read(); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (r *LedgerBackend) IsPrepared(ctx context.Context, ledgerRange ledgerbackend.Range) (bool, error) {
	return r.isPrepared(ledgerRange), nil
}

func (r *LedgerBackend) isPrepared(ledgerRange ledgerbackend.Range) bool {
	if r.nextLedgerSeq == 0 {
		return false
	}

	if r.preparedRange.Bounded() != ledgerRange.Bounded() {
		return false
	}

	if ledgerRange.From() < r.cachedLedger.LedgerSequence() {
		return false
	}

	return ledgerRange.From() >= r.cachedLedger.LedgerSequence() && ledgerRange.To() <= r.preparedRange.To()
}

func (r *LedgerBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	if r.nextLedgerSeq == 0 {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("PrepareRange() must be called before GetLedger()")
	}
	if sequence < r.cachedLedger.LedgerSequence() {
		return xdr.LedgerCloseMeta{}, fmt.Errorf(
			"sequence number %v is behind the ledger stream sequence %d",
			sequence,
			r.cachedLedger.LedgerSequence(),
		)
	}
	for ; r.nextLedgerSeq <= sequence; r.nextLedgerSeq++ {
		var ledger xdr.LedgerCloseMeta
		if err := r.mergedLedgersStream.ReadOne(&ledger); err == io.EOF {
			return ledger, fmt.Errorf(
				"sequence number %v is greater than the latest ledger available",
				sequence,
			)
		} else if err != nil {
			return ledger, fmt.Errorf("could read ledger from merged ledgers stream: %w", err)
		}
		if ledger.LedgerSequence() != r.nextLedgerSeq {
			return ledger, fmt.Errorf(
				"unexpected ledger sequence (expected=%d actual=%d)",
				r.nextLedgerSeq,
				ledger.LedgerSequence(),
			)
		}
		r.cachedLedger = ledger
	}
	i := int(sequence - r.startLedgerSeq)
	// the i'th ledger will only be available after (i+1) * r.ledgerCloseDuration time has elapsed
	closeTime := r.startTime.Add(time.Duration(i+1) * r.config.LedgerCloseDuration)
	time.Sleep(time.Until(closeTime))
	return r.cachedLedger, nil
}

func (r *LedgerBackend) Close() error {
	if err := r.config.LedgerBackend.Close(); err != nil {
		return fmt.Errorf("could not close real ledger backend: %w", err)
	}
	if r.mergedLedgersStream != nil {
		// closing the stream will also close the ledgers file
		if err := r.mergedLedgersStream.Close(); err != nil {
			return fmt.Errorf("could not close merged ledgers xdr stream: %w", err)
		}
		if err := os.Remove(r.mergedLedgersFilePath); err != nil {
			return fmt.Errorf("could not remove merged ledgers file: %w", err)
		}
	}
	return nil
}

func validLedger(ledger xdr.LedgerCloseMeta) error {
	switch ledger.V {
	case 1:
		if _, ok := ledger.MustV1().TxSet.GetV1TxSet(); !ok {
			return fmt.Errorf("ledger txset %v is not supported", ledger.MustV2().TxSet.V)
		}
	case 2:
		if _, ok := ledger.MustV2().TxSet.GetV1TxSet(); !ok {
			return fmt.Errorf("ledger txset %v is not supported", ledger.MustV2().TxSet.V)
		}
	default:
		return fmt.Errorf("ledger version %v is not supported", ledger.V)
	}
	return nil
}

// MergeLedgers merges two xdr.LedgerCloseMeta instances.
// getLedgerSeq is used to determine the ledger sequence value for all ledger entries
// contained in src during the merge.
func MergeLedgers(dst *xdr.LedgerCloseMeta, src xdr.LedgerCloseMeta, getLedgerSeq func(cur uint32) uint32) error {
	if err := validLedger(*dst); err != nil {
		return err
	}
	if err := validLedger(src); err != nil {
		return err
	}
	if src.V != dst.V {
		return fmt.Errorf("src ledger version %v is incompatible with dst ledger version %v", src.V, dst.V)
	}
	if err := UpdateLedgerSeq(&src, getLedgerSeq); err != nil {
		return err
	}

	// src is merged into dst by appending all the transactions from src into dst,
	// appending all the upgrades from src into dst, and appending all the evictions
	// from src into dst
	switch dst.V {
	case 1:
		dst.V1.TxSet.V1TxSet.Phases = append(dst.V1.TxSet.V1TxSet.Phases, src.V1.TxSet.V1TxSet.Phases...)
		dst.V1.TxProcessing = append(dst.V1.TxProcessing, src.V1.TxProcessing...)
		dst.V1.UpgradesProcessing = append(dst.V1.UpgradesProcessing, src.V1.UpgradesProcessing...)
		dst.V1.EvictedKeys = append(dst.V1.EvictedKeys, src.V1.EvictedKeys...)
	case 2:
		dst.V2.TxSet.V1TxSet.Phases = append(dst.V2.TxSet.V1TxSet.Phases, src.V2.TxSet.V1TxSet.Phases...)
		dst.V2.TxProcessing = append(dst.V2.TxProcessing, src.V2.TxProcessing...)
		dst.V2.UpgradesProcessing = append(dst.V2.UpgradesProcessing, src.V2.UpgradesProcessing...)
		dst.V2.EvictedKeys = append(dst.V2.EvictedKeys, src.V2.EvictedKeys...)
	default:
		return fmt.Errorf("unexpected ledger version %v", dst.V)
	}

	return nil
}
