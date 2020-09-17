package serve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
	"gopkg.in/square/go-jose.v2"
)

const stellarTomlMaxSize = 100 * 1024

type Options struct {
	Logger                      *supportlog.Entry
	HorizonURL                  string
	Port                        int
	NetworkPassphrase           string
	SigningKeys                 string
	StellarTOMLDomain           string
	AuthHomeDomains             string
	ChallengeExpiresIn          time.Duration
	JWK                         string
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
	var signingKeyFull *keypair.Full
	signingKeyStrs := strings.Split(opts.SigningKeys, ",")
	signingAddresses := make([]*keypair.FromAddress, 0, len(signingKeyStrs))

	for i, signingKeyStr := range signingKeyStrs {
		signingKey, err := keypair.ParseFull(signingKeyStr)
		if err != nil {
			return nil, errors.Wrap(err, "parsing signing key seed")
		}

		// Only the first key is used for signing. The rest is for verifying challenge transactions, if any.
		if i == 0 {
			var signingKeyPub string
			signingKeyPub, err = getStellarTOMLSigningKey(opts.StellarTOMLDomain)
			if err != nil {
				opts.Logger.Errorf("Error reading SIGNING_KEY from domain %s: %v", opts.StellarTOMLDomain, err)
			}

			if err == nil && signingKey.Address() != signingKeyPub {
				opts.Logger.Error("The configured signing key does not match the private key counterpart of the SIGNING_KEY in the stellar.toml file.")
			}

			signingKeyFull = signingKey
		}
		signingAddresses = append(signingAddresses, signingKey.FromAddress())
		opts.Logger.Info("Signing key ", i, ": ", signingKey.Address())
	}

	homeDomains := strings.Split(opts.AuthHomeDomains, ",")
	trimmedHomeDomains := make([]string, 0, len(homeDomains))
	for _, homeDomain := range homeDomains {
		// In some cases the full stop (period) character is used at the end of a FQDN.
		trimmedHomeDomains = append(trimmedHomeDomains, strings.TrimSuffix(homeDomain, "."))
	}

	jwk := jose.JSONWebKey{}
	err := json.Unmarshal([]byte(opts.JWK), &jwk)
	if err != nil {
		return nil, errors.Wrap(err, "parsing JSON Web Key (JWK)")
	}
	if jwk.Algorithm == "" {
		return nil, errors.New("algorithm (alg) field must be set")
	}

	horizonTimeout := horizonclient.HorizonTimeout
	httpClient := &http.Client{
		Timeout: horizonTimeout,
	}
	horizonClient := &horizonclient.Client{
		HorizonURL: opts.HorizonURL,
		HTTP:       httpClient,
	}
	horizonClient.SetHorizonTimeout(horizonTimeout)

	mux := supporthttp.NewAPIMux(opts.Logger)

	mux.NotFound(errorHandler{Error: notFound}.ServeHTTP)
	mux.MethodNotAllowed(errorHandler{Error: methodNotAllowed}.ServeHTTP)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Get("/", challengeHandler{
		Logger:             opts.Logger,
		NetworkPassphrase:  opts.NetworkPassphrase,
		SigningKey:         signingKeyFull,
		ChallengeExpiresIn: opts.ChallengeExpiresIn,
		HomeDomains:        trimmedHomeDomains,
	}.ServeHTTP)
	mux.Post("/", tokenHandler{
		Logger:                      opts.Logger,
		HorizonClient:               horizonClient,
		NetworkPassphrase:           opts.NetworkPassphrase,
		SigningAddresses:            signingAddresses,
		JWK:                         jwk,
		JWTIssuer:                   opts.JWTIssuer,
		JWTExpiresIn:                opts.JWTExpiresIn,
		AllowAccountsThatDoNotExist: opts.AllowAccountsThatDoNotExist,
		HomeDomains:                 trimmedHomeDomains,
	}.ServeHTTP)

	return mux, nil
}

func getStellarTOMLSigningKey(domain string) (string, error) {
	var signingKeyTOML struct {
		SigningKey string `toml:"SIGNING_KEY"`
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	domain = strings.TrimRight(domain, "./")
	resp, err := httpClient.Get(fmt.Sprintf("https://%s/.well-known/stellar.toml", domain))
	if err != nil {
		return "", errors.Wrap(err, "sending http request")
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return "", errors.New("http request failed with non-200 status code")
	}

	safeResBody := io.LimitReader(resp.Body, stellarTomlMaxSize)
	_, err = toml.DecodeReader(safeResBody, &signingKeyTOML)
	if err != nil {
		return "", errors.Wrap(err, "decoding signing key")
	}

	return signingKeyTOML.SigningKey, nil
}
