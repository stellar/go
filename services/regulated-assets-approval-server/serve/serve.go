package serve

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	supporthttp "github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
)

type Options struct {
	IssuerAccountSecret    string
	AssetCode              string
	FriendbotPaymentAmount int
	HorizonURL             string
	NetworkPassphrase      string
	Port                   int
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

func handleHTTP(opts Options) http.Handler {
	issuerKP, err := keypair.ParseFull(opts.IssuerAccountSecret)
	if err != nil {
		log.Fatal(errors.Wrap(err, "parsing secret"))
	}
	mux := chi.NewMux()

	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(supporthttp.LoggingMiddleware)
	mux.Use(corsHandler)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Get("/.well-known/stellar.toml", stellarTOMLHandler{
		assetCode:         opts.AssetCode,
		issuerAddress:     issuerKP.Address(),
		networkPassphrase: opts.NetworkPassphrase,
	}.ServeHTTP)
	mux.Get("/friendbot", friendbotHandler{
		assetCode:           opts.AssetCode,
		issuerAccountSecret: opts.IssuerAccountSecret,
		horizonClient:       opts.horizonClient(),
		horizonURL:          opts.HorizonURL,
		networkPassphrase:   opts.NetworkPassphrase,
		paymentAmount:       opts.FriendbotPaymentAmount,
	}.ServeHTTP)
	mux.Post("/tx_approve", txApproveHandler{
		assetCode: opts.AssetCode,
		issuerKP:  issuerKP,
	}.ServeHTTP)
	return mux
}

func (opts Options) horizonClient() horizonclient.ClientInterface {
	return &horizonclient.Client{
		HorizonURL: opts.HorizonURL,
		HTTP:       &http.Client{Timeout: 30 * time.Second},
	}
}
