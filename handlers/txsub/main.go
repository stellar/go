package txsub

import (
	"context"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/txsub"
)

type Handler struct {
	Driver   Driver
	Context  context.Context
	Result   txsub.Result
	Err      error
	Ticks    *time.Ticker
	Resource horizon.TransactionSuccess
}

// Driver is a wrapper around the configurable parts of txsub.System. By requiring
// that the following methods are implemented, we have essentially created an
// interface for the txsub.System struct.
type Driver interface {
	// Need to access system since driver is an interface
	SubmitTransaction(context.Context, string) <-chan txsub.Result
	// Tick Tock
	Tick(ctx context.Context)
}

type HorizonProxyDriver struct {
	submissionSystem *txsub.System
}

type HorizonProxyResultProvider struct {
	client horizon.Client
}

type HorizonProxySequenceProvider struct {
	client horizon.Client
}

type HorizonProxySubmitterProvider struct {
	client horizon.Client
}
