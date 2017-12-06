package stellarcore

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/stellar/go/support/errors"
)

// Client represents a client that is capable of communicating with a
// stellar-core server using HTTP
type Client struct {
	// HTTP is the client to use when communicating with stellar-core.  If nil,
	// http.DefaultClient will be used.
	HTTP HTTP

	// URL of Stellar Core server to connect.
	URL string
}

// Info returns
func (c *Client) Info(ctx context.Context) (resp *InfoResponse, err error) {
	var hresp *http.Response

	req, err := http.NewRequest(http.MethodGet, c.URL+"/info", nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create request")
		return
	}

	req = req.WithContext(ctx)

	hresp, err = c.http().Do(req)
	if err != nil {
		err = errors.Wrap(err, "http request errored")
		return
	}
	defer hresp.Body.Close()

	if !(hresp.StatusCode >= 200 && hresp.StatusCode < 300) {
		err = errors.New("http request failed with non-200 status code")
		return
	}

	err = json.NewDecoder(hresp.Body).Decode(&resp)

	if err != nil {
		err = errors.Wrap(err, "json decode failed")
		return
	}

	return
}

// WaitForNetworkSync continually polls the connected stellar-core until it
// receives a response that indicated the node has synced with the network
func (c *Client) WaitForNetworkSync(ctx context.Context) error {
	for {
		info, err := c.Info(ctx)

		if err != nil {
			return errors.Wrap(err, "info request failed")
		}

		if info.IsSynced() {
			return nil
		}

		// wait for next attempt or error if canceled while waiting
		select {
		case <-ctx.Done():
			return errors.New("canceled")
		case <-time.After(5 * time.Second):
			continue
		}
	}
}

func (c *Client) http() HTTP {
	if c.HTTP == nil {
		return http.DefaultClient
	}

	return c.HTTP
}
