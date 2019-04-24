package ingestadapters

import (
	"fmt"
	"testing"

	"github.com/stellar/go/support/historyarchive"

	"github.com/stretchr/testify/assert"
)

func TestGetLatestLedgerSequence(t *testing.T) {
	archive, e := getTestArchive()
	if !assert.NoError(t, e) {
		return
	}

	haa := MakeHistoryArchiveAdapter(archive)
	seq, e := haa.GetLatestLedgerSequence()
	if !assert.NoError(t, e) {
		return
	}
	assert.Equal(t, uint32(931455), seq)
}

func getTestArchive() (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		fmt.Sprintf("s3://history.stellar.org/prd/core-testnet/core_testnet_001/"),
		historyarchive.ConnectOptions{
			S3Region:         "eu-west-1",
			UnsignedRequests: true,
		},
	)
}
