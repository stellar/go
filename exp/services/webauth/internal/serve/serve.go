package serve

import (
	"fmt"
	"net/http"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/support/jwtkey"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
)

type Options struct {
	Logger                      *supportlog.Entry
	HorizonURL                  string
	Port                        int
	NetworkPassphrase           string
	SigningKey                  string
	ChallengeExpiresIn          time.Duration
	JWTPrivateKey               string
	JWTIssuer                   string
	JWTExpiresIn                time.Duration
	AllowAccountsThatDoNotExist bool
}

func Serve(opts Options) {
	handler, err := handler(opts)
	if err != nil {
		opts.Logger.Fatalf("Error: %v", err)
		return
	}

	addr := fmt.Sprintf(":%d", opts.Port)
	supporthttp.Run(supporthttp.Config{
		ListenAddr: addr,
		Handler:    handler,
		OnStarting: func() {
			opts.Logger.Info("Starting SEP-10 Web Authentication Server")
			opts.Logger.Infof("Listening on %s", addr)
		},
	})
}

func handler(opts Options) (http.Handler, error) {
	signingKey, err := keypair.ParseFull(opts.SigningKey)
	if err != nil {
		return nil, errors.Wrap(err, "parsing signing key seed")
	}

	jwtPrivateKey, err := jwtkey.PrivateKeyFromString(opts.JWTPrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "parsing JWT private key")
	}

	horizonTimeout := 1 * time.Minute
	httpClient := &http.Client{
		Timeout: horizonTimeout,
	}
	horizonClient := &horizonclient.Client{
		HorizonURL: opts.HorizonURL,
		HTTP:       httpClient,
	}
	horizonClient.SetHorizonTimeOut(uint(horizonTimeout / time.Second))

	mux := supporthttp.NewAPIMux()

	mux.NotFound(errorHandler{Error: notFound}.ServeHTTP)
	mux.MethodNotAllowed(errorHandler{Error: methodNotAllowed}.ServeHTTP)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Get("/", challengeHandler{
		Logger:             opts.Logger,
		NetworkPassphrase:  opts.NetworkPassphrase,
		SigningKey:         signingKey,
		ChallengeExpiresIn: opts.ChallengeExpiresIn,
	}.ServeHTTP)
	mux.Post("/", tokenHandler{
		Logger:                      opts.Logger,
		HorizonClient:               horizonClient,
		NetworkPassphrase:           opts.NetworkPassphrase,
		SigningAddress:              signingKey.FromAddress(),
		JWTPrivateKey:               jwtPrivateKey,
		JWTIssuer:                   opts.JWTIssuer,
		JWTExpiresIn:                opts.JWTExpiresIn,
		AllowAccountsThatDoNotExist: opts.AllowAccountsThatDoNotExist,
	}.ServeHTTP)

	return mux, nil
}
