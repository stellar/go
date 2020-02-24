package io

import (
	"testing"

	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
)

func TestStreamReaderError(t *testing.T) {
	tt := assert.New(t)

	mockChangeReader := &MockChangeReader{}
	mockChangeReader.
		On("Read").
		Return(Change{}, errors.New("transient error")).Once()
	mockChangeProcessor := &MockChangeProcessor{}

	err := StreamChanges(mockChangeProcessor, mockChangeReader)
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

	err := StreamChanges(mockChangeProcessor, mockChangeReader)
	tt.EqualError(err, "could not process change: transient error")
}
