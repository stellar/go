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
	"time"

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

func TestArchivePoolDoesBackoff(t *testing.T) {
	requestTimes := []time.Time{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.Write([]byte("failure"))
	}))

	pool, err := NewArchivePool([]string{
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "1"),
	}, ArchiveOptions{})
	require.NoError(t, err)

	_, err = pool.GetPathHAS("path")
	require.Error(t, err)

	require.Len(t, pool.(*ArchivePool).errors, 1)
	var statLine *errStats
	for _, stats := range pool.(*ArchivePool).errors {
		statLine = stats
		break
	} // lazy way to get the stats w/o the map key
	assert.NotNil(t, statLine.lastErr)
	assert.Equal(t, 1, statLine.count)
	assert.Equal(t, 1, statLine.backoffs)

	_, err = pool.GetPathHAS("path")
	require.Error(t, err)
	assert.Equal(t, 2, statLine.count)
	assert.Equal(t, 2, statLine.backoffs)

	require.Len(t, requestTimes, 2)
	delay := requestTimes[1].Sub(requestTimes[0])
	require.GreaterOrEqualf(t, delay, 249*time.Millisecond, "wrong backoff delay")

	_, err = pool.GetPathHAS("path")
	require.Error(t, err)
	assert.Equal(t, 3, statLine.count)
	assert.Equal(t, 3, statLine.backoffs)

	require.Len(t, requestTimes, 3)
	delay = requestTimes[2].Sub(requestTimes[1])
	assert.GreaterOrEqualf(t, delay, 499*time.Millisecond, "wrong backoff delay")
}

func TestArchivePoolSkips(t *testing.T) {
	requestTimes := []time.Time{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.Write([]byte("failure"))
	}))

	pool, err := NewArchivePool([]string{
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "1"),
		fmt.Sprintf("%s/%s/%s", server.URL, "fake-archive", "2"),
	}, ArchiveOptions{})
	require.NoError(t, err)

	_, err = pool.GetPathHAS("path")
	require.Error(t, err)
	_, err = pool.BucketExists(EmptyXdrArrayHash())
	require.NoError(t, err)

	require.Len(t, requestTimes, 3) // 2 req + 1 retry
	delay := requestTimes[1].Sub(requestTimes[0])
	require.LessOrEqualf(t, delay, 2*time.Millisecond, "expected no backoff")
}

func TestArchivePoolCycles(t *testing.T) {
	requestTimes := []time.Time{}
	accesses := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		accesses = append(accesses, parts[2])
		requestTimes = append(requestTimes, time.Now())
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

	accesses = []string{}

	//
	// The next access will also try all pools but it should notice a back-off
	// for all of them, too, and *still* stop after a cycle. This ensures 3
	// duped distinct accesses and a *single* backoff (since they all share the
	// same single back-off).
	//

	_, err = pool.GetPathHAS("path")
	require.Error(t, err)

	require.Len(t, accesses, 3)
	assert.Contains(t, accesses, "1")
	assert.Contains(t, accesses, "2")
	assert.Contains(t, accesses, "3")

	require.Len(t, requestTimes, 6)
	require.GreaterOrEqualf(t, time.Since(requestTimes[0]), 250*time.Millisecond, "")
}
