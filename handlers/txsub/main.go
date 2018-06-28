// Package txsub provides a pluggable handler that satisfies the Stellar
// transaction submission implementation defined in support/txsub. Add an
// instance of `Handler` onto your router to allow a server to satisfy the protocol.
//
// The central type in this package is the "Driver" interfaces.  Implementing
// these interfaces allows a developer to plug in their own back end and
// submission service. It also allows a developer to query public infastructure
// such as SDF's horizon.stellar.org.
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
	// SubmitTransaction submits the provided base64 encoded transaction envelope
	// to the network using this submission system.
	SubmitTransaction(context.Context, string) <-chan txsub.Result
	// Tick triggers the system to update itself with any new data available.
	Tick(ctx context.Context)
}

// HorizonProxyDriver is a pre-baked implementation of `Driver` that provides the
// entirety of the submission system necessary to satisfy the protocol.
type HorizonProxyDriver struct {
	submissionSystem *txsub.System
}

// HorizonProxyResultProvider represents a Horizon Proxy that can lookup Result objects
// by transaction hash or by [address,sequence] pairs.
type HorizonProxyResultProvider struct {
	client horizon.Client
}

// HorizonProxySequenceProvider represents a Horizon proxy that can lookup the current
// sequence number of an account.
type HorizonProxySequenceProvider struct {
	client horizon.Client
}

// HorizonProxySubmitterProvider represents the high-level "submit a transaction to
// an upstream horizon" provider.
type HorizonProxySubmitterProvider struct {
	client horizon.Client
}
