// package horizonclient is an experimental horizon client that provides access to the horizon server
package horizonclient

import (
	"errors"
	"net/http"
	"net/url"
)

// Cursor represents `cursor` param in queries
type Cursor string

// Limit represents `limit` param in queries
type Limit uint

// Order represents `order` param in queries
type Order string

// AssetCode represets `asset_code` param in queries
type AssetCode string

// AssetIssuer represents `asset_issuer` param in queries
type AssetIssuer string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

// Error struct contains the problem returned by Horizon
type Error struct {
	Response *http.Response
	Problem  Problem
}

var (
	// ErrResultCodesNotPopulated is the error returned from a call to
	// ResultCodes() against a `Problem` value that doesn't have the
	// "result_codes" extra field populated when it is expected to be.
	ErrResultCodesNotPopulated = errors.New("result_codes not populated")

	// ErrEnvelopeNotPopulated is the error returned from a call to
	// Envelope() against a `Problem` value that doesn't have the
	// "envelope_xdr" extra field populated when it is expected to be.
	ErrEnvelopeNotPopulated = errors.New("envelope_xdr not populated")

	// ErrResultNotPopulated is the error returned from a call to
	// Result() against a `Problem` value that doesn't have the
	// "result_xdr" extra field populated when it is expected to be.
	ErrResultNotPopulated = errors.New("result_xdr not populated")
)

// HTTP represents the HTTP client that a horizon client uses to communicate
type HTTP interface {
	Do(req *http.Request) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
	PostForm(url string, data url.Values) (resp *http.Response, err error)
}

// Client struct contains data for creating an horizon client that connects to the stellar network
type Client struct {
	HorizonURL string
	HTTP       HTTP
}

// ClientInterface contains methods implemented by the horizon client
type ClientInterface interface {
	AccountDetail(request AccountRequest) (Account, error)
	AccountData(request AccountRequest) (AccountData, error)
	Effects(request EffectRequest) (EffectsPage, error)
	Assets(request AssetRequest) (AssetsPage, error)
}

// DefaultTestNetClient is a default client to connect to test network
var DefaultTestNetClient = &Client{
	HorizonURL: "https://horizon-testnet.stellar.org",
	HTTP:       http.DefaultClient,
}

// DefaultPublicNetClient is a default client to connect to public network
var DefaultPublicNetClient = &Client{
	HorizonURL: "https://horizon.stellar.org",
	HTTP:       http.DefaultClient,
}

type HorizonRequest interface {
	BuildUrl() (string, error)
}

// AccountRequest struct contains data for making requests to the accounts endpoint of an horizon server
type AccountRequest struct {
	AccountId string
	DataKey   string
}

// EffectRequest struct contains data for getting effects from an horizon server.
// ForAccount, ForLedger, ForOperation and ForTransaction: Not more than one of these can be set at a time. If none are set, the default is to return all effects.
// The query parameters (Order, Cursor and Limit) can all be set at the same time
type EffectRequest struct {
	ForAccount     string
	ForLedger      string
	ForOperation   string
	ForTransaction string
	Order          Order
	Cursor         Cursor
	Limit          Limit
}

type AssetRequest struct {
	ForAssetCode   AssetCode
	ForAssetIssuer AssetIssuer
	Order          Order
	Cursor         Cursor
	Limit          Limit
}
