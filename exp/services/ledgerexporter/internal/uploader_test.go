package exporter

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io"
	"testing"
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

	buffer, err := archive.GetBinaryData()
	assert.NoError(s.T(), err)

	compressedBuf, err := Compress(buffer)
	assert.NoError(s.T(), err)

	var capturedData io.Reader
	var capturedKey string
	s.mockDataStore.On("PutFileIfNotExists", key,
		io.NopCloser(bytes.NewReader(compressedBuf))).
		Run(func(args mock.Arguments) {
			capturedKey = args.Get(0).(string)
			capturedData = args.Get(1).(io.Reader)
		}).Return(nil).Once()

	dataUploader := uploader{destination: &s.mockDataStore}
	assert.NoError(s.T(), dataUploader.Upload(archive))

	var buf bytes.Buffer
	_, err = io.Copy(&buf, capturedData)

	var actualArchive LedgerMetaArchive
	decompressedData, err := Decompress(buf.Bytes())
	assert.NoError(s.T(), err)

	assert.NoError(s.T(), actualArchive.SetBinaryData(decompressedData))

	// Assert that the decoded data matches the original data
	assert.Equal(s.T(), key, capturedKey)
	assert.Equal(s.T(), archive.data, actualArchive.data)
}

func (s *UploaderSuite) TestUploadPutError() {
	key, start, end := "test-1-100", uint32(1), uint32(100)
	archive := NewLedgerMetaArchive(key, start, end)

	buf, err := archive.GetBinaryData()
	assert.NoError(s.T(), err)

	compressedBuf, err := Compress(buf)
	assert.NoError(s.T(), err)

	s.mockDataStore.On("PutFileIfNotExists", key,
		io.NopCloser(bytes.NewReader(compressedBuf))).Return(errors.New("error in PutFileIfNotExists"))

	dataUploader := uploader{destination: &s.mockDataStore}
	err = dataUploader.Upload(archive)
	assert.Equal(s.T(), fmt.Sprintf("error uploading %s: error in PutFileIfNotExists", key), err.Error())
}
