package txsub

import (
	"context"
	"strconv"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/txsub"
	"github.com/stellar/go/support/txsub/sequence"
)

// NewHorizonProxyDriver utilizes the config params to construct and
// return a full initialized HorizonProxyDriver.
func NewHorizonProxyDriver(client horizon.Client, networkPassphrase string) HorizonProxyDriver {
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
func (d *HorizonProxyResultProvider) ResultByHash(cts context.Context, transactionID string) txsub.Result {
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

	// if no result was found, return ErrNoResults
	return txsub.Result{Err: txsub.ErrNoResults}
}

// Get txsub.SequenceProvider, tilizing an upstream horizon instance.
func (d *HorizonProxySequenceProvider) Get(addys []string) (map[string]uint64, error) {
	results := make(map[string]uint64)

	// TODO: this is inefficient.
	// https://github.com/stellar/go/issues/522
	for _, addy := range addys {
		a, err := d.client.LoadAccount(addy)
		if err != nil {
			return nil, err
		}

		seq, err := strconv.ParseUint(a.Sequence, 10, 64)
		if err != nil {
			return nil, err
		}

		results[addy] = seq
	}

	return results, nil
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
