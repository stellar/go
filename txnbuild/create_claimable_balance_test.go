package txnbuild

import (
	"testing"
)

func TestCreateClaimableBalanceRoundTrip(t *testing.T) {
	and := AndPredicate(BeforeAbsoluteTimePredicate(100), BeforeRelativeTimePredicate(10))
	createNativeBalance := &CreateClaimableBalance{
		Amount: "1234.0000000",
		Asset:  NativeAsset{},
		Destinations: []Claimant{
			NewClaimant(newKeypair1().Address(), &UnconditionalPredicate),
			NewClaimant(newKeypair1().Address(), &and),
		},
	}

	not := NotPredicate(UnconditionalPredicate)
	or := OrPredicate(BeforeAbsoluteTimePredicate(100), BeforeRelativeTimePredicate(10))
	createAssetBalance := &CreateClaimableBalance{
		Amount: "99.0000000",
		Asset: CreditAsset{
			Code:   "COP",
			Issuer: "GB56OJGSA6VHEUFZDX6AL2YDVG2TS5JDZYQJHDYHBDH7PCD5NIQKLSDO",
		},
		Destinations: []Claimant{
			NewClaimant(newKeypair1().Address(), &not),
			NewClaimant(newKeypair1().Address(), &or),
		},
	}

	roundTrip(t, []Operation{createNativeBalance, createAssetBalance})
}
