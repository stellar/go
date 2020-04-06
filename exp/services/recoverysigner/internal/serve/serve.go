package serve

import (
	"crypto/ecdsa"
	"fmt"
	"net/http"

	firebaseauth "firebase.google.com/go/auth"
	"github.com/go-chi/chi"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db"
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
	DatabaseURL       string
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
	Logger             *supportlog.Entry
	NetworkPassphrase  string
	SigningKey         *keypair.Full
	AccountStore       account.Store
	SEP10JWTPublicKey  *ecdsa.PublicKey
	FirebaseAuthClient *firebaseauth.Client
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

	db, dbErr := db.Open(opts.DatabaseURL)
	if dbErr != nil {
		return handlerDeps{}, errors.Wrap(err, "error parsing database url")
	}
	dbErr = db.Ping()
	if dbErr != nil {
		opts.Logger.Warn("Error pinging to Database: ", dbErr)
	}
	accountStore := &account.DBStore{DB: db}

	firebaseAuthClient, err := auth.NewFirebaseAuthClient(opts.FirebaseProjectID)
	if err != nil {
		return handlerDeps{}, errors.Wrap(err, "error setting up firebase auth client")
	}

	deps := handlerDeps{
		Logger:             opts.Logger,
		NetworkPassphrase:  opts.NetworkPassphrase,
		SigningKey:         signingKey,
		AccountStore:       accountStore,
		SEP10JWTPublicKey:  sep10JWTPublicKey,
		FirebaseAuthClient: firebaseAuthClient,
	}

	return deps, nil
}

func handler(deps handlerDeps) http.Handler {
	mux := supporthttp.NewAPIMux(deps.Logger)

	mux.NotFound(errorHandler{Error: notFound}.ServeHTTP)
	mux.MethodNotAllowed(errorHandler{Error: methodNotAllowed}.ServeHTTP)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Route("/accounts", func(mux chi.Router) {
		mux.Use(auth.SEP10Middleware(deps.SEP10JWTPublicKey))
		mux.Use(auth.FirebaseMiddleware(auth.FirebaseTokenVerifierLive{AuthClient: deps.FirebaseAuthClient}))
		mux.Get("/", accountListHandler{
			Logger:         deps.Logger,
			SigningAddress: deps.SigningKey.FromAddress(),
			AccountStore:   deps.AccountStore,
		}.ServeHTTP)
		mux.Route("/{address}", func(mux chi.Router) {
			mux.Post("/", accountPostHandler{
				Logger:         deps.Logger,
				SigningAddress: deps.SigningKey.FromAddress(),
				AccountStore:   deps.AccountStore,
			}.ServeHTTP)
			// TODO: mux.Put("/", accountPutHandler{}.ServeHTTP)
			// TODO: mux.Get("/", accountGetHandler{}.ServeHTTP)
			// TODO: mux.Delete("/", accountDeleteHandler{}.ServeHTTP)
			mux.Post("/sign", accountSignHandler{
				Logger:            deps.Logger,
				SigningKey:        deps.SigningKey,
				NetworkPassphrase: deps.NetworkPassphrase,
				AccountStore:      deps.AccountStore,
			}.ServeHTTP)
		})
	})

	return mux
}
