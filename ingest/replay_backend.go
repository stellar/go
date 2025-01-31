package ingest

import (
	"bytes"
	"context"
	"encoding"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
)

type ReplayBackend struct {
	ledgerBackend          ledgerbackend.LedgerBackend
	mergedLedgers          []xdr.LedgerCloseMeta
	index                  int
	generatedLedgers       []xdr.LedgerCloseMeta
	generatedLedgerEntries []xdr.LedgerEntry
	ledgerCloseTime        time.Duration
	startTime              time.Time
	startLedger            uint32
	networkPassphrase      string
}

type ReplayBackendConfig struct {
	NetworkPassphrase     string
	LedgersFilePath       string
	LedgerEntriesFilePath string
	LedgerCloseDuration   time.Duration
}

func unmarshallCompressedXDRFile(path string, dst any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	reader, err := zstd.NewReader(file)
	if err != nil {
		return err
	}
	if _, err = xdr.Unmarshal(reader, dst); err != nil {
		return err
	}
	reader.Close()
	if err = file.Close(); err != nil {
		return err
	}
	return nil
}

func NewReplayBackend(config ReplayBackendConfig, ledgerBackend ledgerbackend.LedgerBackend) (*ReplayBackend, error) {
	var generatedLedgers []xdr.LedgerCloseMeta
	var generatedLedgerEntries []xdr.LedgerEntry

	if err := unmarshallCompressedXDRFile(config.LedgerEntriesFilePath, &generatedLedgerEntries); err != nil {
		return nil, err
	}
	if err := unmarshallCompressedXDRFile(config.LedgersFilePath, &generatedLedgers); err != nil {
		return nil, err
	}
	return &ReplayBackend{
		ledgerBackend:          ledgerBackend,
		ledgerCloseTime:        config.LedgerCloseDuration,
		generatedLedgers:       generatedLedgers,
		generatedLedgerEntries: generatedLedgerEntries,
		networkPassphrase:      config.NetworkPassphrase,
	}, nil
}

func (r *ReplayBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	return r.ledgerBackend.GetLatestLedgerSequence(ctx)
}

func (r *ReplayBackend) PrepareRange(ctx context.Context, ledgerRange ledgerbackend.Range) error {
	err := r.ledgerBackend.PrepareRange(ctx, ledgerRange)
	if err != nil {
		return err
	}
	cur := ledgerRange.From()
	ledger, err := r.ledgerBackend.GetLedger(ctx, cur)
	if err != nil {
		return err
	}
	var changes xdr.LedgerEntryChanges
	for i := 0; i < len(r.generatedLedgerEntries); i++ {
		changes = append(changes, xdr.LedgerEntryChange{
			Type:    xdr.LedgerEntryChangeTypeLedgerEntryCreated,
			Created: &r.generatedLedgerEntries[i],
		})
	}
	var flag xdr.Uint32 = 1
	ledger.V1.UpgradesProcessing = append(ledger.V1.UpgradesProcessing, xdr.UpgradeEntryMeta{
		Upgrade: xdr.LedgerUpgrade{
			Type:     xdr.LedgerUpgradeTypeLedgerUpgradeFlags,
			NewFlags: &flag,
		},
		Changes: changes,
	})
	r.mergedLedgers = append(r.mergedLedgers, ledger)
	end := ledgerRange.From() + uint32(len(r.generatedLedgers))
	if ledgerRange.Bounded() && end > ledgerRange.To() {
		end = ledgerRange.To()
	}
	for cur = cur + 1; cur <= end; cur++ {
		ledger, err = r.ledgerBackend.GetLedger(ctx, cur)
		if err != nil {
			return err
		}
		if err = MergeLedgers(r.networkPassphrase, &ledger, r.generatedLedgers[0]); err != nil {
			return err
		}
		r.mergedLedgers = append(r.mergedLedgers, ledger)
		r.generatedLedgers = r.generatedLedgers[1:]
	}
	r.startTime = time.Now()
	r.startLedger = ledgerRange.From()
	return nil
}

func (r *ReplayBackend) IsPrepared(ctx context.Context, ledgerRange ledgerbackend.Range) (bool, error) {
	return r.ledgerBackend.IsPrepared(ctx, ledgerRange)
}

func (r *ReplayBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	if sequence < r.startLedger {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("sequence number %v out of range", sequence)
	}
	i := int(sequence - r.startLedger)
	if i >= len(r.mergedLedgers) {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("sequence number %v out of range", sequence)
	}
	closeTime := r.startTime.Add(time.Duration(i+1) * r.ledgerCloseTime)
	time.Sleep(time.Until(closeTime))
	return r.mergedLedgers[i], nil
}

func (r *ReplayBackend) Close() error {
	return r.ledgerBackend.Close()
}

func validLedger(ledger xdr.LedgerCloseMeta) error {
	if _, ok := ledger.GetV1(); !ok {
		return fmt.Errorf("ledger version %v is not supported", ledger.V)
	}
	if _, ok := ledger.MustV1().TxSet.GetV1TxSet(); !ok {
		return fmt.Errorf("ledger txset %v is not supported", ledger.MustV1().TxSet.V)
	}
	return nil
}

func extractChanges(networkPassphrase string, changeMap map[string][]Change, ledger xdr.LedgerCloseMeta) error {
	reader, err := NewLedgerChangeReaderFromLedgerCloseMeta(networkPassphrase, ledger)
	if err != nil {
		return err
	}
	for {
		var change Change
		var ledgerKey xdr.LedgerKey
		var b64 string
		change, err = reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		ledgerKey, err = change.LedgerKey()
		if err != nil {
			return err
		}
		b64, err = ledgerKey.MarshalBinaryBase64()
		if err != nil {
			return err
		}
		changeMap[b64] = append(changeMap[b64], change)
	}
	return nil
}

func xdrEquals(a, b encoding.BinaryMarshaler) (bool, error) {
	serialized, err := a.MarshalBinary()
	if err != nil {
		return false, err
	}
	otherSerialized, err := b.MarshalBinary()
	if err != nil {
		return false, err
	}
	return bytes.Equal(serialized, otherSerialized), nil
}

func changeIsEqual(a, b Change) (bool, error) {
	if a.Type != b.Type || a.Reason != b.Reason {
		return false, nil
	}
	if a.Pre == nil {
		if b.Pre != nil {
			return false, nil
		}
	} else {
		if ok, err := xdrEquals(a.Pre, b.Pre); err != nil || !ok {
			return ok, err
		}
	}
	if a.Post == nil {
		if b.Post != nil {
			return false, nil
		}
	} else {
		if ok, err := xdrEquals(a.Post, b.Post); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

func changesAreEqual(a, b map[string][]Change) (bool, error) {
	if len(a) != len(b) {
		return false, nil
	}
	for key, aChanges := range a {
		bChanges := b[key]
		if len(aChanges) != len(bChanges) {
			return false, nil
		}
		for i, aChange := range aChanges {
			bChange := bChanges[i]
			if ok, err := changeIsEqual(aChange, bChange); !ok || err != nil {
				return ok, err
			}
		}
	}
	return true, nil
}

// MergeLedgers merges two xdr.LedgerCloseMeta instances.
func MergeLedgers(networkPassphrase string, dst *xdr.LedgerCloseMeta, src xdr.LedgerCloseMeta) error {
	if err := validLedger(*dst); err != nil {
		return err
	}
	if err := validLedger(src); err != nil {
		return err
	}

	combinedChangesByKey := map[string][]Change{}
	if err := extractChanges(networkPassphrase, combinedChangesByKey, *dst); err != nil {
		return err
	}
	if err := extractChanges(networkPassphrase, combinedChangesByKey, src); err != nil {
		return err
	}

	dst.V1.TxSet.V1TxSet.Phases = append(dst.V1.TxSet.V1TxSet.Phases, src.V1.TxSet.V1TxSet.Phases...)
	dst.V1.TxProcessing = append(dst.V1.TxProcessing, src.V1.TxProcessing...)
	dst.V1.UpgradesProcessing = append(dst.V1.UpgradesProcessing, src.V1.UpgradesProcessing...)
	dst.V1.EvictedTemporaryLedgerKeys = append(dst.V1.EvictedTemporaryLedgerKeys, src.V1.EvictedTemporaryLedgerKeys...)
	dst.V1.EvictedPersistentLedgerEntries = append(dst.V1.EvictedPersistentLedgerEntries, src.V1.EvictedPersistentLedgerEntries...)

	mergedChangesByKey := map[string][]Change{}
	if err := extractChanges(networkPassphrase, mergedChangesByKey, *dst); err != nil {
		return err
	}

	if ok, err := changesAreEqual(combinedChangesByKey, mergedChangesByKey); err != nil {
		return err
	} else if !ok {
		return errors.New("order of changes are not preserved")
	}

	return nil
}
