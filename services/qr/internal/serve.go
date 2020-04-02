package internal

import (
	"fmt"
	"net/http"

	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
)

type Options struct {
	Logger *supportlog.Entry
	Port   int
}

func Serve(opts Options) {
	handler := handler(opts)

	addr := fmt.Sprintf(":%d", opts.Port)
	supporthttp.Run(supporthttp.Config{
		ListenAddr: addr,
		Handler:    handler,
		OnStarting: func() {
			opts.Logger.Info("Starting SEP-XX Stellar account QR code generation server")
			opts.Logger.Infof("Listening on %s", addr)
		},
	})
}

func handler(opts Options) http.Handler {
	mux := supporthttp.NewAPIMux(opts.Logger)

	mux.NotFound(errorHandler{Error: notFound}.ServeHTTP)
	mux.MethodNotAllowed(errorHandler{Error: methodNotAllowed}.ServeHTTP)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Get("/{address}.svg", qrCodeHandler{Logger: opts.Logger}.ServeHTTP)

	return mux
}
