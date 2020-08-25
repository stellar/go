package internal

import (
	"context"
	"sync"
	"time"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
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

// PrepareRange executes the PrepareRange operation on the captive core instance.
func (c *CaptiveCoreAPI) PrepareRange(ledgerRange ledgerbackend.Range) (ledgerbackend.PrepareRangeResponse, error) {
	c.activeRequest.Lock()
	defer c.activeRequest.Unlock()
	if c.ctx.Err() != nil {
		return ledgerbackend.PrepareRangeResponse{}, errors.New("Cannot prepare range when shut down")
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

		return ledgerbackend.PrepareRangeResponse{
			LedgerRange:   ledgerRange,
			StartTime:     c.activeRequest.startTime,
			Ready:         false,
			ReadyDuration: 0,
		}, nil
	}

	return ledgerbackend.PrepareRangeResponse{
		LedgerRange:   c.activeRequest.ledgerRange,
		StartTime:     c.activeRequest.startTime,
		Ready:         c.activeRequest.ready,
		ReadyDuration: c.activeRequest.readyDuration,
	}, nil
}

// GetLatestLedgerSequence determines the latest ledger sequence available on the captive core instance.
func (c *CaptiveCoreAPI) GetLatestLedgerSequence() (ledgerbackend.LatestLedgerSequenceResponse, error) {
	c.activeRequest.RLock()
	defer c.activeRequest.RUnlock()

	if !c.activeRequest.valid {
		return ledgerbackend.LatestLedgerSequenceResponse{}, ErrMissingPrepareRange
	}
	if !c.activeRequest.ready {
		return ledgerbackend.LatestLedgerSequenceResponse{}, ErrPrepareRangeNotReady
	}

	seq, err := c.core.GetLatestLedgerSequence()
	return ledgerbackend.LatestLedgerSequenceResponse{Sequence: seq}, err
}

// GetLedger fetches the ledger with the given sequence number from the captive core instance.
func (c *CaptiveCoreAPI) GetLedger(sequence uint32) (ledgerbackend.LedgerResponse, error) {
	c.activeRequest.RLock()
	defer c.activeRequest.RUnlock()

	if !c.activeRequest.valid {
		return ledgerbackend.LedgerResponse{}, ErrMissingPrepareRange
	}
	if !c.activeRequest.ready {
		return ledgerbackend.LedgerResponse{}, ErrPrepareRangeNotReady
	}

	present, ledger, err := c.core.GetLedger(sequence)
	return ledgerbackend.LedgerResponse{
		Present: present,
		Ledger:  ledgerbackend.Base64Ledger(ledger),
	}, err
}
