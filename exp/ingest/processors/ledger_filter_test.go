package processors

import (
	"context"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
)

// ledgerWriterRecorder records all transactions which have been passed into Write()
// MockLedgerWriter cannot emulate ledgerWriterRecorder because testify's mock
// implementation doesn't allow you to assert the order in which a function was called
type ledgerWriterRecorder struct {
	written []io.LedgerTransaction
}

func (m *ledgerWriterRecorder) Write(transaction io.LedgerTransaction) error {
	m.written = append(m.written, transaction)
	return nil
}

func (m *ledgerWriterRecorder) Close() error {
	return nil
}

func TestLedgerFilter(t *testing.T) {

	filter := &LedgerFilter{IgnoreLedgersBefore: 123}

	for _, sequence := range []uint32{120, 121, 122} {
		reader := &io.MockLedgerReader{}
		reader.On("GetSequence").Return(sequence)
		reader.On("Close").Return(nil).Once()
		reader.On("Read").Return(io.LedgerTransaction{Index: sequence}, nil).Once()
		reader.On("Read").Return(io.LedgerTransaction{Index: sequence + 1}, nil).Once()
		reader.On("Read").Return(io.LedgerTransaction{Index: sequence + 2}, nil).Once()
		reader.On("Read").Return(io.LedgerTransaction{}, stdio.EOF).Once()
		writer := &io.MockLedgerWriter{}
		writer.On("Close").Return(nil).Once()

		if err := filter.ProcessLedger(context.Background(), nil, reader, writer); err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		reader.AssertExpectations(t)
		writer.AssertExpectations(t)
	}

	for _, sequence := range []uint32{123, 124} {
		buffer := []io.LedgerTransaction{
			io.LedgerTransaction{Index: sequence},
			io.LedgerTransaction{Index: sequence + 1},
			io.LedgerTransaction{Index: sequence + 2},
		}
		reader := &io.MockLedgerReader{}
		reader.On("GetSequence").Return(sequence)
		reader.On("Close").Return(nil).Once()
		for i := range buffer {
			reader.On("Read").Return(buffer[i], nil).Once()
		}
		reader.On("Read").Return(io.LedgerTransaction{}, stdio.EOF).Once()
		writer := &ledgerWriterRecorder{}

		if err := filter.ProcessLedger(context.Background(), nil, reader, writer); err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if len(writer.written) != len(buffer) {
			t.Fatal("expected written to match read buffer")
		}
		for i := range buffer {
			if buffer[i].Index != writer.written[i].Index {
				t.Fatal("expected written to match read buffer")
			}
		}
	}
}
