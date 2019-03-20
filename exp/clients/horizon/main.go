// package horizonclient is an experimental horizon client that provides access to the horizon server
package horizonclient

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/render/problem"
)

// cursor represents `cursor` param in queries
type cursor string

// limit represents `limit` param in queries
type limit uint

// Order represents `order` param in queries
type Order string

// assetCode represets `asset_code` param in queries
type assetCode string

// assetIssuer represents `asset_issuer` param in queries
type assetIssuer string

// includeFailed represents `include_failed` param in queries
type includeFailed bool

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

// Error struct contains the problem returned by Horizon
type Error struct {
	Response *http.Response
	Problem  problem.P
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
	AccountDetail(request AccountRequest) (hProtocol.Account, error)
	AccountData(request AccountRequest) (hProtocol.AccountData, error)
	Effects(request EffectRequest) (hProtocol.EffectsPage, error)
	Assets(request AssetRequest) (hProtocol.AssetsPage, error)
	Ledgers(request LedgerRequest) (hProtocol.LedgersPage, error)
	LedgerDetail(sequence uint32) (hProtocol.Ledger, error)
	Metrics() (hProtocol.Metrics, error)
	Stream(ctx context.Context, request StreamRequest, handler func(interface{})) error
	FeeStats() (hProtocol.FeeStats, error)
	Offers(request OfferRequest) (hProtocol.OffersPage, error)
	Operations(request OperationRequest) (operations.OperationsPage, error)
	OperationDetail(id string) (operations.Operation, error)
}

// DefaultTestNetClient is a default client to connect to test network
var DefaultTestNetClient = &Client{
	HorizonURL: "https://horizon-testnet.stellar.org/",
	HTTP:       http.DefaultClient,
}

// DefaultPublicNetClient is a default client to connect to public network
var DefaultPublicNetClient = &Client{
	HorizonURL: "https://horizon.stellar.org/",
	HTTP:       http.DefaultClient,
}

// HorizonRequest contains methods implemented by request structs for horizon endpoints
type HorizonRequest interface {
	BuildUrl() (string, error)
}

// HorizonRequest contains methods implemented by request structs for endpoints that support streaming
type StreamRequest interface {
	Stream(ctx context.Context, horizonURL string, handler func(interface{})) error
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
	Cursor         string
	Limit          uint
}

// AssetRequest struct contains data for getting asset details from an horizon server.
type AssetRequest struct {
	ForAssetCode   string
	ForAssetIssuer string
	Order          Order
	Cursor         string
	Limit          uint
}

// LedgerRequest struct contains data for getting ledger details from an horizon server.
type LedgerRequest struct {
	Order       Order
	Cursor      string
	Limit       uint
	forSequence uint32
}

type metricsRequest struct {
	endpoint string
}

type feeStatsRequest struct {
	endpoint string
}

// OfferRequest struct contains data for getting offers made by an account from an horizon server
type OfferRequest struct {
	ForAccount string
	Order      Order
	Cursor     string
	Limit      uint
}

// OperationRequest struct contains data for getting operation details from an horizon servers
type OperationRequest struct {
	ForAccount     string
	ForLedger      int
	ForTransaction string
	forOperationId string
	Order          Order
	Cursor         string
	Limit          uint
	IncludeFailed  bool
}
