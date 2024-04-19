package ledgerbackend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLatestFileNameLedgerSequence(t *testing.T) {
	directory := "ledgers/pubnet/21-30"
	filenames := []string{
		"ledgers/pubnet/21-30/21.xdr.gz",
		"ledgers/pubnet/21-30/22.xdr.gz",
		"ledgers/pubnet/21-30/23.xdr.gz",
	}
	latestLedgerSequence, _ := getLatestFileNameLedgerSequence(filenames, directory)

	assert.Equal(t, uint32(23), latestLedgerSequence)
}

func TestGetLatestDirectory(t *testing.T) {
	directories := []string{"ledgers/pubnet/1-10", "ledgers/pubnet/11-20", "ledgers/pubnet/21-30"}
	latestDirectory, _ := getLatestDirectory(directories)

	assert.Equal(t, "ledgers/pubnet/21-30", latestDirectory)
}
