package io

import (
	"io"
	"testing"

	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
)

func TestStreamChangesReturnsProcessedChanges(t *testing.T) {
	tt := assert.New(t)

	change := Change{}

	mockChangeReader := &MockChangeReader{}
	mockChangeReader.
		On("Read").
		Return(change, nil).Once()
	mockChangeReader.
		On("Read").
		Return(Change{}, io.EOF).Once()

	mockChangeProcessor := &MockChangeProcessor{}
	mockChangeProcessor.
		On(
			"ProcessChange",
			change,
		).
		Return(nil).Once()

	changes, err := StreamChanges(mockChangeProcessor, mockChangeReader)
	tt.NoError(err)
	tt.Equal(1, changes)
}

func TestStreamReaderError(t *testing.T) {
	tt := assert.New(t)

	mockChangeReader := &MockChangeReader{}
	mockChangeReader.
		On("Read").
		Return(Change{}, errors.New("transient error")).Once()
	mockChangeProcessor := &MockChangeProcessor{}

	_, err := StreamChanges(mockChangeProcessor, mockChangeReader)
	tt.EqualError(err, "could not read transaction: transient error")
}

func TestStreamChangeProcessorError(t *testing.T) {
	tt := assert.New(t)

	change := Change{}
	mockChangeReader := &MockChangeReader{}
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

	_, err := StreamChanges(mockChangeProcessor, mockChangeReader)
	tt.EqualError(err, "could not process change: transient error")
}
