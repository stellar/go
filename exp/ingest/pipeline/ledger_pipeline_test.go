package pipeline_test

import (
	"context"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/processors"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestUpgradeChangesPassed(t *testing.T) {
	mockLedgerReader := &io.MockLedgerReader{}

	account := xdr.AccountEntry{
		AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Thresholds: [4]byte{1, 1, 1, 1},
	}

	mockLedgerReader.On("GetSequence").Return(uint32(1000))
	mockLedgerReader.On("GetHeader").Return(xdr.LedgerHeaderHistoryEntry{})
	mockLedgerReader.On("Read").Return(io.LedgerTransaction{}, stdio.EOF)
	mockLedgerReader.On("GetUpgradeChanges").Return([]io.Change{
		io.Change{
			Type: xdr.LedgerEntryTypeAccount,
			Pre:  nil,
			Post: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &account,
				},
				LastModifiedLedgerSeq: 1000,
			},
		},
	})
	mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	mockLedgerReader.On("Close").Return(nil).Once()
	mockLedgerReader.On("SuccessfulLedgerOperationCount").Return(1)
	mockLedgerReader.On("FailedTransactionCount").Return(0)
	mockLedgerReader.On("SuccessfulLedgerOperationCount").Return(1)
	mockLedgerReader.On("CloseTime").Return(int64(234))

	// Ensure upgrade changes are available to processors to read
	ledgerPipeline := &pipeline.LedgerPipeline{}
	ledgerPipeline.SetRoot(
		pipeline.LedgerNode(&processors.RootProcessor{}).
			Pipe(
				pipeline.LedgerNode(&testLedgerProcessor{t}).
					Pipe(
						pipeline.LedgerNode(&testLedgerProcessor{t}),
					),
				pipeline.LedgerNode(&testLedgerProcessor{t}).
					Pipe(
						pipeline.LedgerNode(&testLedgerProcessor{t}),
					),
			),
	)

	err := <-ledgerPipeline.Process(mockLedgerReader)
	assert.NoError(t, err)
}

type testLedgerProcessor struct {
	t *testing.T
}

func (p *testLedgerProcessor) ProcessLedger(ctx context.Context, store *supportPipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer func() {
		// io.LedgerReader.Close() returns error if upgrade changes have not
		// been processed so it's worth checking the error.
		closeErr := r.Close()
		// Do not overwrite the previous error
		if err == nil {
			err = closeErr
		}
	}()
	defer w.Close()

	_, err = r.Read()
	assert.Error(p.t, err)
	assert.Equal(p.t, stdio.EOF, err)

	change, err := r.ReadUpgradeChange()
	assert.NoError(p.t, err)
	assert.Equal(p.t, change.Type, xdr.LedgerEntryTypeAccount)

	_, err = r.ReadUpgradeChange()
	assert.Error(p.t, err)
	assert.Equal(p.t, stdio.EOF, err)

	return nil
}

func (*testLedgerProcessor) Name() string {
	return "Test processor"
}

func (*testLedgerProcessor) Reset() {
	//
}
