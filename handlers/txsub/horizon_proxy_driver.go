package txsub

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/support/txsub"
	"github.com/stellar/go/support/txsub/sequence"
)

// NewHorizonProxyDriver utilizes the config params to construct and
// return a fully initialized HorizonProxyDriver.
func NewHorizonProxyDriver(client *horizon.Client, networkPassphrase string) HorizonProxyDriver {
	txsub := &txsub.System{
		Pending:           txsub.NewDefaultSubmissionList(),
		Submitter:         &HorizonProxySubmitterProvider{client: client},
		SubmissionQueue:   sequence.NewManager(),
		Results:           &HorizonProxyResultProvider{client: client},
		Sequences:         &HorizonProxySequenceProvider{client: client},
		NetworkPassphrase: networkPassphrase,
	}

	driver := HorizonProxyDriver{submissionSystem: txsub}

	return driver
}

// SubmitTransaction is a wrapper around the txsub.System's default submission method.
func (d HorizonProxyDriver) SubmitTransaction(ctx context.Context, env string) (result <-chan txsub.Result) {
	return d.submissionSystem.Submit(ctx, env)
}

// Tick is a wrapper around the txsub.System's default tick method.
func (d HorizonProxyDriver) Tick(ctx context.Context) {
	d.submissionSystem.Tick(ctx)
}

// ResultByHash implements txsub.ResultProvider, utilizing an upstream horizon instance.
func (d *HorizonProxyResultProvider) ResultByHash(ctx context.Context, transactionID string) txsub.Result {
	tx, err := d.client.LoadTransaction(transactionID)
	if err == nil {
		return txsub.Result{
			Hash:           tx.Hash,
			LedgerSequence: tx.Ledger,
			EnvelopeXDR:    tx.EnvelopeXdr,
			ResultXDR:      tx.ResultXdr,
			ResultMetaXDR:  tx.ResultMetaXdr,
		}
	}

	switch e := err.(type) {
	case *horizon.Error:
		p := e.Problem.ToProblem()

		if p.Title == problem.NotFound.Title {
			return txsub.Result{Err: txsub.ErrNoResults}
		}
		return txsub.Result{Err: p}
	default:
		return txsub.Result{Err: err}
	}
}

// Get txsub.SequenceProvider, utilizing an upstream horizon instance.
func (d *HorizonProxySequenceProvider) Get(addys []string) (map[string]uint64, error) {
	results := make(map[string]uint64)
	var mainError error

	var wg sync.WaitGroup

	var resultsMutex sync.Mutex
	var errorMutex sync.Mutex

	for _, addy := range addys {
		wg.Add(1)
		go func(addy string) {
			defer wg.Done()

			a, err := d.client.LoadAccount(addy)
			if err != nil {
				errorMutex.Lock()
				mainError = err
				errorMutex.Unlock()
				return
			}

			seq, err := strconv.ParseUint(a.Sequence, 10, 64)
			if err != nil {
				errorMutex.Lock()
				mainError = err
				errorMutex.Unlock()
				return
			}

			resultsMutex.Lock()
			results[addy] = seq
			resultsMutex.Unlock()
		}(addy)
	}

	wg.Wait()
	return results, mainError
}

// Submit sends the provided envelope to an upstream horizon and parses the response into
// a txsub.SubmissionResult.
func (d *HorizonProxySubmitterProvider) Submit(ctx context.Context, env string) (result txsub.SubmissionResult) {
	start := time.Now()
	defer func() { result.Duration = time.Since(start) }()

	_, err := d.client.SubmitTransaction(env)
	if err != nil {
		result.Err = errors.Wrap(err, "failed to submit")
		return
	}

	return
}
