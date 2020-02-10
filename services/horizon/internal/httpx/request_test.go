package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/require"
)

// Test that when a client closes its connection by canceling its request that
// the server cancels the context returned by RequestContext.
func TestRequestContext_cancelsContextWhenClientClosesConnection(t *testing.T) {
	canceledInServer := false
	finishedInServer := make(chan struct{})

	handler := func(w http.ResponseWriter, r *http.Request) {
		defer close(finishedInServer)
		t.Log("Server: Request received")
		ctx, _ := RequestContext(r.Context(), w, r)
		t.Log("Server: Sending response")
		w.WriteHeader(200)
		w.Write([]byte("response body"))
		w.(http.Flusher).Flush()
		t.Log("Server: Waiting up to 10s for the context to be canceled before finishing with the request")
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.Canceled {
				canceledInServer = true
				t.Logf("Server: Context done with canceled error 🎉")
			} else if err == nil {
				t.Error("Server: Context done without error, expected to be done with canceled error")
			} else {
				t.Errorf("Server: Context done with error: %#v, expected to be done with canceled error", err)
			}
		case <-time.After(10 * time.Second):
			t.Errorf("Server: 10s timeout reached without context being done")
		}
	}

	mux := chi.NewMux()
	mux.Get("/", handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)
	reqCancel := make(chan struct{})
	req.Cancel = reqCancel
	client := &http.Client{}

	t.Log("Client: Sending request")
	_, err = client.Do(req)
	require.NoError(t, err)
	t.Log("Client: Started receiving response")
	t.Log("Client: Canceling request")
	close(reqCancel)

	t.Log("Test: Waiting up to 10s for server to finish handling request")
	select {
	case <-finishedInServer:
		if canceledInServer {
			t.Log("Test: Success! Server confirmed the context canceled! 🎉")
		} else {
			t.Error("Test: Fail! Server finished without the context being canceled")
		}
	case <-time.After(10 * time.Second):
		t.Error("Test: Fail! 10s timeout reached without acknowledgement server finished")
	}
}
