package txsub

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSubmitter(t *testing.T) {
	ctx := NewTestContext()

	t.Run("submits to the configured stellar-core instance correctly", func(t *testing.T) {
		server := NewStaticMockServer(`{
			"status": "PENDING",
			"error": null
			}`)
		defer server.Close()

		s := NewDefaultSubmitter(http.DefaultClient, server.URL)
		sr := s.Submit(ctx, "hello")
		assert.Nil(t, sr.Err)
		assert.True(t, sr.Duration > 0)
		assert.Equal(t, "hello", server.LastRequest.URL.Query().Get("blob"))
	})

	t.Run("succeeds when the stellar-core responds with DUPLICATE status", func(t *testing.T) {
		server := NewStaticMockServer(`{
			"status": "DUPLICATE",
			"error": null
			}`)
		defer server.Close()

		s := NewDefaultSubmitter(http.DefaultClient, server.URL)
		sr := s.Submit(ctx, "hello")
		assert.Nil(t, sr.Err)
	})

	t.Run("errors when the stellar-core url is empty", func(t *testing.T) {
		s := NewDefaultSubmitter(http.DefaultClient, "")
		sr := s.Submit(ctx, "hello")
		assert.NotNil(t, sr.Err)

	})

	t.Run("errors when the stellar-core url is not parseable", func(t *testing.T) {
		s := NewDefaultSubmitter(http.DefaultClient, "http://Not a url")
		sr := s.Submit(ctx, "hello")
		assert.NotNil(t, sr.Err)
	})

	t.Run("errors when the stellar-core url is not reachable", func(t *testing.T) {
		s := NewDefaultSubmitter(http.DefaultClient, "http://127.0.0.1:65535")
		sr := s.Submit(ctx, "hello")
		assert.NotNil(t, sr.Err)
	})

	t.Run("errors when the stellar-core returns an unparseable response", func(t *testing.T) {
		server := NewStaticMockServer(`{`)
		defer server.Close()

		s := NewDefaultSubmitter(http.DefaultClient, server.URL)
		sr := s.Submit(ctx, "hello")
		assert.NotNil(t, sr.Err)
	})

	t.Run("errors when the stellar-core returns an exception response", func(t *testing.T) {
		server := NewStaticMockServer(`{"exception": "Invalid XDR"}`)
		defer server.Close()

		s := NewDefaultSubmitter(http.DefaultClient, server.URL)
		sr := s.Submit(ctx, "hello")
		assert.NotNil(t, sr.Err)
		assert.Contains(t, sr.Err.Error(), "Invalid XDR")
	})

	t.Run("errors when the stellar-core returns an unrecognized status", func(t *testing.T) {
		server := NewStaticMockServer(`{"status": "NOTREAL"}`)
		defer server.Close()

		s := NewDefaultSubmitter(http.DefaultClient, server.URL)
		sr := s.Submit(ctx, "hello")
		assert.NotNil(t, sr.Err)
		assert.Contains(t, sr.Err.Error(), "NOTREAL")
	})

	t.Run("errors when the stellar-core returns an error response", func(t *testing.T) {
		server := NewStaticMockServer(`{"status": "ERROR", "error": "1234"}`)
		defer server.Close()

		s := NewDefaultSubmitter(http.DefaultClient, server.URL)
		sr := s.Submit(ctx, "hello")
		assert.IsType(t, &FailedTransactionError{}, sr.Err)
		ferr := sr.Err.(*FailedTransactionError)
		assert.Equal(t, "1234", ferr.ResultXDR)
	})
}
