package ledgerbackend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func MockGCSBackend() GCSBackend {
	return GCSBackend{
		fileSuffix: ".xdr.gz",
	}
}

func TestGetLatestFileNameLedgerSequence(t *testing.T) {
	csb := MockGCSBackend()
	directory := "ledgers/pubnet/21-30"
	filenames := []string{
		"ledgers/pubnet/21-30/21.xdr.gz",
		"ledgers/pubnet/21-30/22.xdr.gz",
		"ledgers/pubnet/21-30/23.xdr.gz",
	}
	latestLedgerSequence, _ := csb.GetLatestFileNameLedgerSequence(filenames, directory)

	assert.Equal(t, uint32(23), latestLedgerSequence)
}

func TestGetLatestDirectory(t *testing.T) {
	csb := MockGCSBackend()
	directories := []string{"ledgers/pubnet/1-10", "ledgers/pubnet/11-20", "ledgers/pubnet/21-30"}
	latestDirectory, _ := csb.GetLatestDirectory(directories)

	assert.Equal(t, "ledgers/pubnet/21-30", latestDirectory)
}
