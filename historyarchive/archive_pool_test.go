// Copyright 2016 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stellar/go/support/storage"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

	ok, err := archivePool.BucketExists(EmptyXdrArrayHash())
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, userAgent, "uatest")
}
