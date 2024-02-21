package exporter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestUploaderSuite(t *testing.T) {
	suite.Run(t, new(UploaderSuite))
}

// UploaderSuite is a test suite for the Uploader.
type UploaderSuite struct {
	suite.Suite
	ctx           context.Context
	mockDataStore MockDataStore
}

func (s *UploaderSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockDataStore = MockDataStore{}
}

func (s *UploaderSuite) TestUpload() {
	key, start, end := "test-1-100", uint32(1), uint32(100)
	archive := NewLedgerMetaArchive(key, start, end)
	for i := start; i <= end; i++ {
		_ = archive.AddLedger(createLedgerCloseMeta(i))
	}

	var capturedWriterTo io.WriterTo
	var capturedKey string
	s.mockDataStore.On("PutFileIfNotExists", key, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedKey = args.Get(0).(string)
			capturedWriterTo = args.Get(1).(io.WriterTo)
		}).Return(nil).Once()

	dataUploader := uploader{destination: &s.mockDataStore}
	assert.NoError(s.T(), dataUploader.Upload(context.Background(), archive))

	var capturedBuf bytes.Buffer
	_, err := capturedWriterTo.WriteTo(&capturedBuf)
	assert.NoError(s.T(), err)

	var decodedArchive LedgerMetaArchive
	decoder := &XDRGzipDecoder{XdrPayload: &decodedArchive.data}
	_, err = decoder.ReadFrom(&capturedBuf)
	assert.NoError(s.T(), err)

	// Assert that the decoded data matches the original test data
	assert.Equal(s.T(), key, capturedKey)
	assert.Equal(s.T(), archive.data, decodedArchive.data)
}

func (s *UploaderSuite) TestUploadPutError() {
	key, start, end := "test-1-100", uint32(1), uint32(100)
	archive := NewLedgerMetaArchive(key, start, end)

	s.mockDataStore.On("PutFileIfNotExists", key,
		mock.Anything).Return(errors.New("error in PutFileIfNotExists"))

	dataUploader := uploader{destination: &s.mockDataStore}
	err := dataUploader.Upload(context.Background(), archive)
	assert.Equal(s.T(), fmt.Sprintf("error uploading %s: error in PutFileIfNotExists", key), err.Error())
}
