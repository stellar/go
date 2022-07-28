package metaarchive

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/storage"
	"github.com/stellar/go/xdr"
)

type MetaArchive struct {
	s storage.Storage
}

func NewMetaArchive(b storage.Storage) *MetaArchive {
	return &MetaArchive{
		s: b,
	}
}

func (m *MetaArchive) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	r, err := m.s.GetFile("latest")
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

func (m *MetaArchive) GetLedger(ctx context.Context, sequence uint32) (xdr.SerializedLedgerCloseMeta, error) {
	var ledger xdr.SerializedLedgerCloseMeta
	r, err := m.s.GetFile("ledgers/" + strconv.FormatUint(uint64(sequence), 10))
	if err != nil {
		return xdr.SerializedLedgerCloseMeta{}, err
	}
	defer r.Close()
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		return xdr.SerializedLedgerCloseMeta{}, err
	}
	if err = ledger.UnmarshalBinary(buf.Bytes()); err != nil {
		return xdr.SerializedLedgerCloseMeta{}, err
	}
	return ledger, nil
}
