package stellarcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	// HTTP is the client to use when communicating with stellar-core. If nil,
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
	_, err2 := io.Copy(io.Discard, hresp.Body)
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

	if hresp.StatusCode < 200 || hresp.StatusCode >= 300 {
		return errors.New("http request failed with non-200 status code")
	}
	return nil
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
	raw, err = io.ReadAll(hresp.Body)
	if err != nil {
		return err
	}

	body := strings.TrimSpace(string(raw))
	if body != SetCursorDone {
		return errors.Errorf("failed to set cursor on stellar-core: %s", body)
	}

	return nil
}
func (c *Client) GetLedgerEntryRaw(ctx context.Context, ledgerSeq uint32, keys ...xdr.LedgerKey) (proto.GetLedgerEntryRawResponse, error) {
	var resp proto.GetLedgerEntryRawResponse
	return resp, c.makeLedgerKeyRequest(ctx, &resp, "getledgerentryraw", ledgerSeq, keys...)
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

	if hresp.StatusCode < 200 || hresp.StatusCode >= 300 {
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

	if hresp.StatusCode < 200 || hresp.StatusCode >= 300 {
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

// rawPost returns a new POST request to the connected stellar-core using the
// provided path and the params values encoded as the request body to construct
// the result.
func (c *Client) rawPost(
	ctx context.Context,
	newPath string,
	params string,
) (*http.Request, error) {
	u, err := url.Parse(c.URL)
	if err != nil {
		return nil, errors.Wrap(err, "unparseable url")
	}

	u.Path = path.Join(u.Path, newPath)
	newURL := u.String()

	var req *http.Request
	req, err = http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		newURL,
		bytes.NewBuffer([]byte(params)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	return req, nil
}

// makeLedgerKeyRequest is a generic method to perform a request in the form
// `key=...&key=...&ledgerSeq=...` which is useful because several Stellar Core
// endpoints all use this request format. Be sure to pass `target` by reference.
func (c *Client) makeLedgerKeyRequest(
	ctx context.Context,
	target interface{},
	endpoint string,
	ledgerSeq uint32,
	keys ...xdr.LedgerKey,
) error {
	q, err := buildMultiKeyRequest(keys...)
	if err != nil {
		return err
	} else if ledgerSeq >= 2 { // optional param
		q += fmt.Sprintf("ledgerSeq=%d", ledgerSeq)
	}

	var req *http.Request
	req, err = c.rawPost(ctx, endpoint, q)
	if err != nil {
		return err
	}

	var hresp *http.Response
	hresp, err = c.http().Do(req)
	if err != nil {
		return errors.Wrap(err, "http request errored")
	}
	defer drainReponse(hresp, true, &err) //nolint:errcheck

	if hresp.StatusCode < 200 || hresp.StatusCode >= 300 {
		return fmt.Errorf("http request failed with non-200 status code (%d)", hresp.StatusCode)
	}

	// wrap returns nil if the inner error is nil
	return errors.Wrap(json.NewDecoder(hresp.Body).Decode(&target), "json decode failed")
}

// buildMultiKeyRequest is a workaround helper because, unfortunately,
// url.Values does not support multiple keys via Set(), so we have to build our
// URL parameters manually.
func buildMultiKeyRequest(keys ...xdr.LedgerKey) (string, error) {
	// The average ledger key length, according to a simple
	//
	//     SELECT AVG(LENGTH(HEX(key))) / 2 FROM ledger_entries;
	//
	// on a pubnet RPC instance is ~57.6. We can use this to preallocate a
	// string buffer for performance.
	//
	// We know that these endpoints will almost exclusively be used for
	// ContractData and the like, so we could optimize the buffer further for
	// that, but that data is harder to query since it'd involve parsing the XDR
	// from the DB to check the key type.
	q := strings.Builder{}
	q.Grow(50 * len(keys))

	for _, key := range keys {
		keyB64, err := key.MarshalBinaryBase64()
		if err != nil {
			return q.String(), errors.Wrap(err, "failed to encode LedgerKey")
		}
		q.WriteString("key=" + url.QueryEscape(keyB64) + "&")
	}

	s, _ := strings.CutSuffix(q.String(), "&") // trim trailing &
	return s, nil
}
