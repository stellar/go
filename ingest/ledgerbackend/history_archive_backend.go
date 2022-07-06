package ledgerbackend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
)

const CurrentLedgerFileVersion = 0x01

type HistoryArchiveBackend struct {
	historyarchive.ArchiveBackend
}

func NewHistoryArchiveBackend(b historyarchive.ArchiveBackend) *HistoryArchiveBackend {
	return &HistoryArchiveBackend{
		ArchiveBackend: b,
	}
}

func (b *HistoryArchiveBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	r, err := b.GetFile("latest")
	if os.IsNotExist(err) {
		return 2, nil
	} else if err != nil {
		return 0, errors.Wrap(err, "could not open latest ledger bucket")
	}
	defer r.Close()
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		return 0, errors.Wrap(err, "could not read latest ledger")
	}
	parsed, err := strconv.ParseUint(buf.String(), 10, 32)
	if err != nil {
		return 0, errors.Wrapf(err, "could not parse latest ledger: %q", buf.String())
	}
	return uint32(parsed), nil
}

func (b *HistoryArchiveBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	// Noop
	return nil
}

func (b *HistoryArchiveBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	// Noop
	return true, nil
}

func (b *HistoryArchiveBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	var ledger xdr.LedgerCloseMeta
	r, err := b.GetFile("ledgers/" + strconv.FormatUint(uint64(sequence), 10))
	if err != nil {
		return ledger, err
	}
	defer r.Close()
	var buf bytes.Buffer
	// read and check version header
	var version [1]byte
	l, err := r.Read(version[:])
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}
	if l != 1 || version[0] != CurrentLedgerFileVersion {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("unexpected ledger header version number (0x%x)", version[0])
	}

	if _, err = io.Copy(&buf, r); err != nil {
		return ledger, err
	}
	if err = ledger.UnmarshalBinary(buf.Bytes()); err != nil {
		return ledger, err
	}
	return ledger, nil
}
