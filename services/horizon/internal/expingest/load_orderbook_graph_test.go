package expingest

// import (
// 	"sort"
// 	"testing"

// 	"github.com/stellar/go/exp/orderbook"
// 	"github.com/stellar/go/services/horizon/internal/db2/history"
// 	"github.com/stellar/go/xdr"
// 	"github.com/stretchr/testify/suite"
// )

// var (
// 	issuer   = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
// 	usdAsset = xdr.Asset{
// 		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
// 		AlphaNum4: &xdr.AssetAlphaNum4{
// 			AssetCode: [4]byte{'u', 's', 'd', 0},
// 			Issuer:    issuer,
// 		},
// 	}

// 	nativeAsset = xdr.Asset{
// 		Type: xdr.AssetTypeAssetTypeNative,
// 	}

// 	eurAsset = xdr.Asset{
// 		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
// 		AlphaNum4: &xdr.AssetAlphaNum4{
// 			AssetCode: [4]byte{'e', 'u', 'r', 0},
// 			Issuer:    issuer,
// 		},
// 	}
// 	eurOffer = xdr.OfferEntry{
// 		SellerId: issuer,
// 		OfferId:  xdr.Int64(4),
// 		Buying:   eurAsset,
// 		Selling:  nativeAsset,
// 		Price: xdr.Price{
// 			N: 1,
// 			D: 1,
// 		},
// 		Flags:  1,
// 		Amount: xdr.Int64(500),
// 	}
// 	twoEurOffer = xdr.OfferEntry{
// 		SellerId: issuer,
// 		OfferId:  xdr.Int64(5),
// 		Buying:   eurAsset,
// 		Selling:  nativeAsset,
// 		Price: xdr.Price{
// 			N: 2,
// 			D: 1,
// 		},
// 		Flags:  2,
// 		Amount: xdr.Int64(500),
// 	}
// )

// type LoadOffersIntoMemoryTestSuite struct {
// 	suite.Suite
// 	graph *orderbook.OrderBookGraph
// 	// session  *mockDBSession
// 	// historyQ *mockDBQ
// 	system *System
// }

// func (s *LoadOffersIntoMemoryTestSuite) SetupTest() {
// 	s.graph = orderbook.NewOrderBookGraph()
// 	// s.session = &mockDBSession{}
// 	// s.historyQ = &mockDBQ{}
// 	s.system = &System{
// 		historySession: s.session,
// 		// historyQ:       s.historyQ,
// 		graph: s.graph,
// 	}

// 	s.Assert().Equal(uint32(1), s.system.state.latestSuccessfullyProcessedLedger)

// 	// s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// }

// func (s *LoadOffersIntoMemoryTestSuite) TearDownTest() {
// 	t := s.T()
// 	// s.session.AssertExpectations(t)
// 	// s.historyQ.AssertExpectations(t)
// }

// func (s *LoadOffersIntoMemoryTestSuite) TestLoadOrderBookGraphFromEmptyDB() {
// 	// s.historyQ.On("GetAllOffers").Return([]history.Offer{}, nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(resumeState, nextState.systemState)
// 	s.Assert().Equal(uint32(1), nextState.latestSuccessfullyProcessedLedger)
// 	s.Assert().True(s.graph.IsEmpty())
// }

// func (s *LoadOffersIntoMemoryTestSuite) TestLoadOrderBookGraph() {
// 	s.historyQ.On("GetAllOffers").Return([]history.Offer{
// 		history.Offer{
// 			OfferID:      eurOffer.OfferId,
// 			SellerID:     eurOffer.SellerId.Address(),
// 			SellingAsset: eurOffer.Selling,
// 			BuyingAsset:  eurOffer.Buying,
// 			Amount:       eurOffer.Amount,
// 			Pricen:       int32(eurOffer.Price.N),
// 			Priced:       int32(eurOffer.Price.D),
// 			Price:        float64(eurOffer.Price.N) / float64(eurOffer.Price.D),
// 			Flags:        uint32(eurOffer.Flags),
// 		},
// 		history.Offer{
// 			OfferID:      twoEurOffer.OfferId,
// 			SellerID:     twoEurOffer.SellerId.Address(),
// 			SellingAsset: twoEurOffer.Selling,
// 			BuyingAsset:  twoEurOffer.Buying,
// 			Amount:       twoEurOffer.Amount,
// 			Pricen:       int32(twoEurOffer.Price.N),
// 			Priced:       int32(twoEurOffer.Price.D),
// 			Price:        float64(twoEurOffer.Price.N) / float64(twoEurOffer.Price.D),
// 			Flags:        uint32(twoEurOffer.Flags),
// 		},
// 	}, nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(resumeState, nextState.systemState)
// 	s.Assert().Equal(uint32(1), nextState.latestSuccessfullyProcessedLedger)
// 	s.Assert().False(s.graph.IsEmpty())
// 	offers := s.graph.Offers()
// 	sort.Slice(offers, func(i, j int) bool {
// 		return offers[i].OfferId < offers[j].OfferId
// 	})
// 	expectedOffers := []xdr.OfferEntry{
// 		eurOffer, twoEurOffer,
// 	}
// 	s.Assert().Equal(expectedOffers, offers)
// }

// func TestLoadOffersIntoMemoryTestSuite(t *testing.T) {
// 	suite.Run(t, new(LoadOffersIntoMemoryTestSuite))
// }
