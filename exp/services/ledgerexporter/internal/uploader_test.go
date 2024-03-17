package ledgerexporter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestUploaderSuite(t *testing.T) {
	suite.Run(t, new(UploaderSuite))
}

// UploaderSuite is a test suite for the Uploader.
type UploaderSuite struct {
	suite.Suite
	ctx           context.Context
	mockDataStore datastore.MockDataStore
}

func (s *UploaderSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockDataStore = datastore.MockDataStore{}
}

func (s *UploaderSuite) TestUpload() {
	key, start, end := "test-1-100", uint32(1), uint32(100)
	archive := NewLedgerMetaArchive(key, start, end)
	for i := start; i <= end; i++ {
		_ = archive.AddLedger(createLedgerCloseMeta(i))
	}

	var capturedWriterTo io.WriterTo
	var capturedKey string
	s.mockDataStore.On("PutFileIfNotExists", mock.Anything, key, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedKey = args.Get(1).(string)
			capturedWriterTo = args.Get(2).(io.WriterTo)
		}).Return(nil).Once()

	dataUploader := uploader{dataStore: &s.mockDataStore}
	require.NoError(s.T(), dataUploader.Upload(context.Background(), archive))

	var capturedBuf bytes.Buffer
	_, err := capturedWriterTo.WriteTo(&capturedBuf)
	require.NoError(s.T(), err)

	var decodedArchive LedgerMetaArchive
	decoder := &XDRGzipDecoder{XdrPayload: &decodedArchive.data}
	_, err = decoder.ReadFrom(&capturedBuf)
	require.NoError(s.T(), err)

	// require that the decoded data matches the original test data
	require.Equal(s.T(), key, capturedKey)
	require.Equal(s.T(), archive.data, decodedArchive.data)
}

func (s *UploaderSuite) TestUploadPutError() {
	key, start, end := "test-1-100", uint32(1), uint32(100)
	archive := NewLedgerMetaArchive(key, start, end)

	s.mockDataStore.On("PutFileIfNotExists", context.Background(), key,
		mock.Anything).Return(errors.New("error in PutFileIfNotExists"))

	dataUploader := uploader{dataStore: &s.mockDataStore}
	err := dataUploader.Upload(context.Background(), archive)
	require.Equal(s.T(), fmt.Sprintf("error uploading %s: error in PutFileIfNotExists", key), err.Error())
}

func (s *UploaderSuite) TestRunChannelClose() {
	s.mockDataStore.On("PutFileIfNotExists", mock.Anything,
		mock.Anything, mock.Anything).Return(nil)

	objectCh := make(chan *LedgerMetaArchive, 1)
	go func() {
		key, start, end := "test", uint32(1), uint32(100)
		for i := start; i <= end; i++ {
			objectCh <- NewLedgerMetaArchive(key, i, i)
		}
		<-time.After(time.Second * 2)
		close(objectCh)
	}()

	dataUploader := uploader{dataStore: &s.mockDataStore, metaArchiveCh: objectCh}
	require.NoError(s.T(), dataUploader.Run(context.Background()))
}

func (s *UploaderSuite) TestRunContextCancel() {
	objectCh := make(chan *LedgerMetaArchive, 1)
	s.mockDataStore.On("PutFileIfNotExists", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			objectCh <- NewLedgerMetaArchive("test", 1, 1)
		}
	}()

	go func() {
		<-time.After(time.Second * 2)
		cancel()
	}()

	dataUploader := uploader{dataStore: &s.mockDataStore, metaArchiveCh: objectCh}
	err := dataUploader.Run(ctx)

	require.EqualError(s.T(), err, "context canceled")
}

func (s *UploaderSuite) TestRunUploadError() {
	objectCh := make(chan *LedgerMetaArchive, 10)
	objectCh <- NewLedgerMetaArchive("test", 1, 1)

	s.mockDataStore.On("PutFileIfNotExists", mock.Anything, "test",
		mock.Anything).Return(errors.New("Put error"))

	dataUploader := uploader{dataStore: &s.mockDataStore, metaArchiveCh: objectCh}
	err := dataUploader.Run(context.Background())
	require.Equal(s.T(), "error uploading test: Put error", err.Error())
}
