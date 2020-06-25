package orderbook

import (
	"bytes"
	"encoding"
	"math"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/price"
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

	set.add(dollarOffer.Buying.String(), dollarOffer)
	set.add(eurOffer.Buying.String(), eurOffer)
	set.add(twoEurOffer.Buying.String(), twoEurOffer)
	set.add(threeEurOffer.Buying.String(), threeEurOffer)
	set.add(quarterOffer.Buying.String(), quarterOffer)
	set.add(fiftyCentsOffer.Buying.String(), fiftyCentsOffer)

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

	set.add(dollarOffer.Buying.String(), dollarOffer)
	set.add(eurOffer.Buying.String(), eurOffer)
	set.add(twoEurOffer.Buying.String(), twoEurOffer)
	set.add(threeEurOffer.Buying.String(), threeEurOffer)
	set.add(quarterOffer.Buying.String(), quarterOffer)
	set.add(fiftyCentsOffer.Buying.String(), fiftyCentsOffer)

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

func TestApplyOutdatedLedger(t *testing.T) {
	graph := NewOrderBookGraph()
	if graph.lastLedger != 0 {
		t.Fatalf("expected last ledger to be %v but got %v", 0, graph.lastLedger)
	}

	graph.AddOffer(fiftyCentsOffer)
	err := graph.Apply(2)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, graph.lastLedger)
	}

	graph.AddOffer(eurOffer)
	err = graph.Apply(1)
	if err != errUnexpectedLedger {
		t.Fatalf("expected error %v but got %v", errUnexpectedLedger, err)
	}
	if graph.lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, graph.lastLedger)
	}

	graph.Discard()

	graph.AddOffer(eurOffer)
	err = graph.Apply(2)
	if err != errUnexpectedLedger {
		t.Fatalf("expected error %v but got %v", errUnexpectedLedger, err)
	}
	if graph.lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, graph.lastLedger)
	}

	graph.Discard()

	err = graph.Apply(4)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 4 {
		t.Fatalf("expected last ledger to be %v but got %v", 4, graph.lastLedger)
	}
}

func TestAddOfferOrderBook(t *testing.T) {
	graph := NewOrderBookGraph()

	graph.AddOffer(dollarOffer)
	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	graph.AddOffer(quarterOffer)
	graph.AddOffer(fiftyCentsOffer)
	err := graph.Apply(1)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 1 {
		t.Fatalf("expected last ledger to be %v but got %v", 1, graph.lastLedger)
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

	graph.AddOffer(eurUsdOffer)
	graph.AddOffer(otherEurUsdOffer)
	graph.AddOffer(usdEurOffer)
	err = graph.Apply(2)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, graph.lastLedger)
	}

	expectedGraph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{
			nativeAsset.String(): {
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
			usdAsset.String(): {
				eurAsset.String(): []xdr.OfferEntry{
					eurUsdOffer,
					otherEurUsdOffer,
				},
			},
			eurAsset.String(): {
				usdAsset.String(): []xdr.OfferEntry{
					usdEurOffer,
				},
			},
		},
		tradingPairForOffer: map[xdr.Int64]tradingPair{
			quarterOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			fiftyCentsOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			dollarOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			twoEurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			threeEurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurUsdOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			otherEurUsdOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			usdEurOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: eurAsset.String(),
			},
		},
	}

	// adding the same orders multiple times should have no effect
	graph.AddOffer(otherEurUsdOffer)
	graph.AddOffer(usdEurOffer)
	graph.AddOffer(dollarOffer)
	graph.AddOffer(threeEurOffer)
	err = graph.Apply(3)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 3 {
		t.Fatalf("expected last ledger to be %v but got %v", 3, graph.lastLedger)
	}

	assertGraphEquals(t, graph, expectedGraph)
}

func TestUpdateOfferOrderBook(t *testing.T) {
	graph := NewOrderBookGraph()

	if !graph.IsEmpty() {
		t.Fatal("expected graph to be empty")
	}

	graph.AddOffer(dollarOffer)
	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	graph.AddOffer(quarterOffer)
	graph.AddOffer(fiftyCentsOffer)
	err := graph.Apply(1)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 1 {
		t.Fatalf("expected last ledger to be %v but got %v", 1, graph.lastLedger)
	}

	if graph.IsEmpty() {
		t.Fatal("expected graph to not be empty")
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

	graph.AddOffer(eurUsdOffer)
	graph.AddOffer(otherEurUsdOffer)
	graph.AddOffer(usdEurOffer)
	err = graph.Apply(2)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, graph.lastLedger)
	}

	usdEurOffer.Price.N = 4
	usdEurOffer.Price.D = 1

	otherEurUsdOffer.Price.N = 1
	otherEurUsdOffer.Price.D = 2

	dollarOffer.Amount = 12

	graph.AddOffer(usdEurOffer)
	graph.AddOffer(otherEurUsdOffer)
	graph.AddOffer(dollarOffer)
	err = graph.Apply(3)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 3 {
		t.Fatalf("expected last ledger to be %v but got %v", 3, graph.lastLedger)
	}

	expectedGraph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{
			nativeAsset.String(): {
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
			usdAsset.String(): {
				eurAsset.String(): []xdr.OfferEntry{
					otherEurUsdOffer,
					eurUsdOffer,
				},
			},
			eurAsset.String(): {
				usdAsset.String(): []xdr.OfferEntry{
					usdEurOffer,
				},
			},
		},
		tradingPairForOffer: map[xdr.Int64]tradingPair{
			quarterOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			fiftyCentsOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			dollarOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			twoEurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			threeEurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurUsdOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			otherEurUsdOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
			usdEurOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: eurAsset.String(),
			},
		},
	}

	assertGraphEquals(t, graph, expectedGraph)
}

func TestDiscard(t *testing.T) {
	graph := NewOrderBookGraph()

	graph.AddOffer(dollarOffer)
	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	graph.AddOffer(quarterOffer)
	graph.AddOffer(fiftyCentsOffer)
	graph.Discard()
	if graph.lastLedger != 0 {
		t.Fatalf("expected last ledger to be %v but got %v", 0, graph.lastLedger)
	}

	if err := graph.Apply(1); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if !graph.IsEmpty() {
		t.Fatal("expected graph to be empty")
	}
	if graph.lastLedger != 1 {
		t.Fatalf("expected last ledger to be %v but got %v", 1, graph.lastLedger)
	}

	graph.AddOffer(dollarOffer)
	err := graph.Apply(2)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.IsEmpty() {
		t.Fatal("expected graph to be not empty")
	}
	if graph.lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, graph.lastLedger)
	}

	expectedOffers := []xdr.OfferEntry{dollarOffer}
	assertOfferListEquals(t, graph.Offers(), expectedOffers)

	graph.AddOffer(threeEurOffer)
	graph.Discard()
	assertOfferListEquals(t, graph.Offers(), expectedOffers)
}

func TestRemoveOfferOrderBook(t *testing.T) {
	graph := NewOrderBookGraph()

	graph.AddOffer(dollarOffer)
	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	graph.AddOffer(quarterOffer)
	graph.AddOffer(fiftyCentsOffer)
	err := graph.Apply(1)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 1 {
		t.Fatalf("expected last ledger to be %v but got %v", 1, graph.lastLedger)
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

	graph.AddOffer(eurUsdOffer)
	graph.AddOffer(otherEurUsdOffer)
	graph.AddOffer(usdEurOffer)
	graph.RemoveOffer(usdEurOffer.OfferId)
	graph.RemoveOffer(otherEurUsdOffer.OfferId)
	graph.RemoveOffer(dollarOffer.OfferId)
	err = graph.Apply(2)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, graph.lastLedger)
	}

	expectedGraph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{
			nativeAsset.String(): {
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
			usdAsset.String(): {
				eurAsset.String(): []xdr.OfferEntry{
					eurUsdOffer,
				},
			},
		},
		tradingPairForOffer: map[xdr.Int64]tradingPair{
			quarterOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			fiftyCentsOffer.OfferId: {
				buyingAsset:  usdAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			twoEurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			threeEurOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: nativeAsset.String(),
			},
			eurUsdOffer.OfferId: {
				buyingAsset:  eurAsset.String(),
				sellingAsset: usdAsset.String(),
			},
		},
	}

	assertGraphEquals(t, graph, expectedGraph)

	err = graph.
		RemoveOffer(quarterOffer.OfferId).
		RemoveOffer(fiftyCentsOffer.OfferId).
		RemoveOffer(eurOffer.OfferId).
		RemoveOffer(twoEurOffer.OfferId).
		RemoveOffer(threeEurOffer.OfferId).
		RemoveOffer(eurUsdOffer.OfferId).
		Apply(3)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 3 {
		t.Fatalf("expected last ledger to be %v but got %v", 3, graph.lastLedger)
	}

	// Skip over offer ids which are not present in the graph
	err = graph.
		RemoveOffer(988888).
		Apply(5)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if graph.lastLedger != 5 {
		t.Fatalf("expected last ledger to be %v but got %v", 3, graph.lastLedger)
	}

	expectedGraph.edgesForSellingAsset = map[string]edgeSet{}
	expectedGraph.tradingPairForOffer = map[xdr.Int64]tradingPair{}
	assertGraphEquals(t, graph, expectedGraph)

	if !graph.IsEmpty() {
		t.Fatal("expected graph to be empty")
	}
}

func TestFindOffers(t *testing.T) {
	graph := NewOrderBookGraph()

	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		graph.findOffers(nativeAsset.String(), eurAsset.String(), 0),
	)

	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		graph.findOffers(nativeAsset.String(), eurAsset.String(), 5),
	)

	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	err := graph.Apply(1)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		graph.findOffers(nativeAsset.String(), eurAsset.String(), 0),
	)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{eurOffer, twoEurOffer},
		graph.findOffers(nativeAsset.String(), eurAsset.String(), 2),
	)

	extraTwoEurOffers := []xdr.OfferEntry{}
	for i := 0; i < 4; i++ {
		otherTwoEurOffer := twoEurOffer
		otherTwoEurOffer.OfferId += xdr.Int64(i + 17)
		graph.AddOffer(otherTwoEurOffer)
		extraTwoEurOffers = append(extraTwoEurOffers, otherTwoEurOffer)
	}
	if err := graph.Apply(2); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	assertOfferListEquals(
		t,
		append([]xdr.OfferEntry{eurOffer, twoEurOffer}, extraTwoEurOffers...),
		graph.findOffers(nativeAsset.String(), eurAsset.String(), 2),
	)
	assertOfferListEquals(
		t,
		append(append([]xdr.OfferEntry{eurOffer, twoEurOffer}, extraTwoEurOffers...), threeEurOffer),
		graph.findOffers(nativeAsset.String(), eurAsset.String(), 3),
	)
}

func TestFindAsksAndBids(t *testing.T) {
	graph := NewOrderBookGraph()

	asks, bids, lastLedger := graph.FindAsksAndBids(nativeAsset, eurAsset, 0)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		asks,
	)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		bids,
	)
	if lastLedger != 0 {
		t.Fatalf("expected last ledger to be %v but got %v", 0, lastLedger)
	}

	asks, bids, lastLedger = graph.FindAsksAndBids(nativeAsset, eurAsset, 5)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		asks,
	)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		bids,
	)
	if lastLedger != 0 {
		t.Fatalf("expected last ledger to be %v but got %v", 0, lastLedger)
	}

	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	err := graph.Apply(1)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	asks, bids, lastLedger = graph.FindAsksAndBids(nativeAsset, eurAsset, 0)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		asks,
	)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{},
		bids,
	)
	if lastLedger != 1 {
		t.Fatalf("expected last ledger to be %v but got %v", 1, lastLedger)
	}

	extraTwoEurOffers := []xdr.OfferEntry{}
	for i := 0; i < 4; i++ {
		otherTwoEurOffer := twoEurOffer
		// make sure de-normalized prices are dealt with properly
		otherTwoEurOffer.Price.N *= xdr.Int32(i + 1)
		otherTwoEurOffer.Price.D *= xdr.Int32(i + 1)
		otherTwoEurOffer.OfferId += xdr.Int64(i + 17)
		graph.AddOffer(otherTwoEurOffer)
		extraTwoEurOffers = append(extraTwoEurOffers, otherTwoEurOffer)
	}
	if err := graph.Apply(2); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	sellEurOffer := twoEurOffer
	sellEurOffer.Buying, sellEurOffer.Selling = sellEurOffer.Selling, sellEurOffer.Buying
	sellEurOffer.OfferId = 35
	graph.AddOffer(sellEurOffer)
	if err := graph.Apply(3); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	asks, bids, lastLedger = graph.FindAsksAndBids(nativeAsset, eurAsset, 3)
	assertOfferListEquals(
		t,
		append(append([]xdr.OfferEntry{eurOffer, twoEurOffer}, extraTwoEurOffers...), threeEurOffer),
		asks,
	)
	assertOfferListEquals(
		t,
		[]xdr.OfferEntry{sellEurOffer},
		bids,
	)
	if lastLedger != 3 {
		t.Fatalf("expected last ledger to be %v but got %v", 3, lastLedger)
	}
}

func TestConsumeOffersForSellingAsset(t *testing.T) {
	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	ignoreOffersFrom := xdr.MustAddress(kp.Address())
	otherSellerTwoEurOffer := twoEurOffer
	otherSellerTwoEurOffer.SellerId = ignoreOffersFrom

	denominatorZeroOffer := twoEurOffer
	denominatorZeroOffer.Price.D = 0

	overflowOffer := twoEurOffer
	overflowOffer.Amount = math.MaxInt64
	overflowOffer.Price.N = math.MaxInt32
	overflowOffer.Price.D = 1

	for _, testCase := range []struct {
		name               string
		offers             []xdr.OfferEntry
		ignoreOffersFrom   *xdr.AccountId
		currentAssetAmount xdr.Int64
		result             xdr.Int64
		err                error
	}{
		{
			"offers must not be empty",
			[]xdr.OfferEntry{},
			&issuer,
			100,
			0,
			errEmptyOffers,
		},
		{
			"currentAssetAmount must be positive",
			[]xdr.OfferEntry{eurOffer},
			&ignoreOffersFrom,
			0,
			0,
			errAssetAmountIsZero,
		},
		{
			"ignore all offers",
			[]xdr.OfferEntry{eurOffer},
			&issuer,
			1,
			-1,
			nil,
		},
		{
			"offer denominator cannot be zero",
			[]xdr.OfferEntry{denominatorZeroOffer},
			&ignoreOffersFrom,
			10000,
			0,
			price.ErrDivisionByZero,
		},
		{
			"ignore some offers",
			[]xdr.OfferEntry{eurOffer, otherSellerTwoEurOffer},
			&issuer,
			100,
			200,
			nil,
		},
		{
			"ignore overflow offers",
			[]xdr.OfferEntry{overflowOffer},
			nil,
			math.MaxInt64,
			-1,
			nil,
		},
		{
			"not enough offers to consume",
			[]xdr.OfferEntry{eurOffer, twoEurOffer},
			nil,
			1001,
			-1,
			nil,
		},
		{
			"consume all offers",
			[]xdr.OfferEntry{eurOffer, twoEurOffer, threeEurOffer},
			nil,
			1500,
			3000,
			nil,
		},
		{
			"consume offer partially",
			[]xdr.OfferEntry{eurOffer, twoEurOffer},
			nil,
			2,
			2,
			nil,
		},
		{
			"round up",
			[]xdr.OfferEntry{quarterOffer},
			nil,
			5,
			2,
			nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := consumeOffersForSellingAsset(
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

func TestConsumeOffersForBuyingAsset(t *testing.T) {
	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	ignoreOffersFrom := xdr.MustAddress(kp.Address())
	otherSellerTwoEurOffer := twoEurOffer
	otherSellerTwoEurOffer.SellerId = ignoreOffersFrom

	denominatorZeroOffer := twoEurOffer
	denominatorZeroOffer.Price.D = 0

	overflowOffer := twoEurOffer
	overflowOffer.Price.N = 1
	overflowOffer.Price.D = math.MaxInt32

	for _, testCase := range []struct {
		name               string
		offers             []xdr.OfferEntry
		currentAssetAmount xdr.Int64
		result             xdr.Int64
		err                error
	}{
		{
			"offers must not be empty",
			[]xdr.OfferEntry{},
			100,
			0,
			errEmptyOffers,
		},
		{
			"currentAssetAmount must be positive",
			[]xdr.OfferEntry{eurOffer},
			0,
			0,
			errAssetAmountIsZero,
		},
		{
			"offer denominator cannot be zero",
			[]xdr.OfferEntry{denominatorZeroOffer},
			10000,
			-1,
			nil,
		},
		{
			"balance too low to consume offers",
			[]xdr.OfferEntry{twoEurOffer},
			1,
			-1,
			nil,
		},
		{
			"not enough offers to consume",
			[]xdr.OfferEntry{eurOffer, twoEurOffer},
			1502,
			-1,
			nil,
		},
		{
			"ignore overflow offers",
			[]xdr.OfferEntry{overflowOffer},
			math.MaxInt64,
			-1,
			nil,
		},
		{
			"consume all offers",
			[]xdr.OfferEntry{eurOffer, twoEurOffer, threeEurOffer},
			3000,
			1500,
			nil,
		},
		{
			"consume offer partially",
			[]xdr.OfferEntry{eurOffer, twoEurOffer},
			2,
			2,
			nil,
		},
		{
			"round down",
			[]xdr.OfferEntry{eurOffer, twoEurOffer},
			1501,
			1000,
			nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := consumeOffersForBuyingAsset(
				testCase.offers,
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

func TestSortAndFilterPathsBySourceAsset(t *testing.T) {
	allPaths := []Path{
		{
			SourceAmount:      3,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      4,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      1,
			SourceAsset:       usdAsset,
			sourceAssetString: usdAsset.String(),
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      2,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      2,
			SourceAsset:       eurAsset,
			sourceAssetString: eurAsset.String(),
			InteriorNodes: []xdr.Asset{
				nativeAsset,
			},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      10,
			SourceAsset:       nativeAsset,
			sourceAssetString: nativeAsset.String(),
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
	}
	sortedAndFiltered, err := sortAndFilterPaths(
		allPaths,
		3,
		sortBySourceAsset,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	expectedPaths := []Path{
		{
			SourceAmount:      2,
			SourceAsset:       eurAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount: 2,
			SourceAsset:  eurAsset,
			InteriorNodes: []xdr.Asset{
				nativeAsset,
			},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      3,
			SourceAsset:       eurAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      1,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
		{
			SourceAmount:      10,
			SourceAsset:       nativeAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  yenAsset,
			DestinationAmount: 1000,
		},
	}

	assertPathEquals(t, sortedAndFiltered, expectedPaths)
}

func TestSortAndFilterPathsByDestinationAsset(t *testing.T) {
	allPaths := []Path{
		{
			SourceAmount:           1000,
			SourceAsset:            yenAsset,
			InteriorNodes:          []xdr.Asset{},
			DestinationAsset:       eurAsset,
			destinationAssetString: eurAsset.String(),
			DestinationAmount:      3,
		},
		{
			SourceAmount:           1000,
			SourceAsset:            yenAsset,
			InteriorNodes:          []xdr.Asset{},
			DestinationAsset:       eurAsset,
			destinationAssetString: eurAsset.String(),
			DestinationAmount:      4,
		},
		{
			SourceAmount:           1000,
			SourceAsset:            yenAsset,
			InteriorNodes:          []xdr.Asset{},
			DestinationAsset:       usdAsset,
			destinationAssetString: usdAsset.String(),
			DestinationAmount:      1,
		},
		{
			SourceAmount:           1000,
			SourceAsset:            yenAsset,
			sourceAssetString:      eurAsset.String(),
			InteriorNodes:          []xdr.Asset{},
			DestinationAsset:       eurAsset,
			destinationAssetString: eurAsset.String(),
			DestinationAmount:      2,
		},
		{
			SourceAmount: 1000,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				nativeAsset,
			},
			DestinationAsset:       eurAsset,
			destinationAssetString: eurAsset.String(),
			DestinationAmount:      2,
		},
		{
			SourceAmount:           1000,
			SourceAsset:            yenAsset,
			InteriorNodes:          []xdr.Asset{},
			DestinationAsset:       nativeAsset,
			destinationAssetString: nativeAsset.String(),
			DestinationAmount:      10,
		},
	}
	sortedAndFiltered, err := sortAndFilterPaths(
		allPaths,
		3,
		sortByDestinationAsset,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	expectedPaths := []Path{
		{
			SourceAmount:      1000,
			SourceAsset:       yenAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  eurAsset,
			DestinationAmount: 4,
		},
		{
			SourceAmount:      1000,
			SourceAsset:       yenAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  eurAsset,
			DestinationAmount: 3,
		},
		{
			SourceAmount:      1000,
			SourceAsset:       yenAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  eurAsset,
			DestinationAmount: 2,
		},
		{
			SourceAmount:      1000,
			SourceAsset:       yenAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  usdAsset,
			DestinationAmount: 1,
		},
		{
			SourceAmount:      1000,
			SourceAsset:       yenAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 10,
		},
	}

	assertPathEquals(t, sortedAndFiltered, expectedPaths)
}

func TestFindPaths(t *testing.T) {
	graph := NewOrderBookGraph()

	graph.AddOffer(dollarOffer)
	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	graph.AddOffer(quarterOffer)
	graph.AddOffer(fiftyCentsOffer)
	err := graph.Apply(1)
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

	graph.AddOffer(eurUsdOffer)
	graph.AddOffer(otherEurUsdOffer)
	graph.AddOffer(usdEurOffer)
	graph.AddOffer(chfEurOffer)
	graph.AddOffer(yenChfOffer)
	err = graph.Apply(2)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	ignoreOffersFrom := xdr.MustAddress(kp.Address())

	paths, lastLedger, err := graph.FindPaths(
		3,
		nativeAsset,
		20,
		&ignoreOffersFrom,
		[]xdr.Asset{
			yenAsset,
			usdAsset,
		},
		[]xdr.Int64{
			0,
			0,
		},
		true,
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	assertPathEquals(t, paths, []Path{})
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	paths, lastLedger, err = graph.FindPaths(
		3,
		nativeAsset,
		20,
		&ignoreOffersFrom,
		[]xdr.Asset{
			yenAsset,
			usdAsset,
		},
		[]xdr.Int64{
			100000,
			60000,
		},
		true,
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths := []Path{
		{
			SourceAmount:      5,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		{
			SourceAmount: 7,
			SourceAsset:  usdAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		{
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

	paths, lastLedger, err = graph.FindPaths(
		3,
		nativeAsset,
		20,
		&ignoreOffersFrom,
		[]xdr.Asset{
			yenAsset,
			usdAsset,
		},
		[]xdr.Int64{
			0,
			0,
		},
		false,
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	assertPathEquals(t, paths, expectedPaths)
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	paths, lastLedger, err = graph.FindPaths(
		4,
		nativeAsset,
		20,
		&ignoreOffersFrom,
		[]xdr.Asset{
			yenAsset,
			usdAsset,
		},
		[]xdr.Int64{
			100000,
			60000,
		},
		true,
		5,
	)
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths = []Path{
		{
			SourceAmount:      5,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		{
			SourceAmount: 7,
			SourceAsset:  usdAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		{
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
		{
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

	paths, lastLedger, err = graph.FindPaths(
		4,
		nativeAsset,
		20,
		&ignoreOffersFrom,
		[]xdr.Asset{
			yenAsset,
			usdAsset,
		},
		[]xdr.Int64{
			100000,
			60000,
		},
		true,
		5,
	)
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths = []Path{
		{
			SourceAmount:      5,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		{
			SourceAmount: 7,
			SourceAsset:  usdAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		{
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
		{
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

func TestFindPathsStartingAt(t *testing.T) {
	graph := NewOrderBookGraph()

	graph.AddOffer(dollarOffer)
	graph.AddOffer(threeEurOffer)
	graph.AddOffer(eurOffer)
	graph.AddOffer(twoEurOffer)
	graph.AddOffer(quarterOffer)
	graph.AddOffer(fiftyCentsOffer)

	err := graph.Apply(1)
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

	graph.AddOffer(eurUsdOffer)
	graph.AddOffer(otherEurUsdOffer)
	graph.AddOffer(usdEurOffer)
	graph.AddOffer(chfEurOffer)
	graph.AddOffer(yenChfOffer)
	err = graph.Apply(2)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	paths, lastLedger, err := graph.FindFixedPaths(
		3,
		usdAsset,
		5,
		[]xdr.Asset{nativeAsset},
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths := []Path{
		{
			SourceAmount:      5,
			SourceAsset:       usdAsset,
			InteriorNodes:     []xdr.Asset{},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
		{
			SourceAmount: 5,
			SourceAsset:  usdAsset,
			InteriorNodes: []xdr.Asset{
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 15,
		},
	}

	assertPathEquals(t, paths, expectedPaths)

	paths, lastLedger, err = graph.FindFixedPaths(
		2,
		yenAsset,
		5,
		[]xdr.Asset{nativeAsset},
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths = []Path{}

	assertPathEquals(t, paths, expectedPaths)

	paths, lastLedger, err = graph.FindFixedPaths(
		3,
		yenAsset,
		5,
		[]xdr.Asset{nativeAsset},
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths = []Path{
		{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				chfAsset,
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
	}

	assertPathEquals(t, paths, expectedPaths)

	paths, lastLedger, err = graph.FindFixedPaths(
		5,
		yenAsset,
		5,
		[]xdr.Asset{nativeAsset},
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths = []Path{
		{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				chfAsset,
				eurAsset,
				usdAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 80,
		},
		{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				chfAsset,
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
	}

	assertPathEquals(t, paths, expectedPaths)

	paths, lastLedger, err = graph.FindFixedPaths(
		5,
		yenAsset,
		5,
		[]xdr.Asset{nativeAsset, usdAsset},
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if lastLedger != 2 {
		t.Fatalf("expected last ledger to be %v but got %v", 2, lastLedger)
	}

	expectedPaths = []Path{
		{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				chfAsset,
				eurAsset,
			},
			DestinationAsset:  usdAsset,
			DestinationAmount: 20,
		},
		{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				chfAsset,
				eurAsset,
				usdAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 80,
		},
		{
			SourceAmount: 5,
			SourceAsset:  yenAsset,
			InteriorNodes: []xdr.Asset{
				chfAsset,
				eurAsset,
			},
			DestinationAsset:  nativeAsset,
			DestinationAmount: 20,
		},
	}
	assertPathEquals(t, paths, expectedPaths)
}
