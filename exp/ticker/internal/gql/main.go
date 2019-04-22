package gql

import (
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/stellar/go/exp/ticker/internal/gql/schema"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
)

// asset represents a Stellar asset, with some type
// adaptations to match the GraphQL type system
type asset struct {
	Code                        string
	IssuerAccount               string
	Type                        string
	NumAccounts                 int32
	AuthRequired                bool
	AuthRevocable               bool
	Amount                      float64
	AssetControlledByDomain     bool
	AnchorAssetCode             string
	AnchorAssetType             string
	IsValid                     bool
	DisplayDecimals             BigInt
	Name                        string
	Desc                        string
	Conditions                  string
	IsAssetAnchored             bool
	FixedNumber                 BigInt
	MaxNumber                   BigInt
	IsUnlimited                 bool
	RedemptionInstructions      string
	CollateralAddresses         string
	CollateralAddressSignatures string
	Countries                   string
	Status                      string
	IssuerID                    int32
}

// partialMarket represents the aggregated market data for a
// specific pair of assets since <Since>
type partialMarket struct {
	TradePair          string
	BaseAssetCode      string
	BaseAssetIssuer    string
	CounterAssetCode   string
	CounterAssetIssuer string
	BaseVolume         float64
	CounterVolume      float64
	TradeCount         int32
	Open               float64
	Low                float64
	High               float64
	Change             float64
	Close              float64
	CloseTime          graphql.Time
	Since              graphql.Time
}

type resolver struct {
	db *tickerdb.TickerSession
}

func Serve(session *tickerdb.TickerSession) {
	r := resolver{db: session}
	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers()}
	s := graphql.MustParseSchema(schema.String(), &r, opts...)

	http.Handle("/query", &relay.Handler{Schema: s})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
