package datastore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

// mockS3Object stores object data and metadata within the mock server.
type mockS3Object struct {
	body     []byte
	metadata map[string]string
	crc32c   string
}

// mockS3Server is our mock S3 server, holding an in-memory "bucket".
type mockS3Server struct {
	mu      sync.Mutex
	objects map[string]mockS3Object
}

// ServeHTTP is the core logic of the mock server, handling all incoming HTTP requests.
func (s *mockS3Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pathParts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)

	switch r.Method {
	case http.MethodHead:
		// See https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadBucket.html
		// See https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html
		s.handleHeadRequest(w, pathParts)
	case http.MethodGet:
		// See https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html
		s.handleGetRequest(w, pathParts)
	case http.MethodPut:
		// See https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html
		s.handlePutRequest(w, r, pathParts)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *mockS3Server) handleHeadRequest(w http.ResponseWriter, pathParts []string) {
	// Handle HeadBucket: A request with no key part in the path.
	// We assume the bucket always exists in the mock.
	if len(pathParts) < 2 || pathParts[1] == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle HeadObject: A request with a key.
	key := pathParts[1]
	obj, exists := s.objects[key]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	s.setObjectHeaders(w, obj)
	w.WriteHeader(http.StatusOK)
}

func (s *mockS3Server) handleGetRequest(w http.ResponseWriter, pathParts []string) {
	if len(pathParts) < 2 {
		http.Error(w, "Invalid path: Key is required for GET", http.StatusBadRequest)
		return
	}

	key := pathParts[1]
	obj, exists := s.objects[key]
	if !exists {
		s.writeS3NoSuchKeyError(w)
		return
	}

	s.setObjectHeaders(w, obj)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(obj.body)
}

func (s *mockS3Server) handlePutRequest(w http.ResponseWriter, r *http.Request, pathParts []string) {
	if len(pathParts) < 2 {
		http.Error(w, "Invalid path: Key is required for PUT", http.StatusBadRequest)
		return
	}

	key := pathParts[1]

	// Handle conditional put
	if r.Header.Get("If-None-Match") == "*" {
		if _, exists := s.objects[key]; exists {
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metadata := s.extractMetadata(r.Header)
	crc32c := r.Header.Get("x-amz-checksum-crc32c")
	s.objects[key] = mockS3Object{body: body, metadata: metadata, crc32c: crc32c}
	w.WriteHeader(http.StatusOK)
}

func (s *mockS3Server) setObjectHeaders(w http.ResponseWriter, obj mockS3Object) {
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.body)))
	for k, v := range obj.metadata {
		w.Header().Set("x-amz-meta-"+k, v)
	}
	if obj.crc32c != "" {
		w.Header().Set("x-amz-checksum-crc32c", obj.crc32c)
	}
}

func (s *mockS3Server) writeS3NoSuchKeyError(w http.ResponseWriter) {
	const s3NoSuchKeyError = `<Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(s3NoSuchKeyError))
}

func (s *mockS3Server) extractMetadata(headers http.Header) map[string]string {
	metadata := make(map[string]string)
	for k, v := range headers {
		if strings.HasPrefix(strings.ToLower(k), "x-amz-meta-") {
			metaKey := strings.TrimPrefix(strings.ToLower(k), "x-amz-meta-")
			metadata[metaKey] = v[0]
		}
	}
	return metadata
}

// setupTestS3DataStore is a helper function that initializes the mock server and an S3DataStore instance.
func setupTestS3DataStore(t *testing.T, ctx context.Context, bucketPath string, initObjects map[string]mockS3Object) (DataStore, func()) {
	t.Helper()
	mockServer := &mockS3Server{
		objects: make(map[string]mockS3Object),
	}
	// Initialize the mock server with provided objects.
	for key, obj := range initObjects {
		mockServer.objects[key] = obj
	}
	server := httptest.NewServer(mockServer)

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-west-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("KEY", "SECRET", "")),
	)
	require.NoError(t, err)

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(server.URL)
		o.UsePathStyle = true
	})

	store, err := FromS3Client(ctx, client, bucketPath, DataStoreSchema{})
	require.NoError(t, err)

	teardown := func() {
		server.Close()
	}

	return store, teardown
}

func TestS3Exists(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{})
	defer teardown()

	content := []byte("inside the file")
	err := store.PutFile(ctx, "file.txt", bytes.NewReader(content), nil)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "file.txt")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = store.Exists(ctx, "missing-file.txt")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestS3Size(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{})
	defer teardown()

	content := []byte("inside the file")
	err := store.PutFile(ctx, "file.txt", bytes.NewReader(content), nil)
	require.NoError(t, err)

	size, err := store.Size(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), size)

	_, err = store.Size(ctx, "missing-file.txt")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestS3PutFile(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{})
	defer teardown()

	content := []byte("inside the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(content),
	}
	err := store.PutFile(ctx, "file.txt", writerTo, nil)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), writerTo.total)

	reader, err := store.GetFile(ctx, "file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, content)

	metadata, err := store.GetFileMetadata(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)

	otherContent := []byte("other text")
	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(otherContent),
	}
	err = store.PutFile(ctx, "file.txt", writerTo, nil)
	require.NoError(t, err)
	require.Equal(t, int64(len(otherContent)), writerTo.total)

	reader, err = store.GetFile(ctx, "file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, otherContent)

	metadata, err = store.GetFileMetadata(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)
}

func TestS3PutFileIfNotExists(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{})
	defer teardown()

	existingContent := []byte("inside the file")
	err := store.PutFile(ctx, "file.txt", bytes.NewReader(existingContent), nil)
	require.NoError(t, err)

	newContent := []byte("overwrite the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err := store.PutFileIfNotExists(ctx, "file.txt", writerTo, nil)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	reader, err := store.GetFile(ctx, "file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, existingContent)

	metadata, err := store.GetFileMetadata(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)

	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err = store.PutFileIfNotExists(ctx, "other-file.txt", writerTo, nil)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	reader, err = store.GetFile(ctx, "other-file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, newContent)

	metadata, err = store.GetFileMetadata(ctx, "other-file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)
}

func TestS3PutFileWithMetadata(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{})
	defer teardown()

	metadataObj := MetaData{
		StartLedger:          1234,
		EndLedger:            1234,
		StartLedgerCloseTime: 1234,
		EndLedgerCloseTime:   1234,
		NetworkPassPhrase:    "testnet",
		CompressionType:      "zstd",
		ProtocolVersion:      21,
		CoreVersion:          "v1.2.3",
		Version:              "1.0.0",
	}

	content := []byte("inside the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(content),
	}
	err := store.PutFile(ctx, "file.txt", writerTo, metadataObj.ToMap())
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), writerTo.total)

	metadata, err := store.GetFileMetadata(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, metadataObj.ToMap(), metadata)

	modifiedMetadataObj := MetaData{
		StartLedger:          5678,
		EndLedger:            6789,
		StartLedgerCloseTime: 1622547800,
		EndLedgerCloseTime:   1622548900,
		NetworkPassPhrase:    "mainnet",
		CompressionType:      "gzip",
		ProtocolVersion:      23,
		CoreVersion:          "v1.4.0",
		Version:              "2.0.0",
	}

	otherContent := []byte("other text")
	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(otherContent),
	}
	err = store.PutFile(ctx, "file.txt", writerTo, modifiedMetadataObj.ToMap())
	require.NoError(t, err)
	require.Equal(t, int64(len(otherContent)), writerTo.total)

	metadata, err = store.GetFileMetadata(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, modifiedMetadataObj.ToMap(), metadata)
}

func TestS3PutFileIfNotExistsWithMetadata(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{})
	defer teardown()

	metadataObj := MetaData{
		StartLedger:          1234,
		EndLedger:            1234,
		StartLedgerCloseTime: 1234,
		EndLedgerCloseTime:   1234,
		NetworkPassPhrase:    "testnet",
		CompressionType:      "zstd",
		ProtocolVersion:      21,
		CoreVersion:          "v1.2.3",
		Version:              "1.0.0",
	}

	newContent := []byte("overwrite the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err := store.PutFileIfNotExists(ctx, "file.txt", writerTo, metadataObj.ToMap())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	metadata, err := store.GetFileMetadata(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, metadataObj.ToMap(), metadata)

	modifiedMetadataObj := MetaData{
		StartLedger:          5678,
		EndLedger:            6789,
		StartLedgerCloseTime: 1622547800,
		EndLedgerCloseTime:   1622548900,
		NetworkPassPhrase:    "mainnet",
		CompressionType:      "gzip",
		ProtocolVersion:      23,
		CoreVersion:          "v1.4.0",
		Version:              "2.0.0",
	}

	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err = store.PutFileIfNotExists(ctx, "file.txt", writerTo, modifiedMetadataObj.ToMap())
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	metadata, err = store.GetFileMetadata(ctx, "file.txt")
	require.NoError(t, err)
	require.Equal(t, metadataObj.ToMap(), metadata)
}

func TestS3GetNonExistentFile(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{})
	defer teardown()

	_, err := store.GetFile(ctx, "other-file.txt")
	require.ErrorIs(t, err, os.ErrNotExist)

	metadata, err := store.GetFileMetadata(ctx, "other-file.txt")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Equal(t, map[string]string(nil), metadata)
}

func TestS3GetFileValidatesCRC32C(t *testing.T) {
	ctx := context.Background()
	store, teardown := setupTestS3DataStore(t, ctx, "test-bucket/objects/testnet", map[string]mockS3Object{
		"objects/testnet/file.txt": {
			body:     []byte("hello"),
			metadata: map[string]string{},
			crc32c:   "VLn+tw==", // invalid CRC32C for the content
		}})
	defer teardown()

	reader, err := store.GetFile(ctx, "file.txt")
	require.NoError(t, err)
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	require.EqualError(t, err, "checksum did not match: algorithm CRC32C, expect VLn+tw==, actual mnG7TA==")
}
