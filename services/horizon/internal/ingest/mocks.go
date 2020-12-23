package ingest

import (
	"context"

	"github.com/stellar/go/ingest/io"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type mockHistoryArchiveAdapter struct {
	mock.Mock
}

func (m *mockHistoryArchiveAdapter) GetLatestLedgerSequence() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *mockHistoryArchiveAdapter) BucketListHash(sequence uint32) (xdr.Hash, error) {
	args := m.Called(sequence)
	return args.Get(0).(xdr.Hash), args.Error(1)
}

func (m *mockHistoryArchiveAdapter) GetState(ctx context.Context, sequence uint32) (io.ChangeReader, error) {
	args := m.Called(ctx, sequence)
	return args.Get(0).(io.ChangeReader), args.Error(1)
}
