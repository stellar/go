package processors

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/support/errors"
)

func TestStreamReaderError(t *testing.T) {
	tt := assert.New(t)

	mockChangeReader := &ingest.MockChangeReader{}
	mockChangeReader.
		On("Read").
		Return(ingest.Change{}, errors.New("transient error")).Once()
	mockChangeProcessor := &MockChangeProcessor{}

	err := StreamChanges(mockChangeProcessor, mockChangeReader)
	tt.EqualError(err, "could not read transaction: transient error")
}

func TestStreamChangeProcessorError(t *testing.T) {
	tt := assert.New(t)

	change := ingest.Change{}
	mockChangeReader := &ingest.MockChangeReader{}
	mockChangeReader.
		On("Read").
		Return(change, nil).Once()

	mockChangeProcessor := &MockChangeProcessor{}
	mockChangeProcessor.
		On(
			"ProcessChange",
			change,
		).
		Return(errors.New("transient error")).Once()

	err := StreamChanges(mockChangeProcessor, mockChangeReader)
	tt.EqualError(err, "could not process change: transient error")
}
