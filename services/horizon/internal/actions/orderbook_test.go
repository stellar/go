package actions

import (
	"math"
	"strconv"
	"testing"

	"github.com/stellar/go/exp/orderbook"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
)

type intObject int

func (i intObject) Equals(other StreamableObjectResponse) bool {
	return i == other.(intObject)
}

func TestOrderBookResponseEquals(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		response protocol.OrderBookSummary
		other    StreamableObjectResponse
		expected bool
	}{
		{
			"empty orderbook summary",
			protocol.OrderBookSummary{},
			OrderBookResponse{},
			true,
		},
		{
			"types don't match",
			protocol.OrderBookSummary{},
			intObject(0),
			false,
		},
		{
			"buying asset doesn't match",
			protocol.OrderBookSummary{
				Buying: protocol.Asset{
					Type: "native",
				},
				Selling: protocol.Asset{
					Type: "native",
				},
			},
			OrderBookResponse{
				protocol.OrderBookSummary{
					Buying: protocol.Asset{
						Type:   "credit_alphanum4",
						Code:   "USD",
						Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
					},
					Selling: protocol.Asset{
						Type: "native",
					},
				},
			},
			false,
		},
		{
			"selling asset doesn't match",
			protocol.OrderBookSummary{
				Selling: protocol.Asset{
					Type: "native",
				},
				Buying: protocol.Asset{
					Type: "native",
				},
			},
			OrderBookResponse{
				protocol.OrderBookSummary{
					Selling: protocol.Asset{
						Type:   "credit_alphanum4",
						Code:   "USD",
						Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
					},
					Buying: protocol.Asset{
						Type: "native",
					},
				},
			},
			false,
		},
		{
			"bid lengths don't match",
			protocol.OrderBookSummary{
				Selling: protocol.Asset{
					Type:   "credit_alphanum4",
					Code:   "USD",
					Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
				},
				Buying: protocol.Asset{
					Type: "native",
				},
				Bids: []protocol.PriceLevel{
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 1},
						Price:  "1.0",
						Amount: "123",
					},
				},
			},
			OrderBookResponse{
				protocol.OrderBookSummary{
					Selling: protocol.Asset{
						Type:   "credit_alphanum4",
						Code:   "USD",
						Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
					},
					Buying: protocol.Asset{
						Type: "native",
					},
					Bids: []protocol.PriceLevel{
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
					},
				},
			},
			false,
		},
		{
			"ask lengths don't match",
			protocol.OrderBookSummary{
				Selling: protocol.Asset{
					Type:   "credit_alphanum4",
					Code:   "USD",
					Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
				},
				Buying: protocol.Asset{
					Type: "native",
				},
				Asks: []protocol.PriceLevel{
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 1},
						Price:  "1.0",
						Amount: "123",
					},
				},
			},
			OrderBookResponse{
				protocol.OrderBookSummary{
					Selling: protocol.Asset{
						Type:   "credit_alphanum4",
						Code:   "USD",
						Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
					},
					Buying: protocol.Asset{
						Type: "native",
					},
					Asks: []protocol.PriceLevel{
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
					},
				},
			},
			false,
		},
		{
			"bids don't match",
			protocol.OrderBookSummary{
				Selling: protocol.Asset{
					Type:   "credit_alphanum4",
					Code:   "USD",
					Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
				},
				Buying: protocol.Asset{
					Type: "native",
				},
				Bids: []protocol.PriceLevel{
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 1},
						Price:  "1.0",
						Amount: "123",
					},
				},
			},
			OrderBookResponse{
				protocol.OrderBookSummary{
					Selling: protocol.Asset{
						Type:   "credit_alphanum4",
						Code:   "USD",
						Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
					},
					Buying: protocol.Asset{
						Type: "native",
					},
					Bids: []protocol.PriceLevel{
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 2, D: 1},
							Price:  "2.0",
							Amount: "123",
						},
					},
				},
			},
			false,
		},
		{
			"asks don't match",
			protocol.OrderBookSummary{
				Selling: protocol.Asset{
					Type:   "credit_alphanum4",
					Code:   "USD",
					Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
				},
				Buying: protocol.Asset{
					Type: "native",
				},
				Asks: []protocol.PriceLevel{
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 1},
						Price:  "1.0",
						Amount: "123",
					},
				},
			},
			OrderBookResponse{
				protocol.OrderBookSummary{
					Selling: protocol.Asset{
						Type:   "credit_alphanum4",
						Code:   "USD",
						Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
					},
					Buying: protocol.Asset{
						Type: "native",
					},
					Asks: []protocol.PriceLevel{
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 1},
							Price:  "1.0",
							Amount: "12",
						},
					},
				},
			},
			false,
		},
		{
			"orderbook summaries match",
			protocol.OrderBookSummary{
				Selling: protocol.Asset{
					Type:   "credit_alphanum4",
					Code:   "USD",
					Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
				},
				Buying: protocol.Asset{
					Type: "native",
				},
				Asks: []protocol.PriceLevel{
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 1},
						Price:  "1.0",
						Amount: "123",
					},
				},
				Bids: []protocol.PriceLevel{
					protocol.PriceLevel{
						PriceR: protocol.Price{N: 1, D: 3},
						Price:  "0.33",
						Amount: "13",
					},
				},
			},
			OrderBookResponse{
				protocol.OrderBookSummary{
					Selling: protocol.Asset{
						Type:   "credit_alphanum4",
						Code:   "USD",
						Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
					},
					Buying: protocol.Asset{
						Type: "native",
					},
					Bids: []protocol.PriceLevel{
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 3},
							Price:  "0.33",
							Amount: "13",
						},
					},
					Asks: []protocol.PriceLevel{
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
						protocol.PriceLevel{
							PriceR: protocol.Price{N: 1, D: 1},
							Price:  "1.0",
							Amount: "123",
						},
					},
				},
			},
			true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			equals := (OrderBookResponse{testCase.response}).Equals(testCase.other)
			if equals != testCase.expected {
				t.Fatalf("expected %v but got %v", testCase.expected, equals)
			}
		})
	}
}

func TestOrderbookGetResourceValidation(t *testing.T) {
	graph := orderbook.NewOrderBookGraph()
	handler := GetOrderbookHandler{
		OrderBookGraph: graph,
	}

	var eurAssetType, eurAssetCode, eurAssetIssuer string
	if err := eurAsset.Extract(&eurAssetType, &eurAssetCode, &eurAssetIssuer); err != nil {
		t.Fatalf("cound not extract eur asset: %v", err)
	}
	var usdAssetType, usdAssetCode, usdAssetIssuer string
	if err := eurAsset.Extract(&usdAssetType, &usdAssetCode, &usdAssetIssuer); err != nil {
		t.Fatalf("cound not extract usd asset: %v", err)
	}

	for _, testCase := range []struct {
		name        string
		queryParams map[string]string
	}{
		{
			"missing all params",
			map[string]string{},
		},
		{
			"missing buying asset",
			map[string]string{
				"selling_asset_type":   eurAssetType,
				"selling_asset_code":   eurAssetCode,
				"selling_asset_issuer": eurAssetIssuer,
				"limit":                "25",
			},
		},
		{
			"missing selling asset",
			map[string]string{
				"buying_asset_type":   eurAssetType,
				"buying_asset_code":   eurAssetCode,
				"buying_asset_issuer": eurAssetIssuer,
				"limit":               "25",
			},
		},
		{
			"invalid buying asset",
			map[string]string{
				"buying_asset_type":    "invalid",
				"buying_asset_code":    eurAssetCode,
				"buying_asset_issuer":  eurAssetIssuer,
				"selling_asset_type":   usdAssetType,
				"selling_asset_code":   usdAssetCode,
				"selling_asset_issuer": usdAssetIssuer,
				"limit":                "25",
			},
		},
		{
			"invalid selling asset",
			map[string]string{
				"buying_asset_type":    eurAssetType,
				"buying_asset_code":    eurAssetCode,
				"buying_asset_issuer":  eurAssetIssuer,
				"selling_asset_type":   "invalid",
				"selling_asset_code":   usdAssetCode,
				"selling_asset_issuer": usdAssetIssuer,
				"limit":                "25",
			},
		},
		{
			"limit is not a number",
			map[string]string{
				"buying_asset_type":    eurAssetType,
				"buying_asset_code":    eurAssetCode,
				"buying_asset_issuer":  eurAssetIssuer,
				"selling_asset_type":   usdAssetType,
				"selling_asset_code":   usdAssetCode,
				"selling_asset_issuer": usdAssetIssuer,
				"limit":                "avcdef",
			},
		},
		{
			"limit is negative",
			map[string]string{
				"buying_asset_type":    eurAssetType,
				"buying_asset_code":    eurAssetCode,
				"buying_asset_issuer":  eurAssetIssuer,
				"selling_asset_type":   usdAssetType,
				"selling_asset_code":   usdAssetCode,
				"selling_asset_issuer": usdAssetIssuer,
				"limit":                "-1",
			},
		},
		{
			"limit is too high",
			map[string]string{
				"buying_asset_type":    eurAssetType,
				"buying_asset_code":    eurAssetCode,
				"buying_asset_issuer":  eurAssetIssuer,
				"selling_asset_type":   usdAssetType,
				"selling_asset_code":   usdAssetCode,
				"selling_asset_issuer": usdAssetIssuer,
				"limit":                "20000",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			r := makeRequest(t, testCase.queryParams, map[string]string{})
			_, err := handler.GetResource(r)
			if err == nil || err.Error() != invalidOrderBook.Error() {
				t.Fatalf("expected error %v but got %v", invalidOrderBook, err)
			}
		})
	}
}

func TestOrderbookGetResource(t *testing.T) {
	var eurAssetType, eurAssetCode, eurAssetIssuer string
	if err := eurAsset.Extract(&eurAssetType, &eurAssetCode, &eurAssetIssuer); err != nil {
		t.Fatalf("cound not extract eur asset: %v", err)
	}

	empty := OrderBookResponse{
		OrderBookSummary: protocol.OrderBookSummary{
			Bids: []protocol.PriceLevel{},
			Asks: []protocol.PriceLevel{},
			Selling: protocol.Asset{
				Type: "native",
			},
			Buying: protocol.Asset{
				Type:   eurAssetType,
				Code:   eurAssetCode,
				Issuer: eurAssetIssuer,
			},
		},
	}

	asksButNoBidsGraph := orderbook.NewOrderBookGraph()
	if err := asksButNoBidsGraph.AddOffer(twoEurOffer).Apply(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	asksButNoBidsResponse := empty
	asksButNoBidsResponse.Asks = []protocol.PriceLevel{
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(twoEurOffer.Price.N), D: int32(twoEurOffer.Price.D)},
			Price:  "2.0000000",
			Amount: "0.0000500",
		},
	}

	sellEurOffer := twoEurOffer
	sellEurOffer.Buying, sellEurOffer.Selling = sellEurOffer.Selling, sellEurOffer.Buying
	sellEurOffer.OfferId = 15
	bidsButNoAsksGraph := orderbook.NewOrderBookGraph()
	if err := bidsButNoAsksGraph.AddOffer(sellEurOffer).Apply(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	bidsButNoAsksResponse := empty
	bidsButNoAsksResponse.Bids = []protocol.PriceLevel{
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(sellEurOffer.Price.D), D: int32(sellEurOffer.Price.N)},
			Price:  "0.5000000",
			Amount: "0.0000500",
		},
	}

	fullGraph := orderbook.NewOrderBookGraph()
	if err := fullGraph.AddOffer(twoEurOffer).Apply(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	otherEurOffer := twoEurOffer
	otherEurOffer.Amount = xdr.Int64(math.MaxInt64)
	otherEurOffer.OfferId = 16
	if err := fullGraph.AddOffer(otherEurOffer).Apply(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	threeEurOffer := twoEurOffer
	threeEurOffer.Price.N = 3
	threeEurOffer.OfferId = 20
	if err := fullGraph.AddOffer(threeEurOffer).Apply(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	sellEurOffer.Price.N = 9
	sellEurOffer.Price.D = 10
	if err := fullGraph.AddOffer(sellEurOffer).Apply(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	otherSellEurOffer := sellEurOffer
	otherSellEurOffer.OfferId = 17
	otherSellEurOffer.Price.N *= 2
	if err := fullGraph.AddOffer(otherSellEurOffer).Apply(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	fullResponse := empty
	fullResponse.Asks = []protocol.PriceLevel{
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(twoEurOffer.Price.N), D: int32(twoEurOffer.Price.D)},
			Price:  "2.0000000",
			Amount: "922337203685.4776307",
		},
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(threeEurOffer.Price.N), D: int32(threeEurOffer.Price.D)},
			Price:  "3.0000000",
			Amount: "0.0000500",
		},
	}
	fullResponse.Bids = []protocol.PriceLevel{
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(sellEurOffer.Price.D), D: int32(sellEurOffer.Price.N)},
			Price:  "1.1111111",
			Amount: "0.0000500",
		},
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(otherSellEurOffer.Price.D), D: int32(otherSellEurOffer.Price.N)},
			Price:  "0.5555556",
			Amount: "0.0000500",
		},
	}

	limitResponse := empty
	limitResponse.Asks = []protocol.PriceLevel{
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(twoEurOffer.Price.N), D: int32(twoEurOffer.Price.D)},
			Price:  "2.0000000",
			Amount: "922337203685.4776307",
		},
	}
	limitResponse.Bids = []protocol.PriceLevel{
		protocol.PriceLevel{
			PriceR: protocol.Price{N: int32(sellEurOffer.Price.D), D: int32(sellEurOffer.Price.N)},
			Price:  "1.1111111",
			Amount: "0.0000500",
		},
	}

	for _, testCase := range []struct {
		name     string
		graph    *orderbook.OrderBookGraph
		limit    int
		expected OrderBookResponse
	}{
		{
			"empty orderbook",
			orderbook.NewOrderBookGraph(),
			10,
			empty,
		},
		{
			"orderbook with asks but no bids",
			asksButNoBidsGraph,
			10,
			asksButNoBidsResponse,
		},
		{
			"orderbook with bids but no asks",
			bidsButNoAsksGraph,
			10,
			bidsButNoAsksResponse,
		},
		{
			"full orderbook",
			fullGraph,
			10,
			fullResponse,
		},
		{
			"limit request",
			fullGraph,
			1,
			limitResponse,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler := GetOrderbookHandler{
				OrderBookGraph: testCase.graph,
			}
			r := makeRequest(
				t,
				map[string]string{
					"buying_asset_type":   eurAssetType,
					"buying_asset_code":   eurAssetCode,
					"buying_asset_issuer": eurAssetIssuer,
					"selling_asset_type":  "native",
					"limit":               strconv.Itoa(testCase.limit),
				},
				map[string]string{},
			)
			response, err := handler.GetResource(r)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !response.Equals(testCase.expected) {
				t.Fatalf("expected %v but got %v", testCase.expected, response)
			}
		})
	}
}
