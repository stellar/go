package loadtest

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

// ErrLoadTestDone indicates that the load test has run to completion.
var ErrLoadTestDone = fmt.Errorf("the load test is done")

// LedgerBackend is used to load test ingestion.
// LedgerBackend will take a file of synthetically generated ledgers (see
// services/horizon/internal/integration/generate_ledgers_test.go) and replay
// them to the downstream ingesting system at a configurable rate.
// It is also possible to merge the synthetically generated ledgers with real
// ledgers from the network. To enable the merging behavior, configure the
// LedgerBackend field in LedgerBackendConfig.
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
	done                  bool
	lock                  sync.RWMutex
}

// LedgerBackendConfig configures LedgerBackend
type LedgerBackendConfig struct {
	// NetworkPassphrase is the passphrase of the Stellar network from where the real ledgers
	// will be obtained
	NetworkPassphrase string
	// LedgerBackend is an optional parameter. When LedgerBackend is configured, ledgers from
	// LedgerBackend will be merged with the synthetic ledgers from LedgersFilePath.
	LedgerBackend ledgerbackend.LedgerBackend
	// LedgersFilePath is a file containing the synthetic ledgers that will be replayed to
	// the downstream ingesting system.
	LedgersFilePath string
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
	r.lock.RLock()
	defer r.lock.RUnlock()

	if r.nextLedgerSeq == 0 {
		return 0, fmt.Errorf("PrepareRange() must be called before GetLatestLedgerSequence()")
	}

	return r.latestLedgerSeq, nil
}

func (r *LedgerBackend) PrepareRange(ctx context.Context, ledgerRange ledgerbackend.Range) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.done {
		return ErrLoadTestDone
	}
	if r.nextLedgerSeq != 0 {
		if r.isPrepared(ledgerRange) {
			return nil
		}
		return fmt.Errorf("PrepareRange() already called")
	}
	generatedLedgersFile, err := os.Open(r.config.LedgersFilePath)
	if err != nil {
		return fmt.Errorf("could not open ledgers file: %w", err)
	}
	generatedLedgers, err := xdr.NewZstdStream(generatedLedgersFile)
	if err != nil {
		return fmt.Errorf("could not open zstd stream for ledgers file: %w", err)
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
	var firstLedger xdr.LedgerCloseMeta
	var validatedGeneratedLedgers, validatedNetworkLedgers bool
	for cur := ledgerRange.From(); !ledgerRange.Bounded() || cur <= ledgerRange.To(); cur++ {
		var generatedLedger xdr.LedgerCloseMeta
		if err = generatedLedgers.ReadOne(&generatedLedger); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("could not get generated ledger: %w", err)
		}
		if !validatedGeneratedLedgers && generatedLedger.CountTransactions() > 0 {
			// Here we validate that the generated ledgers have the same network passphrase as the
			// ledgers sourced from the real network. This check only needs to be done once because
			// we assume all the generated ledgers have the same network passphrase.
			if err = validateNetworkPassphrase(r.config.NetworkPassphrase, generatedLedger); err != nil {
				return err
			}
			validatedGeneratedLedgers = true
		}

		ledgerDiff := int64(cur) - int64(generatedLedger.LedgerSequence())
		setLedgerSeq := func(cur uint32) uint32 {
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
		}

		var ledger xdr.LedgerCloseMeta
		if r.config.LedgerBackend != nil {
			if cur == ledgerRange.From() {
				err = r.config.LedgerBackend.PrepareRange(ctx, ledgerRange)
				if err != nil {
					return fmt.Errorf("could not prepare range using real ledger backend: %w", err)
				}
			}
			ledger, err = r.config.LedgerBackend.GetLedger(ctx, cur)
			if err != nil {
				return fmt.Errorf("could not get ledger %v from real ledger backend: %w", cur, err)
			}
			if !validatedNetworkLedgers && ledger.CountTransactions() > 0 {
				if err = validateNetworkPassphrase(r.config.NetworkPassphrase, ledger); err != nil {
					return err
				}
				validatedNetworkLedgers = true
			}
			if err = MergeLedgers(&ledger, generatedLedger, setLedgerSeq); err != nil {
				return fmt.Errorf("could not merge ledgers: %w", err)
			}
		} else {
			ledger = generatedLedger
			if err = UpdateLedgerSeqInLedgerEntries(&ledger, setLedgerSeq); err != nil {
				return fmt.Errorf("could not update ledger seq: %w", err)
			}
			switch ledger.V {
			case 0:
				ledger.V0.LedgerHeader.Header.LedgerSeq = xdr.Uint32(cur)
			case 1:
				ledger.V1.LedgerHeader.Header.LedgerSeq = xdr.Uint32(cur)
			case 2:
				ledger.V2.LedgerHeader.Header.LedgerSeq = xdr.Uint32(cur)
			default:
				return fmt.Errorf("ledger version %v is not supported", ledger.V)
			}
		}

		if cur == ledgerRange.From() {
			firstLedger = ledger
		} else {
			if err = xdr.MarshalFramed(writer, ledger); err != nil {
				return fmt.Errorf("could not marshal ledger to stream: %w", err)
			}
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
	r.lock.RLock()
	defer r.lock.RUnlock()

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
	r.lock.RLock()
	closeLedgerBackend := false
	defer func() {
		r.lock.RUnlock()
		if closeLedgerBackend {
			r.Close()
		}
	}()

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
	if r.done {
		return xdr.LedgerCloseMeta{}, ErrLoadTestDone
	}
	if sequence > r.latestLedgerSeq {
		closeLedgerBackend = true
		return xdr.LedgerCloseMeta{}, ErrLoadTestDone
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
	r.lock.Lock()
	defer r.lock.Unlock()

	r.done = true
	if r.config.LedgerBackend != nil {
		if err := r.config.LedgerBackend.Close(); err != nil {
			return fmt.Errorf("could not close real ledger backend: %w", err)
		}
	}
	if r.mergedLedgersStream != nil {
		// closing the stream will also close the ledgers file
		if err := r.mergedLedgersStream.Close(); err != nil {
			return fmt.Errorf("could not close merged ledgers xdr stream: %w", err)
		}
		r.mergedLedgersStream = nil
	}
	if r.mergedLedgersFilePath != "" {
		if err := os.Remove(r.mergedLedgersFilePath); err != nil {
			return fmt.Errorf("could not remove merged ledgers file: %w", err)
		}
		r.mergedLedgersFilePath = ""
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
	if err := UpdateLedgerSeqInLedgerEntries(&src, getLedgerSeq); err != nil {
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
