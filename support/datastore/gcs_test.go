package datastore

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/require"
)

func TestGCSExists(t *testing.T) {
	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test-bucket",
				Name:       "objects/testnet/file.txt",
			},
			Content: []byte("inside the file"),
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	exists, err := store.Exists(context.Background(), "file.txt")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = store.Exists(context.Background(), "missing-file.txt")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestGCSSize(t *testing.T) {
	content := []byte("inside the file")
	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test-bucket",
				Name:       "objects/testnet/file.txt",
			},
			Content: content,
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	size, err := store.Size(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), size)

	_, err = store.Size(context.Background(), "missing-file.txt")
	require.ErrorIs(t, err, os.ErrNotExist)
}

type writerToRecorder struct {
	io.WriterTo
	total int64
}

func (r *writerToRecorder) WriteTo(w io.Writer) (int64, error) {
	count, err := r.WriterTo.WriteTo(w)
	r.total += count
	return count, err
}

func TestGCSPutFile(t *testing.T) {
	server := fakestorage.NewServer([]fakestorage.Object{})
	defer server.Stop()
	server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{
		Name:                  "test-bucket",
		VersioningEnabled:     false,
		DefaultEventBasedHold: false,
	})

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	content := []byte("inside the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(content),
	}
	err = store.PutFile(context.Background(), "file.txt", writerTo, nil)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), writerTo.total)

	reader, err := store.GetFile(context.Background(), "file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, content)

	metadata, err := store.GetFileMetadata(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)

	otherContent := []byte("other text")
	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(otherContent),
	}
	err = store.PutFile(context.Background(), "file.txt", writerTo, nil)
	require.NoError(t, err)
	require.Equal(t, int64(len(otherContent)), writerTo.total)

	reader, err = store.GetFile(context.Background(), "file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, otherContent)

	metadata, err = store.GetFileMetadata(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)
}

func TestGCSPutFileIfNotExists(t *testing.T) {
	existingContent := []byte("inside the file")
	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test-bucket",
				Name:       "objects/testnet/file.txt",
			},
			Content: existingContent,
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	newContent := []byte("overwrite the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err := store.PutFileIfNotExists(context.Background(), "file.txt", writerTo, nil)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	reader, err := store.GetFile(context.Background(), "file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, existingContent)

	metadata, err := store.GetFileMetadata(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)

	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err = store.PutFileIfNotExists(context.Background(), "other-file.txt", writerTo, nil)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	reader, err = store.GetFile(context.Background(), "other-file.txt")
	require.NoError(t, err)
	requireReaderContentEquals(t, reader, newContent)

	metadata, err = store.GetFileMetadata(context.Background(), "other-file.txt")
	require.NoError(t, err)
	require.Equal(t, map[string]string(nil), metadata)
}

func TestGCSPutFileWithMetadata(t *testing.T) {
	server := fakestorage.NewServer([]fakestorage.Object{})
	defer server.Stop()
	server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{
		Name:                  "test-bucket",
		VersioningEnabled:     false,
		DefaultEventBasedHold: false,
	})

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	// Initial metadata
	metadataObj := map[string]string{
		"start_ledger":            "1234",
		"end_ledger":              "1234",
		"start_ledger_close_time": "1234",
		"end_ledger_close_time":   "1234",
		"network_pass_phrase":     "testnet",
		"compression_type":        "zstd",
		"protocol_version":        "21",
		"core_version":            "v1.2.3",
		"version":                 "1.0.0",
	}

	content := []byte("inside the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(content),
	}
	err = store.PutFile(context.Background(), "file.txt", writerTo, metadataObj)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), writerTo.total)

	metadata, err := store.GetFileMetadata(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, metadataObj, metadata)

	// Update metadata
	modifiedMetadataObj := map[string]string{
		"start_ledger":            "5678",
		"end_ledger":              "6789",
		"start_ledger_close_time": "1622547800",
		"end_ledger_close_time":   "1622548900",
		"network_pass_phrase":     "mainnet",
		"compression_type":        "gzip",
		"protocol_version":        "23",
		"core_version":            "v1.4.0",
		"version":                 "2.0.0",
	}

	otherContent := []byte("other text")
	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(otherContent),
	}
	err = store.PutFile(context.Background(), "file.txt", writerTo, modifiedMetadataObj)
	require.NoError(t, err)
	require.Equal(t, int64(len(otherContent)), writerTo.total)

	metadata, err = store.GetFileMetadata(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, modifiedMetadataObj, metadata)
}

func TestGCSPutFileIfNotExistsWithMetadata(t *testing.T) {
	existingContent := []byte("inside the file")
	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test-bucket",
			},
			Content: existingContent,
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	metadataObj := map[string]string{
		"start_ledger":            "1234",
		"end_ledger":              "1234",
		"start_ledger_close_time": "1234",
		"end_ledger_close_time":   "1234",
		"network_pass_phrase":     "testnet",
		"compression_type":        "zstd",
		"protocol_version":        "21",
		"core_version":            "v1.2.3",
		"version":                 "1.0.0",
	}

	newContent := []byte("overwrite the file")
	writerTo := &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err := store.PutFileIfNotExists(context.Background(), "file.txt", writerTo, metadataObj)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	metadata, err := store.GetFileMetadata(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, metadataObj, metadata)

	modifiedMetadataObj := map[string]string{
		"start_ledger":            "5678",
		"end_ledger":              "6789",
		"start_ledger_close_time": "1622547800",
		"end_ledger_close_time":   "1622548900",
		"network_pass_phrase":     "mainnet",
		"compression_type":        "gzip",
		"protocol_version":        "23",
		"core_version":            "v1.4.0",
		"version":                 "2.0.0",
	}

	writerTo = &writerToRecorder{
		WriterTo: bytes.NewReader(newContent),
	}
	ok, err = store.PutFileIfNotExists(context.Background(), "file.txt", writerTo, modifiedMetadataObj)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, int64(len(newContent)), writerTo.total)

	metadata, err = store.GetFileMetadata(context.Background(), "file.txt")
	require.NoError(t, err)
	require.Equal(t, metadataObj, metadata)
}

func TestGCSGetNonExistentFile(t *testing.T) {
	existingContent := []byte("inside the file")
	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{
				BucketName: "test-bucket",
				Name:       "objects/testnet/file.txt",
			},
			Content: existingContent,
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	_, err = store.GetFile(context.Background(), "other-file.txt")
	require.ErrorIs(t, err, os.ErrNotExist)

	metadata, err := store.GetFileMetadata(context.Background(), "other-file.txt")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Equal(t, map[string]string(nil), metadata)
}

func TestGCSGetFileValidatesCRC32C(t *testing.T) {
	// test on a gzipped file so we can verify ReadCompressed()
	// was called correctly
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Name = "file.gz"
	if _, err := zw.Write([]byte("gzipped object data")); err != nil {
		t.Fatalf("creating gzip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}

	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{
				// set a CRC32C value which doesn't actually match the file contents
				Crc32c:          "mw/l0Q==",
				BucketName:      "test-bucket",
				Name:            "objects/testnet/file.gz",
				ContentEncoding: "gzip",
				ContentType:     "text/plain",
			},
			Content: buf.Bytes(),
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	reader, err := store.GetFile(context.Background(), "file.gz")
	require.NoError(t, err)
	buf.Reset()
	_, err = io.Copy(&buf, reader)
	require.EqualError(t, err, "storage: bad CRC on read: got 985946173, want 2601510353")
}

func TestGCSListFilePaths(t *testing.T) {
	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{BucketName: "test-bucket", Name: "objects/testnet/a"},
			Content:     []byte("1"),
		},
		{
			ObjectAttrs: fakestorage.ObjectAttrs{BucketName: "test-bucket", Name: "objects/testnet/b"},
			Content:     []byte("1"),
		},
		{
			ObjectAttrs: fakestorage.ObjectAttrs{BucketName: "test-bucket", Name: "objects/testnet/c"},
			Content:     []byte("1"),
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	paths, err := store.ListFilePaths(context.Background(), "", 2)
	require.NoError(t, err)

	require.Equal(t, []string{"objects/testnet/a", "objects/testnet/b"}, paths)
}

func TestGCSListFilePaths_WithPrefix(t *testing.T) {
	server := fakestorage.NewServer([]fakestorage.Object{
		{
			ObjectAttrs: fakestorage.ObjectAttrs{BucketName: "test-bucket", Name: "objects/testnet/a/x"},
			Content:     []byte("1"),
		},
		{
			ObjectAttrs: fakestorage.ObjectAttrs{BucketName: "test-bucket", Name: "objects/testnet/a/y"},
			Content:     []byte("1"),
		},
		{
			ObjectAttrs: fakestorage.ObjectAttrs{BucketName: "test-bucket", Name: "objects/testnet/b/z"},
			Content:     []byte("1"),
		},
	})
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	paths, err := store.ListFilePaths(context.Background(), "a", 10)
	require.NoError(t, err)
	require.Equal(t, []string{"objects/testnet/a/x", "objects/testnet/a/y"}, paths)
}

func TestGCSListFilePaths_LimitDefaultAndCap(t *testing.T) {
	objects := make([]fakestorage.Object, 0, 1200)
	for i := 0; i < 1200; i++ {
		objects = append(objects, fakestorage.Object{
			ObjectAttrs: fakestorage.ObjectAttrs{BucketName: "test-bucket", Name: fmt.Sprintf("objects/testnet/%04d", i)},
			Content:     []byte("1"),
		})
	}
	server := fakestorage.NewServer(objects)
	defer server.Stop()

	store, err := FromGCSClient(context.Background(), server.Client(), "test-bucket/objects/testnet")
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	paths, err := store.ListFilePaths(context.Background(), "", 0)
	require.NoError(t, err)
	require.Equal(t, 1000, len(paths))

	paths, err = store.ListFilePaths(context.Background(), "", 5000)
	require.NoError(t, err)
	require.Equal(t, 1000, len(paths))
}
