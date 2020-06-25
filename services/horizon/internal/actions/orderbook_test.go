package actions

import (
	"database/sql"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stretchr/testify/assert"
	"math"
	"net/http/httptest"
	"strconv"
	"testing"

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
					{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					{
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
						{
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
					{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					{
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
						{
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
					{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					{
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
						{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
						{
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
					{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					{
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
						{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
						{
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
					{
						PriceR: protocol.Price{N: 1, D: 2},
						Price:  "0.5",
						Amount: "123",
					},
					{
						PriceR: protocol.Price{N: 1, D: 1},
						Price:  "1.0",
						Amount: "123",
					},
				},
				Bids: []protocol.PriceLevel{
					{
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
						{
							PriceR: protocol.Price{N: 1, D: 3},
							Price:  "0.33",
							Amount: "13",
						},
					},
					Asks: []protocol.PriceLevel{
						{
							PriceR: protocol.Price{N: 1, D: 2},
							Price:  "0.5",
							Amount: "123",
						},
						{
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
	handler := GetOrderbookHandler{}

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
			r := makeRequest(t, testCase.queryParams, map[string]string{}, nil)
			w := httptest.NewRecorder()
			_, err := handler.GetResource(w, r)
			if err == nil || err.Error() != invalidOrderBook.Error() {
				t.Fatalf("expected error %v but got %v", invalidOrderBook, err)
			}
			if lastLedger := w.Header().Get(LastLedgerHeaderName); lastLedger != "" {
				t.Fatalf("expected last ledger to be not set but got %v", lastLedger)
			}
		})
	}
}

func TestOrderbookGetResource(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}

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

	sellEurOffer := twoEurOffer
	sellEurOffer.Buying, sellEurOffer.Selling = sellEurOffer.Selling, sellEurOffer.Buying
	sellEurOffer.OfferId = 15

	otherEurOffer := twoEurOffer
	otherEurOffer.Amount = xdr.Int64(math.MaxInt64)
	otherEurOffer.OfferId = 16

	nonCanonicalPriceTwoEurOffer := twoEurOffer
	nonCanonicalPriceTwoEurOffer.OfferId = 30
	// Add a separate offer with the same price value, but
	// using a non-canonical representation, to make sure
	// they are coalesced into the same price level
	nonCanonicalPriceTwoEurOffer.Price.N *= 15
	nonCanonicalPriceTwoEurOffer.Price.D *= 15

	threeEurOffer := twoEurOffer
	threeEurOffer.Price.N = 3
	threeEurOffer.OfferId = 20

	sellEurOffer.Price.N = 9
	sellEurOffer.Price.D = 10

	otherSellEurOffer := sellEurOffer
	otherSellEurOffer.OfferId = 17
	// sellEurOffer.Price * 2
	otherSellEurOffer.Price.N = 9
	otherSellEurOffer.Price.D = 5

	offers := []xdr.OfferEntry{
		twoEurOffer,
		otherEurOffer,
		nonCanonicalPriceTwoEurOffer,
		threeEurOffer,
		sellEurOffer,
		otherSellEurOffer,
	}

	assert.NoError(t, q.TruncateTables([]string{"offers"}))

	batch := q.NewOffersBatchInsertBuilder(0)
	for i, offer := range offers {
		assert.NoError(t, batch.Add(offer, xdr.Uint32(i+1)))
	}
	assert.NoError(t, batch.Exec())

	assert.NoError(t, q.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}))
	defer q.Rollback()

	fullResponse := empty
	fullResponse.Asks = []protocol.PriceLevel{
		{
			PriceR: protocol.Price{N: int32(twoEurOffer.Price.N), D: int32(twoEurOffer.Price.D)},
			Price:  "2.0000000",
			Amount: "922337203685.4776807",
		},
		{
			PriceR: protocol.Price{N: int32(threeEurOffer.Price.N), D: int32(threeEurOffer.Price.D)},
			Price:  "3.0000000",
			Amount: "0.0000500",
		},
	}
	fullResponse.Bids = []protocol.PriceLevel{
		{
			PriceR: protocol.Price{N: int32(sellEurOffer.Price.D), D: int32(sellEurOffer.Price.N)},
			Price:  "1.1111111",
			Amount: "0.0000500",
		},
		{
			PriceR: protocol.Price{N: int32(otherSellEurOffer.Price.D), D: int32(otherSellEurOffer.Price.N)},
			Price:  "0.5555556",
			Amount: "0.0000500",
		},
	}

	limitResponse := empty
	limitResponse.Asks = []protocol.PriceLevel{
		{
			PriceR: protocol.Price{N: int32(twoEurOffer.Price.N), D: int32(twoEurOffer.Price.D)},
			Price:  "2.0000000",
			Amount: "922337203685.4776807",
		},
	}
	limitResponse.Bids = []protocol.PriceLevel{
		{
			PriceR: protocol.Price{N: int32(sellEurOffer.Price.D), D: int32(sellEurOffer.Price.N)},
			Price:  "1.1111111",
			Amount: "0.0000500",
		},
	}

	for _, testCase := range []struct {
		name     string
		limit    int
		expected OrderBookResponse
	}{

		{
			"full orderbook",
			10,
			fullResponse,
		},
		{
			"limit request",
			1,
			limitResponse,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler := GetOrderbookHandler{}
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
				q.Session,
			)
			w := httptest.NewRecorder()
			response, err := handler.GetResource(w, r)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !response.Equals(testCase.expected) {
				t.Fatalf("expected %v but got %v", testCase.expected, response)
			}
		})
	}
}
