package serve

import (
	"fmt"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	supporthttp "github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
)

type Options struct {
	Port                int
	HorizonURL          string
	NetworkPassphrase   string
	AccountIssuerSecret string
	AssetCode           string
}

func Serve(opts Options) {
	listenAddr := fmt.Sprintf(":%d", opts.Port)
	serverConfig := supporthttp.Config{
		ListenAddr:          listenAddr,
		Handler:             handleHTTP(opts),
		TCPKeepAlive:        time.Minute * 3,
		ShutdownGracePeriod: time.Second * 50,
		ReadTimeout:         time.Second * 5,
		WriteTimeout:        time.Second * 35,
		IdleTimeout:         time.Minute * 2,
		OnStarting: func() {
			log.Info("Starting SEP-8 Approval Server")
			log.Infof("Listening on %s", listenAddr)
		},
		OnStopping: func() {
			log.Info("Stopping SEP-8 Approval Server")
		},
	}
	supporthttp.Run(serverConfig)
}

func handleHTTP(opts Options) *chi.Mux {
	mux := chi.NewMux()

	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(supporthttp.LoggingMiddleware)
	mux.Use(corsHandler)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Get("/.well-known/stellar.toml", stellarTOMLHandler(opts))
	mux.Get("/friendbot", friendbotHandler{
		assetCode:           opts.AssetCode,
		accountIssuerSecret: opts.AccountIssuerSecret,
		horizonClient:       opts.horizonClient(),
		horizonURL:          opts.HorizonURL,
		networkPassphrase:   opts.NetworkPassphrase,
	}.ServeHTTP)

	return mux
}

func (opts Options) horizonClient() horizonclient.ClientInterface {
	var client *horizonclient.Client
	if opts.NetworkPassphrase == network.PublicNetworkPassphrase {
		client = horizonclient.DefaultPublicNetClient
	} else {
		client = horizonclient.DefaultTestNetClient
	}

	client.HorizonURL = opts.HorizonURL

	return client
}
