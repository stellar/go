package adapters

import (
	"context"
	"fmt"
	stdio "io"
	"testing"

	"github.com/stellar/go/support/historyarchive"
	"github.com/stretchr/testify/assert"
)

func TestGetState_Read(t *testing.T) {
	archive, e := getTestArchive()
	if !assert.NoError(t, e) {
		return
	}
	haa := MakeHistoryArchiveAdapter(archive)

	sr, e := haa.GetState(context.Background(), 21686847, 0)
	if !assert.NoError(t, e) {
		return
	}

	lec, e := sr.Read()
	if !assert.NoError(t, e) {
		return
	}
	assert.NotEqual(t, e, stdio.EOF)

	if !assert.NotNil(t, lec) {
		return
	}
	assert.Equal(t, "GAFBQT4VRORLEVEECUYDQGWNVQ563ZN76LGRJR7T7KDL32EES54UOQST", lec.Post.Data.Account.AccountId.Address())
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
