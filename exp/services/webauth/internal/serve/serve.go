package serve

import (
	"fmt"
	"net/http"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/support/jwtkey"
	"github.com/stellar/go/keypair"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
)

type Options struct {
	Logger             *supportlog.Entry
	HorizonURL         string
	Port               int
	NetworkPassphrase  string
	SigningKey         string
	ChallengeExpiresIn time.Duration
	JWTPrivateKey      string
	JWTExpiresIn       time.Duration
}

func Serve(opts Options) {
	handler := handler(opts)
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

func handler(opts Options) http.Handler {
	signingKey, err := keypair.ParseFull(opts.SigningKey)
	if err != nil {
		opts.Logger.Fatalf("Error parsing signing key seed: %v", err)
	}

	jwtPrivateKey, err := jwtkey.PrivateKeyFromString(opts.JWTPrivateKey)
	if err != nil {
		opts.Logger.Fatalf("Error parsing JWT private key: %v", err)
	}

	httpClient := &http.Client{
		Timeout: horizonclient.HorizonTimeOut,
	}

	horizonClient := &horizonclient.Client{
		HorizonURL: opts.HorizonURL,
		HTTP:       httpClient,
	}
	horizonClient.SetHorizonTimeOut(uint(horizonclient.HorizonTimeOut))

	mux := supporthttp.NewAPIMux()

	mux.NotFound(errorHandler{Error: notFound}.ServeHTTP)
	mux.MethodNotAllowed(errorHandler{Error: methodNotAllowed}.ServeHTTP)

	mux.Get("/", challengeHandler{
		Logger:             opts.Logger,
		NetworkPassphrase:  opts.NetworkPassphrase,
		SigningKey:         signingKey,
		ChallengeExpiresIn: opts.ChallengeExpiresIn,
	}.ServeHTTP)
	mux.Post("/", tokenHandler{
		Logger:            opts.Logger,
		HorizonClient:     horizonClient,
		NetworkPassphrase: opts.NetworkPassphrase,
		SigningAddress:    signingKey.FromAddress(),
		JWTPrivateKey:     jwtPrivateKey,
		JWTExpiresIn:      opts.JWTExpiresIn,
	}.ServeHTTP)

	return mux
}
