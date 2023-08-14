package stellarcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
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

// drainReponse is a helper method for draining the body stream off the http
// response object and optionally close the stream. It would also update the
// error but only as long as there wasn't an error before - this would allow
// the various methods to report the first error that took place.
// in case an error was encountered during either the draining or closing of the
// stream, that error would be returned.
func drainReponse(hresp *http.Response, close bool, err *error) (outerror error) {
	_, err2 := io.Copy(ioutil.Discard, hresp.Body)
	if err2 != nil {
		if err != nil && *err == nil {
			*err = errors.Wrap(err2, "unable to read excess data from response")
		}
		outerror = err2
	}
	if close {
		err2 = hresp.Body.Close()
		if err2 != nil {
			if err != nil && *err == nil {
				*err = errors.Wrap(err2, "unable to close response body")
			}
			outerror = err2
		}
	}
	return
}

// Upgrade upgrades the protocol version running on the stellar core instance
func (c *Client) Upgrade(ctx context.Context, version int) (err error) {
	queryParams := url.Values{}
	queryParams.Add("mode", "set")
	queryParams.Add("upgradetime", "1970-01-01T00:00:00Z")
	queryParams.Add("protocolversion", strconv.Itoa(version))

	var req *http.Request
	req, err = c.simpleGet(ctx, "upgrades", queryParams)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	var hresp *http.Response
	hresp, err = c.http().Do(req)
	if err != nil {
		return errors.Wrap(err, "http request errored")
	}
	defer drainReponse(hresp, true, &err) //nolint:errcheck

	if !(hresp.StatusCode >= 200 && hresp.StatusCode < 300) {
		err = errors.New("http request failed with non-200 status code")
		return
	}
	return nil
}

// GetLedgerEntry submits a request to the stellar core instance to get the latest
// state of a given ledger entry.
func (c *Client) GetLedgerEntry(ctx context.Context, ledgerKey xdr.LedgerKey) (proto.GetLedgerEntryResponse, error) {
	b64, err := xdr.MarshalBase64(ledgerKey)
	if err != nil {
		return proto.GetLedgerEntryResponse{}, errors.Wrap(err, "failed to marshal ledger key")
	}
	q := url.Values{}
	q.Set("key", b64)

	req, err := c.simpleGet(ctx, "getledgerentry", q)
	if err != nil {
		return proto.GetLedgerEntryResponse{}, errors.Wrap(err, "failed to create request")
	}

	hresp, err := c.http().Do(req)
	if err != nil {
		return proto.GetLedgerEntryResponse{}, errors.Wrap(err, "http request errored")
	}
	defer hresp.Body.Close()

	if !(hresp.StatusCode >= 200 && hresp.StatusCode < 300) {
		if drainReponse(hresp, false, &err) != nil {
			return proto.GetLedgerEntryResponse{}, err
		}
		return proto.GetLedgerEntryResponse{}, errors.New("http request failed with non-200 status code")
	}

	responseBytes, err := io.ReadAll(hresp.Body)
	if err != nil {
		return proto.GetLedgerEntryResponse{}, errors.Wrap(err, "could not read response")
	}

	var response proto.GetLedgerEntryResponse
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		return proto.GetLedgerEntryResponse{}, errors.Wrap(err, "json decode failed: "+string(responseBytes))
	}

	return response, nil
}

// Info calls the `info` command on the connected stellar core and returns the
// provided response
func (c *Client) Info(ctx context.Context) (resp *proto.InfoResponse, err error) {
	var req *http.Request
	req, err = c.simpleGet(ctx, "info", nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create request")
		return
	}

	var hresp *http.Response
	hresp, err = c.http().Do(req)
	if err != nil {
		err = errors.Wrap(err, "http request errored")
		return
	}
	defer drainReponse(hresp, true, &err) //nolint:errcheck

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
func (c *Client) SetCursor(ctx context.Context, id string, cursor int32) (err error) {
	var req *http.Request
	req, err = c.simpleGet(ctx, "setcursor", url.Values{
		"id":     []string{id},
		"cursor": []string{fmt.Sprintf("%d", cursor)},
	})

	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	var hresp *http.Response
	hresp, err = c.http().Do(req)
	if err != nil {
		return errors.Wrap(err, "http request errored")
	}
	defer drainReponse(hresp, true, &err) //nolint:errcheck

	if !(hresp.StatusCode >= 200 && hresp.StatusCode < 300) {
		err = errors.New("http request failed with non-200 status code")
		return err
	}

	var raw []byte
	raw, err = ioutil.ReadAll(hresp.Body)
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

	var req *http.Request
	req, err = c.simpleGet(ctx, "tx", q)
	if err != nil {
		err = errors.Wrap(err, "failed to create request")
		return
	}

	var hresp *http.Response
	hresp, err = c.http().Do(req)
	if err != nil {
		err = errors.Wrap(err, "http request errored")
		return
	}
	defer drainReponse(hresp, true, &err) //nolint:errcheck

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

// ManualClose closes a ledger when Core is running in `MANUAL_CLOSE` mode
func (c *Client) ManualClose(ctx context.Context) (err error) {

	q := url.Values{}

	var req *http.Request
	req, err = c.simpleGet(ctx, "manualclose", q)
	if err != nil {
		err = errors.Wrap(err, "failed to create request")
		return
	}

	var hresp *http.Response
	hresp, err = c.http().Do(req)
	if err != nil {
		err = errors.Wrap(err, "http request errored")
		return
	}
	defer drainReponse(hresp, true, &err) //nolint:errcheck

	if !(hresp.StatusCode >= 200 && hresp.StatusCode < 300) {
		err = errors.New("http request failed with non-200 status code")
		return
	}

	// verify there wasn't an exception
	resp := struct {
		Exception string `json:"exception"`
	}{}
	if decErr := json.NewDecoder(hresp.Body).Decode(&resp); decErr != nil {
		// At this point we want to do `err = decErr`, but that breaks our unit tests.
		// we should look into this situation and figure out how to validate
		// a correct output for this command.
		return
	}
	if resp.Exception != "" {
		err = fmt.Errorf("exception in response: %s", resp.Exception)
		return
	}

	return
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

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, newURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	return req, nil
}
