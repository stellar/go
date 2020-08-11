package internal

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

var (
	// ErrMissingPrepareRange is returned when attempting an operation without satisfying
	// its PrepareRange dependency
	ErrMissingPrepareRange = errors.New("PrepareRange must be called before any other operations")
	// ErrMissingPrepareRange is returned when attempting an operation before PrepareRange has finished
	// running
	ErrPrepareRangeNotReady = errors.New("PrepareRange operation is not yet complete")
)

type rangeRequest struct {
	ledgerRange   ledgerbackend.Range
	startTime     time.Time
	readyDuration int
	valid         bool
	ready         bool
	sync.RWMutex
}

// CaptiveCoreAPI manages a shared captive core subprocess and exposes an API for
// executing commands remotely on the captive core instance.
type CaptiveCoreAPI struct {
	ctx           context.Context
	cancel        context.CancelFunc
	core          ledgerbackend.LedgerBackend
	activeRequest *rangeRequest
	wg            *sync.WaitGroup
	log           *log.Entry
}

// NewCaptiveCoreAPI constructs a new CaptiveCoreAPI instance.
func NewCaptiveCoreAPI(core ledgerbackend.LedgerBackend, log *log.Entry) CaptiveCoreAPI {
	ctx, cancel := context.WithCancel(context.Background())
	return CaptiveCoreAPI{
		ctx:           ctx,
		cancel:        cancel,
		core:          core,
		log:           log,
		activeRequest: &rangeRequest{},
		wg:            &sync.WaitGroup{},
	}
}

// Shutdown disables the PrepareRange endpoint and closes
// the captive core process.
func (c *CaptiveCoreAPI) Shutdown() {
	c.activeRequest.Lock()
	c.cancel()
	c.activeRequest.Unlock()

	c.wg.Wait()
	c.core.Close()
}

func (c *CaptiveCoreAPI) startPrepareRange(ledgerRange ledgerbackend.Range) {
	defer c.wg.Done()

	err := c.core.PrepareRange(ledgerRange)

	c.activeRequest.Lock()
	defer c.activeRequest.Unlock()
	if c.ctx.Err() != nil {
		return
	}

	if !c.activeRequest.valid || c.activeRequest.ledgerRange != ledgerRange {
		c.log.WithFields(log.F{
			"requestedRange": c.activeRequest.ledgerRange,
			"valid":          c.activeRequest.valid,
			"preparedRange":  ledgerRange,
		}).Warn("Prepared range does not match requested range")
		return
	}

	if c.activeRequest.ready {
		c.log.WithField("preparedRange", ledgerRange).Warn("Prepared range already completed")
		return
	}

	if err != nil {
		c.log.WithError(err).WithField("preparedRange", ledgerRange).Warn("Could not prepare range")
		c.activeRequest.valid = false
		c.activeRequest.ready = false
		return
	}

	c.activeRequest.ready = true
	c.activeRequest.readyDuration = int(time.Since(c.activeRequest.startTime).Seconds())
}

// RangeResponse describes the status of the pending PrepareRange operation.
type RangeResponse struct {
	LedgerRange   ledgerbackend.Range `json:"ledgerRange"`
	StartTime     time.Time           `json:"startTime"`
	Ready         bool                `json:"ready"`
	ReadyDuration int                 `json:"readyDuration"`
}

// PrepareRange executes the PrepareRange operation on the captive core instance.
func (c *CaptiveCoreAPI) PrepareRange(ledgerRange ledgerbackend.Range) (RangeResponse, error) {
	c.activeRequest.Lock()
	defer c.activeRequest.Unlock()
	if c.ctx.Err() != nil {
		return RangeResponse{}, errors.New("Cannot prepare range when shut down")
	}

	if !c.activeRequest.valid || !c.activeRequest.ledgerRange.Contains(ledgerRange) {
		if c.activeRequest.valid {
			c.log.WithFields(log.F{
				"activeRange":    c.activeRequest.ledgerRange,
				"requestedRange": ledgerRange,
			}).Info("Requested range differs from previously requested range")
		}

		c.activeRequest.ledgerRange = ledgerRange
		c.activeRequest.startTime = time.Now()
		c.activeRequest.ready = false
		c.activeRequest.valid = true

		c.wg.Add(1)
		go c.startPrepareRange(ledgerRange)

		return RangeResponse{
			LedgerRange:   ledgerRange,
			StartTime:     c.activeRequest.startTime,
			Ready:         false,
			ReadyDuration: 0,
		}, nil
	}

	return RangeResponse{
		LedgerRange:   c.activeRequest.ledgerRange,
		StartTime:     c.activeRequest.startTime,
		Ready:         c.activeRequest.ready,
		ReadyDuration: c.activeRequest.readyDuration,
	}, nil
}

// LatestLedgerSequenceResponse is the response for the GetLatestLedgerSequence command.
type LatestLedgerSequenceResponse struct {
	Sequence uint32 `json:"sequence"`
}

// GetLatestLedgerSequence determines the latest ledger sequence available on the captive core instance.
func (c *CaptiveCoreAPI) GetLatestLedgerSequence() (LatestLedgerSequenceResponse, error) {
	c.activeRequest.RLock()
	defer c.activeRequest.RUnlock()

	if !c.activeRequest.valid {
		return LatestLedgerSequenceResponse{}, ErrMissingPrepareRange
	}
	if !c.activeRequest.ready {
		return LatestLedgerSequenceResponse{}, ErrPrepareRangeNotReady
	}

	seq, err := c.core.GetLatestLedgerSequence()
	return LatestLedgerSequenceResponse{Sequence: seq}, err
}

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

// LedgerResponse is the response for the GetLedger command.
type LedgerResponse struct {
	Present bool         `json:"present"`
	Ledger  Base64Ledger `json:"ledger"`
}

// GetLedger fetches the ledger with the given sequence number from the captive core instance.
func (c *CaptiveCoreAPI) GetLedger(sequence uint32) (LedgerResponse, error) {
	c.activeRequest.RLock()
	defer c.activeRequest.RUnlock()

	if !c.activeRequest.valid {
		return LedgerResponse{}, ErrMissingPrepareRange
	}
	if !c.activeRequest.ready {
		return LedgerResponse{}, ErrPrepareRangeNotReady
	}

	present, ledger, err := c.core.GetLedger(sequence)
	return LedgerResponse{
		Present: present,
		Ledger:  Base64Ledger(ledger),
	}, err
}
