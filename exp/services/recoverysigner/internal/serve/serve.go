package serve

import (
	"crypto/ecdsa"
	"fmt"
	"net/http"

	firebase "firebase.google.com/go"
	"github.com/go-chi/chi"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/exp/support/jwtkey"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
)

type Options struct {
	Logger            *supportlog.Entry
	HorizonURL        string
	Port              int
	NetworkPassphrase string
	SigningKey        string
	SEP10JWTPublicKey string
	FirebaseProjectID string
}

func Serve(opts Options) {
	deps, err := getHandlerDeps(opts)
	if err != nil {
		opts.Logger.Fatalf("Error: %v", err)
		return
	}

	handler := handler(deps)

	addr := fmt.Sprintf(":%d", opts.Port)
	supporthttp.Run(supporthttp.Config{
		ListenAddr: addr,
		Handler:    handler,
		OnStarting: func() {
			opts.Logger.Info("Starting SEP-XX Recover Signer server")
			opts.Logger.Infof("Listening on %s", addr)
		},
	})
}

type handlerDeps struct {
	Logger            *supportlog.Entry
	HorizonClient     horizonclient.ClientInterface
	NetworkPassphrase string
	SigningKey        *keypair.Full
	AccountStore      account.Store
	SEP10JWTPublicKey *ecdsa.PublicKey
	FirebaseApp       *firebase.App
}

func getHandlerDeps(opts Options) (handlerDeps, error) {
	// TODO: Replace this signing key with randomly generating a unique signing
	// key for each account so that it is not possible to identify which
	// accounts are recoverable via a recovery signer.
	signingKey, err := keypair.ParseFull(opts.SigningKey)
	if err != nil {
		return handlerDeps{}, errors.Wrap(err, "parsing signing key seed")
	}
	opts.Logger.Info("Signing key: ", signingKey.Address())

	sep10JWTPublicKey, err := jwtkey.PublicKeyFromString(opts.SEP10JWTPublicKey)
	if err != nil {
		return handlerDeps{}, errors.Wrap(err, "parsing SEP-10 JWT public key")
	}
	opts.Logger.Info("SEP-10 JWT Public key: ", sep10JWTPublicKey)

	horizonClient := &horizonclient.Client{
		HorizonURL: opts.HorizonURL,
		HTTP: &http.Client{
			Timeout: horizonclient.HorizonTimeOut,
		},
	}
	horizonClient.SetHorizonTimeOut(uint(horizonclient.HorizonTimeOut))

	// TODO: Replace this in-memory store with Postgres.
	accountStore := account.NewMemoryStore()

	firebaseApp, err := auth.NewFirebaseApp(opts.FirebaseProjectID)
	if err != nil {
		return handlerDeps{}, errors.Wrap(err, "error creating firebase app")
	}

	deps := handlerDeps{
		Logger:            opts.Logger,
		HorizonClient:     horizonClient,
		NetworkPassphrase: opts.NetworkPassphrase,
		SigningKey:        signingKey,
		AccountStore:      accountStore,
		SEP10JWTPublicKey: sep10JWTPublicKey,
		FirebaseApp:       firebaseApp,
	}

	return deps, nil
}

func handler(deps handlerDeps) http.Handler {
	mux := supporthttp.NewAPIMux()

	mux.NotFound(errorHandler{Error: notFound}.ServeHTTP)
	mux.MethodNotAllowed(errorHandler{Error: methodNotAllowed}.ServeHTTP)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Route("/accounts", func(mux chi.Router) {
		mux.Use(auth.SEP10Middleware(deps.SEP10JWTPublicKey))
		mux.Use(auth.FirebaseMiddleware(auth.FirebaseTokenVerifierLive{App: deps.FirebaseApp}))
		// TODO: mux.Get("/", accountListHandler{
		// TODO: 	Logger:         deps.Logger,
		// TODO: 	SigningAddress: deps.SigningKey.FromAddress(),
		// TODO: 	AccountStore:   deps.AccountStore,
		// TODO: }.ServeHTTP)
		mux.Route("/{address}", func(mux chi.Router) {
			mux.Post("/", accountPostHandler{
				Logger:         deps.Logger,
				SigningAddress: deps.SigningKey.FromAddress(),
				AccountStore:   deps.AccountStore,
				HorizonClient:  deps.HorizonClient,
			}.ServeHTTP)
			// TODO: mux.Put("/", accountPutHandler{}.ServeHTTP)
			// TODO: mux.Get("/", accountGetHandler{}.ServeHTTP)
			// TODO: mux.Delete("/", accountDeleteHandler{}.ServeHTTP)
			// TODO: mux.Post("/sign", accountSignHandler{
			// TODO: 	Logger:            deps.Logger,
			// TODO: 	SigningKey:        deps.SigningKey,
			// TODO: 	NetworkPassphrase: deps.NetworkPassphrase,
			// TODO: 	AccountStore:      deps.AccountStore,
			// TODO: }.ServeHTTP)
		})
	})

	return mux
}
