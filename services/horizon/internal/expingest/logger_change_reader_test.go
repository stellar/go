package expingest

import (
	"errors"
	"testing"

	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func allChanges(changeReader io.ChangeReader) ([]io.Change, error) {
	all := []io.Change{}
	for {
		change, err := changeReader.Read()
		if err != nil {
			return all, err
		}
		all = append(all, change)
	}
}

func createMockReader(changes []io.Change, err error) *io.MockChangeReader {
	mockChangeReader := &io.MockChangeReader{}
	for _, change := range changes {
		mockChangeReader.On("Read").
			Return(change, nil).Once()
	}
	mockChangeReader.On("Read").
		Return(io.Change{}, err).Once()

	return mockChangeReader
}

func TestLoggingChangeReader(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		changes []io.Change
		err     error
	}{
		{
			"empty list with error",
			[]io.Change{},
			errors.New("test error"),
		},
		{
			"empty list with no errors",
			[]io.Change{},
			stdio.EOF,
		},
		{
			"non empty list and error",
			[]io.Change{
				{Type: xdr.LedgerEntryTypeAccount},
				{Type: xdr.LedgerEntryTypeOffer},
			},
			errors.New("test error"),
		},
		{
			"non empty list with no errors",
			[]io.Change{
				{Type: xdr.LedgerEntryTypeOffer},
				{Type: xdr.LedgerEntryTypeAccount},
			},
			stdio.EOF,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			m := createMockReader(testCase.changes, testCase.err)
			reader := newloggingChangeReader(
				m,
				"test",
				2,
				1,
				false,
			)

			all, err := allChanges(reader)
			assert.Equal(t, testCase.changes, all)
			assert.Equal(t, testCase.err, err)
			assert.Equal(t, len(testCase.changes), reader.entryCount)
			assert.Equal(t, uint32(2), reader.sequence)
			assert.Equal(t, 1, reader.frequency)
			m.AssertExpectations(t)
		})
	}
}
