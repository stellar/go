package expingest

import (
	"bytes"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
)

func TestLoggerStateReader(t *testing.T) {
	mockStateReader := &io.MockStateReader{}
	mockStateReader.
		On("Read").
		Return(io.Change{}, nil).Times(5)
	mockStateReader.
		On("Read").
		Return(io.Change{}, stdio.EOF).Once()
	mockStateReader.
		On("GetSequence").
		Return(uint32(23)).Twice()

	var out bytes.Buffer
	logger := logpkg.New()
	logger.Logger.Out = &out
	done := logger.StartTest(logpkg.InfoLevel)

	loggerStateReader := newLoggerStateReader(
		mockStateReader,
		logger,
		2,
	)

	for {
		_, err := loggerStateReader.Read()
		if err == stdio.EOF {
			break
		}
	}

	logged := done()

	if assert.Len(t, logged, 2) {
		assert.Equal(t, uint32(23), logged[0].Data["ledger"])
		assert.Equal(t, 2, logged[0].Data["numEntries"])
		assert.Equal(t, "Processing entries from History Archive Snapshot", logged[0].Message)

		assert.Equal(t, uint32(23), logged[1].Data["ledger"])
		assert.Equal(t, 4, logged[1].Data["numEntries"])
		assert.Equal(t, "Processing entries from History Archive Snapshot", logged[1].Message)
	}

	mockStateReader.AssertExpectations(t)
}
