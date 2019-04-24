package ingestadapters

import (
	"fmt"
	"log"
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

func TestGetState_Sequence(t *testing.T) {
	archive, e := getTestArchive()
	if !assert.NoError(t, e) {
		return
	}
	haa := MakeHistoryArchiveAdapter(archive)

	seq, e := haa.GetLatestLedgerSequence()
	if !assert.NoError(t, e) {
		return
	}

	sr, e := haa.GetState(seq)
	if !assert.NoError(t, e) {
		return
	}
	assert.Equal(t, sr.GetSequence(), seq)
}

func TestGetState_Read(t *testing.T) {
	archive, e := getTestArchive()
	if !assert.NoError(t, e) {
		return
	}
	haa := MakeHistoryArchiveAdapter(archive)

	sr, e := haa.GetState(21686847)
	if !assert.NoError(t, e) {
		return
	}

	ok, le, e := sr.Read()
	if !assert.NoError(t, e) {
		return
	}
	assert.Equal(t, ok, true)

	log.Printf("%v\n", le)
	if !assert.NotNil(t, le) {
		return
	}
	assert.Equal(t, "GAFBQT4VRORLEVEECUYDQGWNVQ563ZN76LGRJR7T7KDL32EES54UOQST", le.Data.Account.AccountId.Address())
}

func getTestArchive() (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		fmt.Sprintf("s3://history.stellar.org/prd/core-live/core_live_001/"),
		historyarchive.ConnectOptions{
			S3Region:         "eu-west-1",
			UnsignedRequests: true,
		},
	)
}
