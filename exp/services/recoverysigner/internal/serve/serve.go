package serve

import (
	"encoding/json"
	"fmt"
	"net/http"

	firebaseauth "firebase.google.com/go/auth"
	"github.com/go-chi/chi"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
	"gopkg.in/square/go-jose.v2"
)

type Options struct {
	Logger            *supportlog.Entry
	DatabaseURL       string
	Port              int
	NetworkPassphrase string
	SigningKey        string
	SEP10JWKS         string
	SEP10JWTIssuer    string
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
			opts.Logger.Info("Starting SEP-30 Recover Signer server")
			opts.Logger.Infof("Listening on %s", addr)
		},
	})
}

type handlerDeps struct {
	Logger             *supportlog.Entry
	NetworkPassphrase  string
	SigningKey         *keypair.Full
	AccountStore       account.Store
	SEP10JWK           jose.JSONWebKey
	SEP10JWTIssuer     string
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

	sep10JWKS := &jose.JSONWebKeySet{}
	err = json.Unmarshal([]byte(opts.SEP10JWKS), sep10JWKS)
	if err != nil {
		return handlerDeps{}, errors.Wrap(err, "parsing SEP-10 JSON Web Key (JWK) Set")
	}
	if len(sep10JWKS.Keys) == 0 {
		return handlerDeps{}, errors.New("no keys included in SEP-10 JSON Web Key (JWK) Set")
	}
	if len(sep10JWKS.Keys) > 1 {
		return handlerDeps{}, errors.New("more than one key included in SEP-10 JSON Web Key (JWK) Set only one supported")
	}
	sep10JWK := sep10JWKS.Keys[0]

	db, err := db.Open(opts.DatabaseURL)
	if err != nil {
		return handlerDeps{}, errors.Wrap(err, "error parsing database url")
	}
	err = db.Ping()
	if err != nil {
		opts.Logger.Warn("Error pinging to Database: ", err)
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
		SEP10JWK:           sep10JWK,
		SEP10JWTIssuer:     opts.SEP10JWTIssuer,
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
		mux.Use(auth.SEP10Middleware(deps.SEP10JWTIssuer, deps.SEP10JWK))
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
			mux.Put("/", accountPutHandler{
				Logger:         deps.Logger,
				SigningAddress: deps.SigningKey.FromAddress(),
				AccountStore:   deps.AccountStore,
			}.ServeHTTP)
			mux.Get("/", accountGetHandler{
				Logger:         deps.Logger,
				SigningAddress: deps.SigningKey.FromAddress(),
				AccountStore:   deps.AccountStore,
			}.ServeHTTP)
			mux.Delete("/", accountDeleteHandler{
				Logger:         deps.Logger,
				SigningAddress: deps.SigningKey.FromAddress(),
				AccountStore:   deps.AccountStore,
			}.ServeHTTP)
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
