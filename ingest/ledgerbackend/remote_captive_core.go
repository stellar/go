package ledgerbackend

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// PrepareRangeResponse describes the status of the pending PrepareRange operation.
type PrepareRangeResponse struct {
	LedgerRange   Range     `json:"ledgerRange"`
	StartTime     time.Time `json:"startTime"`
	Ready         bool      `json:"ready"`
	ReadyDuration int       `json:"readyDuration"`
}

// LatestLedgerSequenceResponse is the response for the GetLatestLedgerSequence command.
type LatestLedgerSequenceResponse struct {
	Sequence uint32 `json:"sequence"`
}

// LedgerResponse is the response for the GetLedger command.
type LedgerResponse struct {
	Present bool         `json:"present"`
	Ledger  Base64Ledger `json:"ledger"`
}

// Base64Ledger extends xdr.LedgerCloseMeta with JSON encoding and decoding
type Base64Ledger xdr.LedgerCloseMeta

func (r *Base64Ledger) UnmarshalJSON(b []byte) error {
	var base64 string
	if err := json.Unmarshal(b, &base64); err != nil {
		return err
	}

	var parsed xdr.LedgerCloseMeta
	if err := xdr.SafeUnmarshalBase64(base64, &parsed); err != nil {
		return err
	}
	*r = Base64Ledger(parsed)

	return nil
}

func (r Base64Ledger) MarshalJSON() ([]byte, error) {
	base64, err := xdr.MarshalBase64(xdr.LedgerCloseMeta(r))
	if err != nil {
		return nil, err
	}
	return json.Marshal(base64)
}

// RemoteCaptiveStellarCore is an http client for interacting with a remote captive core server.
type RemoteCaptiveStellarCore struct {
	url                      *url.URL
	client                   *http.Client
	lock                     *sync.Mutex
	cancel                   context.CancelFunc
	parentCtx                context.Context
	prepareRangePollInterval time.Duration
}

// RemoteCaptiveOption values can be passed into NewRemoteCaptive to customize a RemoteCaptiveStellarCore instance.
type RemoteCaptiveOption func(c *RemoteCaptiveStellarCore)

// PrepareRangePollInterval configures how often the captive core server will be polled when blocking
// on the PrepareRange operation.
func PrepareRangePollInterval(d time.Duration) RemoteCaptiveOption {
	return func(c *RemoteCaptiveStellarCore) {
		c.prepareRangePollInterval = d
	}
}

// NewRemoteCaptive returns a new RemoteCaptiveStellarCore instance.
//
// Only the captiveCoreURL parameter is required.
func NewRemoteCaptive(ctx context.Context, captiveCoreURL string, options ...RemoteCaptiveOption) (RemoteCaptiveStellarCore, error) {
	u, err := url.Parse(captiveCoreURL)
	if err != nil {
		return RemoteCaptiveStellarCore{}, errors.Wrap(err, "unparseable url")
	}

	client := RemoteCaptiveStellarCore{
		prepareRangePollInterval: time.Second,
		url:                      u,
		client:                   &http.Client{Timeout: 5 * time.Second},
		lock:                     &sync.Mutex{},
		parentCtx:                ctx,
	}
	for _, option := range options {
		option(&client)
	}
	return client, nil
}

func decodeResponse(response *http.Response, payload interface{}) error {
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response body")
		}

		return errors.New(string(body))
	}

	if err := json.NewDecoder(response.Body).Decode(payload); err != nil {
		return errors.Wrap(err, "failed to decode json payload")
	}
	return nil
}

// GetLatestLedgerSequence returns the sequence of the latest ledger available
// in the backend. This method returns an error if not in a session (start with
// PrepareRange).
//
// Note that for UnboundedRange the returned sequence number is not necessarily
// the latest sequence closed by the network. It's always the last value available
// in the backend.
func (c RemoteCaptiveStellarCore) GetLatestLedgerSequence() (sequence uint32, err error) {
	// TODO: Have a context on this request so we can cancel all outstanding
	// requests, not just PrepareRange.
	u := *c.url
	u.Path = path.Join(u.Path, "latest-sequence")

	response, err := c.client.Get(u.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute request")
	}

	var parsed LatestLedgerSequenceResponse
	if err = decodeResponse(response, &parsed); err != nil {
		return 0, err
	}

	return parsed.Sequence, nil
}

// Close cancels any pending PrepareRange requests.
func (c RemoteCaptiveStellarCore) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func (c RemoteCaptiveStellarCore) createContext() context.Context {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Cancel any outstanding PrepareRange request
	if c.cancel != nil {
		c.cancel()
	}

	// Make a new context for this new request.
	ctx, cancel := context.WithCancel(c.parentCtx)
	c.cancel = cancel
	return ctx
}

// PrepareRange prepares the given range (including from and to) to be loaded.
// Captive stellar-core backend needs to initalize Stellar-Core state to be
// able to stream ledgers.
// Stellar-Core mode depends on the provided ledgerRange:
//   * For BoundedRange it will start Stellar-Core in catchup mode.
//   * For UnboundedRange it will first catchup to starting ledger and then run
//     it normally (including connecting to the Stellar network).
// Please note that using a BoundedRange, currently, requires a full-trust on
// history archive. This issue is being fixed in Stellar-Core.
func (c RemoteCaptiveStellarCore) PrepareRange(ledgerRange Range) error {
	ctx := c.createContext()
	u := *c.url
	u.Path = path.Join(u.Path, "prepare-range")
	rangeBytes, err := json.Marshal(ledgerRange)
	if err != nil {
		return errors.Wrap(err, "cannot serialize range")
	}

	timer := time.NewTimer(c.prepareRangePollInterval)
	defer timer.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(rangeBytes))
		if err != nil {
			return errors.Wrap(err, "cannot construct http request")
		}

		var response *http.Response
		response, err = c.client.Do(req)
		if err != nil {
			return errors.Wrap(err, "failed to execute request")
		}

		var parsed PrepareRangeResponse
		if err = decodeResponse(response, &parsed); err != nil {
			return err
		}

		if parsed.Ready {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "shutting down")
		case <-timer.C:
			timer.Reset(c.prepareRangePollInterval)
		}
	}
}

// IsPrepared returns true if a given ledgerRange is prepared.
func (c RemoteCaptiveStellarCore) IsPrepared(ledgerRange Range) (bool, error) {
	// TODO: Have a context on this request so we can cancel all outstanding
	// requests, not just PrepareRange.
	u := *c.url
	u.Path = path.Join(u.Path, "prepare-range")
	rangeBytes, err := json.Marshal(ledgerRange)
	if err != nil {
		return false, errors.Wrap(err, "cannot serialize range")
	}
	body := bytes.NewReader(rangeBytes)

	var response *http.Response
	response, err = c.client.Post(u.String(), "application/json; charset=utf-8", body)
	if err != nil {
		return false, errors.Wrap(err, "failed to execute request")
	}

	var parsed PrepareRangeResponse
	if err = decodeResponse(response, &parsed); err != nil {
		return false, err
	}

	return parsed.Ready, nil
}

// GetLedger returns true when ledger is found and it's LedgerCloseMeta.
// Call PrepareRange first to instruct the backend which ledgers to fetch.
//
// CaptiveStellarCore requires PrepareRange call first to initialize Stellar-Core.
// Requesting a ledger on non-prepared backend will return an error.
//
// Because data is streamed from Stellar-Core ledger after ledger user should
// request sequences in a non-decreasing order. If the requested sequence number
// is less than the last requested sequence number, an error will be returned.
//
// This function behaves differently for bounded and unbounded ranges:
//   * BoundedRange makes GetLedger blocking if the requested ledger is not yet
//     available in the ledger. After getting the last ledger in a range this
//     method will also Close() the backend.
//   * UnboundedRange makes GetLedger non-blocking. The method will return with
//     the first argument equal false.
// This is done to provide maximum performance when streaming old ledgers.
func (c RemoteCaptiveStellarCore) GetLedger(sequence uint32) (bool, xdr.LedgerCloseMeta, error) {
	// TODO: Have a context on this request so we can cancel all outstanding
	// requests, not just PrepareRange.
	u := *c.url
	u.Path = path.Join(u.Path, "ledger", strconv.FormatUint(uint64(sequence), 10))

	response, err := c.client.Get(u.String())
	if err != nil {
		return false, xdr.LedgerCloseMeta{}, errors.Wrap(err, "failed to execute request")
	}

	var parsed LedgerResponse
	if err = decodeResponse(response, &parsed); err != nil {
		return false, xdr.LedgerCloseMeta{}, err
	}

	return parsed.Present, xdr.LedgerCloseMeta(parsed.Ledger), nil
}

func (c RemoteCaptiveStellarCore) GetLedgerBlocking(sequence uint32) (xdr.LedgerCloseMeta, error) {
	for {
		exists, meta, err := c.GetLedger(sequence)
		if err != nil {
			return xdr.LedgerCloseMeta{}, err
		}

		if exists {
			return meta, nil
		} else {
			time.Sleep(time.Second)
		}
	}
}
