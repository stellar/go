package ledgerbackend

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/stellar/go/xdr"
)

type ReplayBackend struct {
	ledgerBackend          LedgerBackend
	mergedLedgers          []xdr.LedgerCloseMeta
	index                  int
	generatedLedgers       []xdr.LedgerCloseMeta
	generatedLedgerEntries []xdr.LedgerEntry
	ledgerCloseTime        time.Duration
	startTime              time.Time
	startLedger            uint32
}

type ReplayBackendConfig struct {
	LedgersFilePath       string
	LedgerEntriesFilePath string
	LedgerCloseDuration   time.Duration
}

func NewReplayBackend(config ReplayBackendConfig, ledgerBackend LedgerBackend) (*ReplayBackend, error) {
	var generatedLedgers []xdr.LedgerCloseMeta
	var generatedLedgerEntries []xdr.LedgerEntry

	ledgersFile, err := os.Open(config.LedgersFilePath)
	if err != nil {
		return nil, err
	}
	ledgerEntriesFile, err := os.Open(config.LedgerEntriesFilePath)
	if err != nil {
		return nil, err
	}
	if _, err = xdr.Unmarshal(ledgersFile, &generatedLedgers); err != nil {
		return nil, err
	}
	if _, err = xdr.Unmarshal(ledgerEntriesFile, &generatedLedgerEntries); err != nil {
		return nil, err
	}
	if err = ledgersFile.Close(); err != nil {
		return nil, err
	}
	if err = ledgerEntriesFile.Close(); err != nil {
		return nil, err
	}
	return &ReplayBackend{
		ledgerBackend:          ledgerBackend,
		ledgerCloseTime:        config.LedgerCloseDuration,
		generatedLedgers:       generatedLedgers,
		generatedLedgerEntries: generatedLedgerEntries,
	}, nil
}

func (r *ReplayBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	return r.ledgerBackend.GetLatestLedgerSequence(ctx)
}

func (r *ReplayBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	err := r.ledgerBackend.PrepareRange(ctx, ledgerRange)
	if err != nil {
		return err
	}
	cur := ledgerRange.from
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
	end := ledgerRange.from + uint32(len(r.generatedLedgers))
	if end > ledgerRange.to {
		end = ledgerRange.to
	}
	for cur = cur + 1; cur <= end; cur++ {
		ledger, err = r.ledgerBackend.GetLedger(ctx, cur)
		if err != nil {
			return err
		}
		if err = mergeLedger(&ledger, r.generatedLedgers[0]); err != nil {
			return err
		}
		r.mergedLedgers = append(r.mergedLedgers, ledger)
		r.generatedLedgers = r.generatedLedgers[1:]
	}
	r.startTime = time.Now()
	r.startLedger = ledgerRange.from
	return nil
}

func (r *ReplayBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
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

func mergeLedger(dst *xdr.LedgerCloseMeta, src xdr.LedgerCloseMeta) error {
	if err := validLedger(*dst); err != nil {
		return err
	}
	if err := validLedger(src); err != nil {
		return err
	}

	dst.V1.TxSet.V1TxSet.Phases = append(dst.V1.TxSet.V1TxSet.Phases, src.V1.TxSet.V1TxSet.Phases...)
	dst.V1.TxProcessing = append(dst.V1.TxProcessing, src.V1.TxProcessing...)
	return nil
}
