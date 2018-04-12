package stellarcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	proto "github.com/stellar/go/protocols/stellarcore"
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

// Info calls the `info` command on the connected stellar core and returns the
// provided response
func (c *Client) Info(ctx context.Context) (resp *proto.InfoResponse, err error) {

	req, err := c.simpleGet(ctx, "info", nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create request")
		return
	}

	hresp, err := c.http().Do(req)
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

// SetCursor calls the `setcursor` command on the connected stellar core
func (c *Client) SetCursor(ctx context.Context, id string, cursor int32) error {
	req, err := c.simpleGet(ctx, "setcursor", url.Values{
		"id":     []string{id},
		"cursor": []string{fmt.Sprintf("%d", cursor)},
	})

	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	hresp, err := c.http().Do(req)
	if err != nil {
		return errors.Wrap(err, "http request errored")
	}
	defer hresp.Body.Close()

	raw, err := ioutil.ReadAll(hresp.Body)
	if err != nil {
		return err
	}

	body := strings.TrimSpace(string(raw))
	if body != SetCursorDone {
		return errors.Errorf("failed to set cursor on stellar-core: %s", body)
	}

	return nil
}

// SubmitTransaction calls the `tx` command on the connected stellar core with the provided envelope
func (c *Client) SubmitTransaction(ctx context.Context, envelope string) (resp *proto.TXResponse, err error) {

	q := url.Values{}
	q.Set("blob", envelope)

	req, err := c.simpleGet(ctx, "tx", q)
	if err != nil {
		err = errors.Wrap(err, "failed to create request")
		return
	}

	hresp, err := c.http().Do(req)
	if err != nil {
		err = errors.Wrap(err, "http request errored")
		return
	}
	defer hresp.Body.Close()

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

// simpleGet returns a new GET request to the connected stellar-core using the
// provided path and query values to construct the result.
func (c *Client) simpleGet(
	ctx context.Context,
	newPath string,
	query url.Values,
) (*http.Request, error) {

	u, err := url.Parse(c.URL)
	if err != nil {
		return nil, errors.Wrap(err, "unparseable url")
	}

	u.Path = path.Join(u.Path, newPath)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	newURL := u.String()

	req, err := http.NewRequest(http.MethodGet, newURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	return req.WithContext(ctx), nil
}
