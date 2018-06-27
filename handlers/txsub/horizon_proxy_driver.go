package txsub

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/txsub"
	"github.com/stellar/go/support/txsub/sequence"
)

func InitHorizonProxyDriver(horizonUrl, networkPassphrase string) HorizonProxyDriver {
	client := horizon.Client{
		URL:  horizonUrl,
		HTTP: http.DefaultClient,
	}

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

func (d HorizonProxyDriver) SubmitTransaction(ctx context.Context, env string) (result <-chan txsub.Result) {
	fmt.Println("Inside")
	return d.submissionSystem.Submit(ctx, env)
}

func (d HorizonProxyDriver) Tick(ctx context.Context) {
	d.submissionSystem.Tick(ctx)
}

func (d *HorizonProxyResultProvider) ResultByHash(cts context.Context, transactionID string) txsub.Result {
	hr, err := d.client.LoadTransaction(transactionID)
	if err == nil {
		return txResultFromHistory(hr)
	}

	// if no result was found, return ErrNoResults
	return txsub.Result{Err: txsub.ErrNoResults}
}

func txResultFromHistory(tx horizon.Transaction) txsub.Result {
	return txsub.Result{
		Hash:           tx.Hash,
		LedgerSequence: tx.Ledger,
		EnvelopeXDR:    tx.EnvelopeXdr,
		ResultXDR:      tx.ResultXdr,
		ResultMetaXDR:  tx.ResultMetaXdr,
	}
}

func (d *HorizonProxySequenceProvider) Get(addys []string) (map[string]uint64, error) {
	results := make(map[string]uint64)

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
