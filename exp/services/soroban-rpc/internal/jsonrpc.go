package internal

import (
	"context"
	"net/http"

	"github.com/creachadair/jrpc2/handler"
	"github.com/creachadair/jrpc2/jhttp"

	"github.com/stellar/go/exp/services/soroban-rpc/internal/methods"
	"github.com/stellar/go/support/log"
)

// Handler is the HTTP handler which serves the Soroban JSON RPC responses
type Handler struct {
	bridge           jhttp.Bridge
	logger           *log.Entry
	transactionProxy *methods.TransactionProxy
}

// Start spawns the background workers necessary for the JSON RPC handlers.
func (h Handler) Start() {
	h.transactionProxy.Start(context.Background())
}

// ServeHTTP implements the http.Handler interface
func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.bridge.ServeHTTP(w, req)
}

// Close closes all of the resources held by the Handler instances.
// After Close is called the Handler instance will stop accepting JSON RPC requests.
func (h Handler) Close() {
	if err := h.bridge.Close(); err != nil {
		h.logger.WithError(err).Warn("could not close bridge")
	}
	h.transactionProxy.Close()
}

type HandlerParams struct {
	AccountStore     methods.AccountStore
	TransactionProxy *methods.TransactionProxy
	Logger           *log.Entry
}

// NewJSONRPCHandler constructs a Handler instance
func NewJSONRPCHandler(params HandlerParams) (Handler, error) {
	return Handler{
		bridge: jhttp.NewBridge(handler.Map{
			"getHealth":            methods.NewHealthCheck(),
			"getAccount":           methods.NewAccountHandler(params.AccountStore),
			"getTransactionStatus": methods.NewGetTransactionStatusHandler(params.TransactionProxy),
			"sendTransaction":      methods.NewSendTransactionHandler(params.TransactionProxy),
		}, nil),
		logger:           params.Logger,
		transactionProxy: params.TransactionProxy,
	}, nil
}
