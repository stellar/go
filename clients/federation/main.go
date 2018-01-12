package federation

import (
	"net/http"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/stellartoml"
)

// FederationResponseMaxSize is the maximum size of response from a federation server
const FederationResponseMaxSize = 100 * 1024

// DefaultTestNetClient is a default federation client for testnet
var DefaultTestNetClient = &Client{
	HTTP:        http.DefaultClient,
	Horizon:     horizon.DefaultTestNetClient,
	StellarTOML: stellartoml.DefaultClient,
}

// DefaultPublicNetClient is a default federation client for pubnet
var DefaultPublicNetClient = &Client{
	HTTP:        http.DefaultClient,
	Horizon:     horizon.DefaultPublicNetClient,
	StellarTOML: stellartoml.DefaultClient,
}

// Client represents a client that is capable of resolving a Stellar.toml file
// using the internet.
type Client struct {
	StellarTOML StellarTOML
	HTTP        HTTP
	Horizon     horizon.ClientInterface
	AllowHTTP   bool
}

// HTTP represents the http client that a federation client uses to make http
// requests.
type HTTP interface {
	Get(url string) (*http.Response, error)
}

// StellarTOML represents a client that can resolve a given domain name to
// stellar.toml file.  The response is used to find the federation server that a
// query should be made against.
type StellarTOML interface {
	GetStellarToml(domain string) (*stellartoml.Response, error)
}

// confirm interface conformity
var _ StellarTOML = stellartoml.DefaultClient
var _ HTTP = http.DefaultClient
