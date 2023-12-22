package stellarcore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiagnosticEventsToSlice(t *testing.T) {
	events := "AAAAAQAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAgAAAA8AAAAFZXJyb3IAAAAAAAACAAAAAwAAAAUAAAAQAAAAAQAAAAMAAAAOAAAAU3RyYW5zYWN0aW9uIGBzb3JvYmFuRGF0YS5yZXNvdXJjZUZlZWAgaXMgbG93ZXIgdGhhbiB0aGUgYWN0dWFsIFNvcm9iYW4gcmVzb3VyY2UgZmVlAAAAAAUAAAAAAAEJcwAAAAUAAAAAAAG6fA=="
	slice, err := DiagnosticEventsToSlice(events)
	assert.NoError(t, err)
	assert.Len(t, slice, 1)
}
