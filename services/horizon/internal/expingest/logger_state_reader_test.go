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
	mockChangeReader := &io.MockChangeReader{}
	mockChangeReader.
		On("Read").
		Return(io.Change{}, nil).Times(5)
	mockChangeReader.
		On("Read").
		Return(io.Change{}, stdio.EOF).Once()

	var out bytes.Buffer
	logger := logpkg.New()
	logger.Logger.Out = &out
	done := logger.StartTest(logpkg.InfoLevel)

	loggerStateReader := newLoggerStateReader(
		mockChangeReader,
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
		assert.Equal(t, "Entries processed from HAS: 2", logged[0].Message)
		assert.Equal(t, "Entries processed from HAS: 4", logged[1].Message)
	}

	mockChangeReader.AssertExpectations(t)
}
