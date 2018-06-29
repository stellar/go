// Package txsub provides a pluggable handler that satisfies the Stellar
// transaction submission implementation defined in support/txsub. Add an
// instance of `Handler` onto your router to allow a server to satisfy the protocol.
//
// The central type in this package is the "Driver" interface.  Implementing
// this interface allows a developer to plug in their own back end for the
// submission service.
package txsub

import (
	"context"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/txsub"
)

// Handler represents an http handler that can service http requests that
// conform to the Stellar txsub servuce.  This handler should be added to
// your chosen mux at the path `/tx` (and for good measure `/tx/` if your
// middleware doesn't normalize trailing slashes).
type Handler struct {
	Driver   Driver
	Context  context.Context
	Result   txsub.Result
	Err      error
	Ticks    *time.Ticker
	Resource TransactionSuccess
}

// Driver is a wrapper around the configurable parts of txsub.System. By requiring
// that the following methods are implemented, we have essentially created an
// interface for the txsub.System struct. You may notice we have not included the
// SubmitOnce() method here, that is because that method does not need to be
// exposed to the Hanlder struct.
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
	client *horizon.Client
}

// HorizonProxySequenceProvider represents a Horizon proxy that can lookup the current
// sequence number of an account.
type HorizonProxySequenceProvider struct {
	client *horizon.Client
}

// HorizonProxySubmitterProvider represents the high-level "submit a transaction to
// an upstream horizon" provider.
type HorizonProxySubmitterProvider struct {
	client *horizon.Client
}

// TransactionSuccess is ported from protocols/horizon without the links. Here links are
// not needed since a) it is not necessarilly clear which domain to fill in (in the
// horizon implementation of txsub, context was used to grab the local horizon domain)
// and b) the hash is returned and that is sufficient to query the transaction in
// question.
type TransactionSuccess struct {
	Hash   string `json:"hash"`
	Ledger int32  `json:"ledger"`
	Env    string `json:"envelope_xdr"`
	Result string `json:"result_xdr"`
	Meta   string `json:"result_meta_xdr"`
}
