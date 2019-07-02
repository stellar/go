package main

import (
	"bytes"
	"encoding"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
)

var (
	issuer, _ = xdr.NewAccountId(xdr.PublicKeyTypePublicKeyTypeEd25519, xdr.Uint256{})

	nativeAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeNative,
	}

	usdAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AssetAlphaNum4{
			AssetCode: [4]byte{'u', 's', 'd', 0},
			Issuer:    issuer,
		},
	}

	eurAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AssetAlphaNum4{
			AssetCode: [4]byte{'e', 'u', 'r', 0},
			Issuer:    issuer,
		},
	}

	chfAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AssetAlphaNum4{
			AssetCode: [4]byte{'c', 'h', 'f', 0},
			Issuer:    issuer,
		},
	}

	yenAsset = xdr.Asset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AssetAlphaNum4{
			AssetCode: [4]byte{'y', 'e', 'n', 0},
			Issuer:    issuer,
		},
	}

	fiftyCentsOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(1),
		Buying:   usdAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 1,
			D: 2,
		},
		Amount: xdr.Int64(500),
	}
	quarterOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(2),
		Buying:   usdAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 1,
			D: 4,
		},
		Amount: xdr.Int64(500),
	}
	dollarOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(3),
		Buying:   usdAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}

	eurOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(4),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}
	twoEurOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(5),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 2,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}
	threeEurOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(6),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 3,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}
)

func assertBinaryMarshalerEquals(t *testing.T, a, b encoding.BinaryMarshaler) {
	serializedA, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("could not marshal %v", a)
	}
	serializedB, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("could not marshal %v", b)
	}

	if !bytes.Equal(serializedA, serializedB) {
		t.Fatalf("expected lists to be equal got %v %v", a, b)
	}
}

func assertOfferListEquals(t *testing.T, a, b []xdr.OfferEntry) {
	if len(a) != len(b) {
		t.Fatalf("expected lists to have same length but got %v %v", a, b)
	}

	for i := 0; i < len(a); i++ {
		assertBinaryMarshalerEquals(t, a[i], b[i])
	}
}

func assertGraphEquals(t *testing.T, a, b *OrderBookGraph) {
	if len(a.edgesForSellingAsset) != len(b.edgesForSellingAsset) {
		t.Fatalf("expected edges to have same length but got %v %v", a, b)
	}
	if len(a.tradingPairForOffer) != len(b.tradingPairForOffer) {
		t.Fatalf("expected trading pairs to have same length but got %v %v", a, b)
	}

	for sellingAsset, edgeSet := range a.edgesForSellingAsset {
		otherEdgeSet := b.edgesForSellingAsset[sellingAsset]
		if len(edgeSet) != len(otherEdgeSet) {
			t.Fatalf(
				"expected edge set for %v to have same length but got %v %v",
				sellingAsset,
				edgeSet,
				otherEdgeSet,
			)
		}
		for buyingAsset, offers := range edgeSet {
			otherOffers := otherEdgeSet[buyingAsset]

			if len(offers) != len(otherOffers) {
				t.Fatalf(
					"expected offers for %v to have same length but got %v %v",
					buyingAsset,
					offers,
					otherOffers,
				)
			}

			assertOfferListEquals(t, offers, otherOffers)
		}
	}

	for offerID, pair := range a.tradingPairForOffer {
		otherPair := b.tradingPairForOffer[offerID]
		if pair.buyingAsset != otherPair.buyingAsset {
			t.Fatalf(
				"expected trading pair to match but got %v %v",
				pair,
				otherPair,
			)
		}
		if pair.sellingAsset != otherPair.sellingAsset {
			t.Fatalf(
				"expected trading pair to match but got %v %v",
				pair,
				otherPair,
			)
		}
	}
}

func assertPathEquals(t *testing.T, a, b []Path) {
	if len(a) != len(b) {
		t.Fatalf("expected paths to have same length but got %v %v", a, b)
	}

	for i := 0; i < len(a); i++ {
		if a[i].SourceAmount != b[i].SourceAmount {
			t.Fatalf("expected paths to be same got %v %v", a, b)
		}
		if a[i].DestinationAmount != b[i].DestinationAmount {
			t.Fatalf("expected paths to be same got %v %v", a, b)
		}
		if !a[i].DestinationAsset.Equals(b[i].DestinationAsset) {
			t.Fatalf("expected paths to be same got %v %v", a, b)
		}
		if !a[i].SourceAsset.Equals(b[i].SourceAsset) {
			t.Fatalf("expected paths to be same got %v %v", a, b)
		}

		if len(a[i].InteriorNodes) != len(b[i].InteriorNodes) {
			t.Fatalf("expected paths to be same got %v %v", a, b)
		}

		for j := 0; j > len(a[i].InteriorNodes); j++ {
			if !a[i].InteriorNodes[j].Equals(b[i].InteriorNodes[j]) {
				t.Fatalf("expected paths to be same got %v %v", a, b)
			}
		}
	}
}

func TestAddEdgeSet(t *testing.T) {
	set := edgeSet{}

	set.add(dollarOffer)
	set.add(eurOffer)
	set.add(twoEurOffer)
	set.add(threeEurOffer)
	set.add(quarterOffer)
	set.add(fiftyCentsOffer)

	if len(set) != 2 {
		t.Fatalf("expected set to have 2 entries but got %v", set)
	}

	assertOfferListEquals(t, set[usdAsset.String()], []xdr.OfferEntry{
		quarterOffer,
		fiftyCentsOffer,
		dollarOffer,
	})

	assertOfferListEquals(t, set[eurAsset.String()], []xdr.OfferEntry{
		eurOffer,
		twoEurOffer,
		threeEurOffer,
	})
}

func TestRemoveEdgeSet(t *testing.T) {
	set := edgeSet{}

	if contains := set.remove(dollarOffer.OfferId, usdAsset.String()); contains {
		t.Fatal("expected set to not contain asset")
	}

	set.add(dollarOffer)
	set.add(eurOffer)
	set.add(twoEurOffer)
	set.add(threeEurOffer)
	set.add(quarterOffer)
	set.add(fiftyCentsOffer)

	if contains := set.remove(dollarOffer.OfferId, usdAsset.String()); !contains {
		t.Fatal("expected set to contain dollar offer")
	}

	if contains := set.remove(dollarOffer.OfferId, usdAsset.String()); contains {
		t.Fatal("expected set to not contain dollar offer after it has been deleted")
	}

	if contains := set.remove(threeEurOffer.OfferId, eurAsset.String()); !contains {
		t.Fatal("expected set to contain three euro offer")
	}
	if contains := set.remove(eurOffer.OfferId, eurAsset.String()); !contains {
		t.Fatal("expected set to contain euro offer")
	}
	if contains := set.remove(twoEurOffer.OfferId, eurAsset.String()); !contains {
		t.Fatal("expected set to contain two euro offer")
	}

	if contains := set.remove(eurOffer.OfferId, eurAsset.String()); contains {
		t.Fatal("expected set to not contain euro offer after it has been deleted")
	}

	if len(set) != 1 {
		t.Fatalf("expected set to have 1 entry but got %v", set)
	}

	assertOfferListEquals(t, set[usdAsset.String()], []xdr.OfferEntry{
		quarterOffer,
		fiftyCentsOffer,
	})
}

func TestApplyTwice(t *testing.T) {
	graph := NewOrderBookGraph()

	batch := graph.Batch()
	err := batch.
		AddOffer(dollarOffer).
		AddOffer(threeEurOffer).
		AddOffer(eurOffer).
		AddOffer(twoEurOffer).
		AddOffer(quarterOffer).
		AddOffer(fiftyCentsOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	err = batch.Apply()
	if err != errBatchAlreadyApplied {
		t.Fatalf("expected error %v but got %v", errBatchAlreadyApplied, err)
	}
}

func TestAddOfferOrderBook(t *testing.T) {
	graph := NewOrderBookGraph()

	err := graph.Batch().
		AddOffer(dollarOffer).
		AddOffer(threeEurOffer).
		AddOffer(eurOffer).
		AddOffer(twoEurOffer).
		AddOffer(quarterOffer).
		AddOffer(fiftyCentsOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	eurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(9),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}
	otherEurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(10),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 2,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}

	usdEurOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(11),
		Buying:   usdAsset,
		Selling:  eurAsset,
		Price: xdr.Price{
			N: 1,
			D: 3,
		},
		Amount: xdr.Int64(500),
	}

	err = graph.Batch().
		AddOffer(eurUsdOffer).
		AddOffer(otherEurUsdOffer).
		AddOffer(usdEurOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	expectedGraph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{
			nativeAsset.String(): edgeSet{
				usdAsset.String(): []xdr.OfferEntry{
					quarterOffer,
					fiftyCentsOffer,
					dollarOffer,
				},
				eurAsset.String(): []xdr.OfferEntry{
					eurOffer,
					twoEurOffer,
					threeEurOffer,
				},
			},
			usdAsset.String(): edgeSet{
				eurAsset.String(): []xdr.OfferEntry{
					eurUsdOffer,
					otherEurUsdOffer,
				},
			},
			eurAsset.String(): edgeSet{
				usdAsset.String(): []xdr.OfferEntry{
					usdEurOffer,
				},
			},
		},
		tradingPairForOffer: map[xdr.Int64]tradingPair{
			quarterOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			fiftyCentsOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			dollarOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			twoEurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			threeEurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurUsdOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			otherEurUsdOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			usdEurOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: eurAsset.String(),
			},
		},
	}

	// adding the same orders multiple times should have no effect
	err = graph.Batch().
		AddOffer(otherEurUsdOffer).
		AddOffer(usdEurOffer).
		AddOffer(dollarOffer).
		AddOffer(threeEurOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	assertGraphEquals(t, graph, expectedGraph)
}

func TestUpdateOfferOrderBook(t *testing.T) {
	graph := NewOrderBookGraph()

	err := graph.Batch().
		AddOffer(dollarOffer).
		AddOffer(threeEurOffer).
		AddOffer(eurOffer).
		AddOffer(twoEurOffer).
		AddOffer(quarterOffer).
		AddOffer(fiftyCentsOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	eurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(9),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}
	otherEurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(10),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 2,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}

	usdEurOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(11),
		Buying:   usdAsset,
		Selling:  eurAsset,
		Price: xdr.Price{
			N: 1,
			D: 3,
		},
		Amount: xdr.Int64(500),
	}

	err = graph.Batch().
		AddOffer(eurUsdOffer).
		AddOffer(otherEurUsdOffer).
		AddOffer(usdEurOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	usdEurOffer.Price.N = 4
	usdEurOffer.Price.D = 1

	otherEurUsdOffer.Price.N = 1
	otherEurUsdOffer.Price.D = 2

	dollarOffer.Amount = 12

	err = graph.Batch().
		AddOffer(usdEurOffer).
		AddOffer(otherEurUsdOffer).
		AddOffer(dollarOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	expectedGraph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{
			nativeAsset.String(): edgeSet{
				usdAsset.String(): []xdr.OfferEntry{
					quarterOffer,
					fiftyCentsOffer,
					dollarOffer,
				},
				eurAsset.String(): []xdr.OfferEntry{
					eurOffer,
					twoEurOffer,
					threeEurOffer,
				},
			},
			usdAsset.String(): edgeSet{
				eurAsset.String(): []xdr.OfferEntry{
					otherEurUsdOffer,
					eurUsdOffer,
				},
			},
			eurAsset.String(): edgeSet{
				usdAsset.String(): []xdr.OfferEntry{
					usdEurOffer,
				},
			},
		},
		tradingPairForOffer: map[xdr.Int64]tradingPair{
			quarterOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			fiftyCentsOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			dollarOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			twoEurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			threeEurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurUsdOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			otherEurUsdOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			usdEurOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: eurAsset.String(),
			},
		},
	}

	assertGraphEquals(t, graph, expectedGraph)
}

func TestRemoveOfferOrderBook(t *testing.T) {
	graph := NewOrderBookGraph()

	err := graph.Batch().
		AddOffer(dollarOffer).
		AddOffer(threeEurOffer).
		AddOffer(eurOffer).
		AddOffer(twoEurOffer).
		AddOffer(quarterOffer).
		AddOffer(fiftyCentsOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	eurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(9),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}
	otherEurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(10),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 2,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}

	usdEurOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(11),
		Buying:   usdAsset,
		Selling:  eurAsset,
		Price: xdr.Price{
			N: 1,
			D: 3,
		},
		Amount: xdr.Int64(500),
	}

	err = graph.Batch().
		AddOffer(eurUsdOffer).
		AddOffer(otherEurUsdOffer).
		AddOffer(usdEurOffer).
		RemoveOffer(usdEurOffer.OfferId).
		RemoveOffer(otherEurUsdOffer.OfferId).
		RemoveOffer(dollarOffer.OfferId).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	expectedGraph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{
			nativeAsset.String(): edgeSet{
				usdAsset.String(): []xdr.OfferEntry{
					quarterOffer,
					fiftyCentsOffer,
				},
				eurAsset.String(): []xdr.OfferEntry{
					eurOffer,
					twoEurOffer,
					threeEurOffer,
				},
			},
			usdAsset.String(): edgeSet{
				eurAsset.String(): []xdr.OfferEntry{
					eurUsdOffer,
				},
			},
		},
		tradingPairForOffer: map[xdr.Int64]tradingPair{
			quarterOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			fiftyCentsOffer.OfferId: tradingPair{
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			twoEurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			threeEurOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurUsdOffer.OfferId: tradingPair{
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
		},
	}

	assertGraphEquals(t, graph, expectedGraph)
}

func TestConsumeOffers(t *testing.T) {
	var key xdr.Uint256
	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	copy(key[:], kp.Address())
	ignoreOffersFrom, err := xdr.NewAccountId(xdr.PublicKeyTypePublicKeyTypeEd25519, key)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	otherSellerTwoEurOffer := twoEurOffer
	otherSellerTwoEurOffer.SellerId = ignoreOffersFrom

	denominatorZeroOffer := twoEurOffer
	denominatorZeroOffer.Price.D = 0

	for _, testCase := range []struct {
		name               string
		offers             []xdr.OfferEntry
		ignoreOffersFrom   xdr.AccountId
		currentAssetAmount xdr.Int64
		result             xdr.Int64
		err                error
	}{
		{
			"offers must not be empty",
			[]xdr.OfferEntry{},
			issuer,
			100,
			0,
			errEmptyOffers,
		},
		{
			"currentAssetAmount must be positive",
			[]xdr.OfferEntry{eurOffer},
			ignoreOffersFrom,
			0,
			0,
			errAssetAmountIsZero,
		},
		{
			"ignore all offers",
			[]xdr.OfferEntry{eurOffer},
			issuer,
			1,
			-1,
			nil,
		},
		{
			"offer denominator cannot be zero",
			[]xdr.OfferEntry{denominatorZeroOffer},
			ignoreOffersFrom,
			10000,
			0,
			errOfferPriceDenominatorIsZero,
		},
		{
			"ignore some offers",
			[]xdr.OfferEntry{eurOffer, otherSellerTwoEurOffer},
			issuer,
			100,
			200,
			nil,
		},
		{
			"not enough offers to consume",
			[]xdr.OfferEntry{eurOffer, twoEurOffer},
			ignoreOffersFrom,
			1001,
			-1,
			nil,
		},
		{
			"consume all offers",
			[]xdr.OfferEntry{eurOffer, twoEurOffer, threeEurOffer},
			ignoreOffersFrom,
			1500,
			3000,
			nil,
		},
		{
			"consume offer partially",
			[]xdr.OfferEntry{eurOffer, twoEurOffer},
			ignoreOffersFrom,
			2,
			2,
			nil,
		},
		{
			"round up",
			[]xdr.OfferEntry{quarterOffer},
			ignoreOffersFrom,
			5,
			2,
			nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := consumeOffers(
				testCase.offers,
				testCase.ignoreOffersFrom,
				testCase.currentAssetAmount,
			)
			if err != testCase.err {
				t.Fatalf("expected error %v but got %v", testCase.err, err)
			}
			if err == nil {
				if result != testCase.result {
					t.Fatalf("expected %v but got %v", testCase.result, result)
				}
			}
		})
	}

}

func TestSortAndFilterPaths(t *testing.T) {
	allPaths := []Path{
		Path{
			SourceAmount:      3,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes:     []xdr.Asset{},
		},
		Path{
			SourceAmount:      4,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes:     []xdr.Asset{},
		},
		Path{
			SourceAmount:      1,
			SourceAsset:       usdAsset,
			sourceAssetString: usdAsset.String(),
			InteriorNodes:     []xdr.Asset{},
		},
		Path{
			SourceAmount:      2,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes:     []xdr.Asset{},
		},
		Path{
			SourceAmount:      2,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes: []xdr.Asset{
				nativeAsset,
			},
		},
		Path{
			SourceAmount:      10,
			SourceAsset:       nativeAsset,
			sourceAssetString: nativeAsset.String(),
			InteriorNodes:     []xdr.Asset{},
		},
	}
	sortedAndFiltered := sortAndFilterPaths(
		allPaths,
		3,
		yenAsset,
		1000,
	)
	expectedPaths := []Path{
		Path{
			SourceAmount:      2,
			SourceAsset:       eurAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		Path{
			SourceAmount: 2,
			SourceAsset:  eurAsset,
			InteriorNodes: []xdr.Asset{
				nativeAsset,
			},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		Path{
			SourceAmount:      3,
			SourceAsset:       eurAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		Path{
			SourceAmount:      1,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		Path{
			SourceAmount:      10,
			SourceAsset:       nativeAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
	}

	assertPathEquals(t, sortedAndFiltered, expectedPaths)
}

func TestFindPaths(t *testing.T) {
	graph := NewOrderBookGraph()

	err := graph.Batch().
		AddOffer(dollarOffer).
		AddOffer(threeEurOffer).
		AddOffer(eurOffer).
		AddOffer(twoEurOffer).
		AddOffer(quarterOffer).
		AddOffer(fiftyCentsOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	eurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(9),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}
	otherEurUsdOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(10),
		Buying:   eurAsset,
		Selling:  usdAsset,
		Price: xdr.Price{
			N: 2,
			D: 1,
		},
		Amount: xdr.Int64(500),
	}

	usdEurOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(11),
		Buying:   usdAsset,
		Selling:  eurAsset,
		Price: xdr.Price{
			N: 1,
			D: 3,
		},
		Amount: xdr.Int64(500),
	}

	chfEurOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(12),
		Buying:   chfAsset,
		Selling:  eurAsset,
		Price: xdr.Price{
			N: 1,
			D: 2,
		},
		Amount: xdr.Int64(500),
	}

	yenChfOffer := xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(13),
		Buying:   yenAsset,
		Selling:  chfAsset,
		Price: xdr.Price{
			N: 1,
			D: 2,
		},
		Amount: xdr.Int64(500),
	}

	err = graph.Batch().
		AddOffer(eurUsdOffer).
		AddOffer(otherEurUsdOffer).
		AddOffer(usdEurOffer).
		AddOffer(chfEurOffer).
		AddOffer(yenChfOffer).
		Apply()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var key xdr.Uint256
	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	copy(key[:], kp.Address())
	ignoreOffersFrom, err := xdr.NewAccountId(xdr.PublicKeyTypePublicKeyTypeEd25519, key)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	paths, err := graph.FindPaths(
		3,
		nativeAsset,
		20,
		ignoreOffersFrom,
		[]xdr.Asset{
			yenAsset,
			usdAsset,
		},
		[]xdr.Int64{
			100000,
			60000,
		},
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	expectedPaths := []Path{
		Path{
			SourceAmount:      5,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		Path{
			SourceAmount: 7,
			SourceAsset:  usdAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		Path{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
				chfAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
	}

	assertPathEquals(t, paths, expectedPaths)

	paths, err = graph.FindPaths(
		4,
		nativeAsset,
		20,
		ignoreOffersFrom,
		[]xdr.Asset{
			yenAsset,
			usdAsset,
		},
		[]xdr.Int64{
			100000,
			60000,
		},
		5,
	)

	expectedPaths = []Path{
		Path{
			SourceAmount:      5,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		Path{
			SourceAmount: 7,
			SourceAsset:  usdAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		Path{
			SourceAmount: 2,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				usdAsset,
				eurAsset,
				chfAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		Path{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
				chfAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
	}

	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	assertPathEquals(t, paths, expectedPaths)
}
