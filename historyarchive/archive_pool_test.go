// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stellar/go/support/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfiguresHttpUserAgentForArchivePool(t *testing.T) {
	var userAgent string
	var archiveURLs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.Header["User-Agent"][0]
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	archiveURLs = append(archiveURLs, server.URL)

	archiveOptions := ArchiveOptions{
		ConnectOptions: storage.ConnectOptions{
			UserAgent: "uatest",
		},
	}

	archivePool, err := NewArchivePool(archiveURLs, archiveOptions)
	require.NoError(t, err)

	ok, err := archivePool.BucketExists(EmptyXdrArrayHash())
	require.True(t, ok)
	require.NoError(t, err)
	require.Equal(t, userAgent, "uatest")
}

func TestArchivePoolRoundRobin(t *testing.T) {
	accesses := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		accesses = append(accesses, parts[2])
		w.Write([]byte("boo"))
	}))

	pool, err := NewArchivePool([]string{
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "1"),
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "2"),
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "3"),
	}, ArchiveOptions{})
	require.NoError(t, err)

	_, err = pool.BucketExists(EmptyXdrArrayHash())
	require.NoError(t, err)
	_, err = pool.BucketExists(EmptyXdrArrayHash())
	require.NoError(t, err)
	_, err = pool.BucketExists(EmptyXdrArrayHash())
	require.NoError(t, err)
	_, err = pool.BucketExists(EmptyXdrArrayHash())
	require.NoError(t, err)

	assert.Contains(t, accesses, "1")
	assert.Contains(t, accesses, "2")
	assert.Contains(t, accesses, "3")
	assert.Len(t, accesses, 4)
}

func TestArchivePoolCycles(t *testing.T) {
	accesses := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		accesses = append(accesses, parts[2])
		w.Write([]byte("failure"))
	}))

	pool, err := NewArchivePool([]string{
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "1"),
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "2"),
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "3"),
	}, ArchiveOptions{})
	require.NoError(t, err)

	//
	// A single access should try all pools and stop after a cycle is
	// encountered, so we ensure here that there are 3 distinct accesses.
	//
	_, err = pool.GetPathHAS("path")
	require.Error(t, err)

	require.Len(t, accesses, 3)
	assert.Contains(t, accesses, "1")
	assert.Contains(t, accesses, "2")
	assert.Contains(t, accesses, "3")

	//
	// The next access will also try all pools but it should notice a back-off
	// for all of them, too, and *still* stop after a cycle. This ensures 3
	// duped distinct accesses and a *single* backoff (since they all share the
	// same single back-off).
	//
	accesses = []string{}
	_, err = pool.GetPathHAS("path")
	require.Error(t, err)

	require.Len(t, accesses, 3)
	assert.Contains(t, accesses, "1")
	assert.Contains(t, accesses, "2")
	assert.Contains(t, accesses, "3")
}
