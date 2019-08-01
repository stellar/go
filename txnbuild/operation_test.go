package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateAccountFromXDR(t *testing.T) {
	txeB64 := "AAAAAMOrP0B2tL9IUn5QL8nn8q88kkFui1x3oW9omCj6hLhfAAAAZAAAAMcAAAAWAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAEAAAAAEH3Rayw4M0iCLoEe96rPFNGYim8AVHJU0z4ebYZW4JwAAAAAAAAAAJ5yfHhgKAxylgecjAymWqNzLWRk/MqSYt+X9duZ2DfyAAAAF0h26AAAAAAAAAAAAvqEuF8AAABAZ5q2N2BHRylT28T1DbUVU7QKTbKZ+6DLefzJoCjHo2T8vcI/PjF8gsRu/r2M60Uzcw3WmqRFerA6DnJILIEdDoZW4JwAAABAsFL3WXr+tDK5tjR/0ZBVuNyzyqSa8Li2tUMUmB23PWuPG71ObUPTShkhlc7ydNN/qYRaA/Mafm+vsIQWDbCRDA=="

	xdrEnv, err := unmarshalBase64(txeB64)
	if assert.NoError(t, err) {
		var ca CreateAccount
		err = ca.FromXDR(xdrEnv.Tx.Operations[0])
		if assert.NoError(t, err) {
			assert.Equal(t, "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR", ca.SourceAccount.GetAccountID(), "source accounts should match")
			assert.Equal(t, "GCPHE7DYMAUAY4UWA6OIYDFGLKRXGLLEMT6MVETC36L7LW4Z3A37EJW5", ca.Destination, "destination should match")
			assert.Equal(t, "10000.0000000", ca.Amount, "starting balance should match")
		}
	}

	txeB64NoSource := "AAAAAGigiN2q4qBXAERImNEncpaADylyBRtzdqpEsku6CN0xAAAAyAAADXYAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAEH3Rayw4M0iCLoEe96rPFNGYim8AVHJU0z4ebYZW4JwAAAAAdgRnAAAAAAAAAAAA"
	xdrEnv, err = unmarshalBase64(txeB64NoSource)
	if assert.NoError(t, err) {
		var ca CreateAccount
		err = ca.FromXDR(xdrEnv.Tx.Operations[0])
		if assert.NoError(t, err) {
			assert.Equal(t, nil, ca.SourceAccount, "source accounts should match")
			assert.Equal(t, "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR", ca.Destination, "destination should match")
			assert.Equal(t, "198.0000000", ca.Amount, "starting balance should match")
		}
	}

}

func TestPaymentFromXDR(t *testing.T) {
	txeB64 := "AAAAAGigiN2q4qBXAERImNEncpaADylyBRtzdqpEsku6CN0xAAABkAAADXYAAAABAAAAAAAAAAAAAAACAAAAAQAAAABooIjdquKgVwBESJjRJ3KWgA8pcgUbc3aqRLJLugjdMQAAAAEAAAAAEH3Rayw4M0iCLoEe96rPFNGYim8AVHJU0z4ebYZW4JwAAAAAAAAAAAX14QAAAAAAAAAAAQAAAAAQfdFrLDgzSIIugR73qs8U0ZiKbwBUclTTPh5thlbgnAAAAAFYWQAAAAAAAGigiN2q4qBXAERImNEncpaADylyBRtzdqpEsku6CN0xAAAAAE/exwAAAAAAAAAAAA=="

	xdrEnv, err := unmarshalBase64(txeB64)
	if assert.NoError(t, err) {
		var p Payment
		err = p.FromXDR(xdrEnv.Tx.Operations[0])
		if assert.NoError(t, err) {
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", p.SourceAccount.GetAccountID(), "source accounts should match")
			assert.Equal(t, "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR", p.Destination, "destination should match")
			assert.Equal(t, "10.0000000", p.Amount, "amount should match")
			assert.Equal(t, true, p.Asset.IsNative(), "Asset should be native")
		}

		err = p.FromXDR(xdrEnv.Tx.Operations[1])
		if assert.NoError(t, err) {
			assert.Equal(t, nil, p.SourceAccount, "source accounts should match")
			assert.Equal(t, "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR", p.Destination, "destination should match")
			assert.Equal(t, "134.0000000", p.Amount, "amount should match")
			assetType, err := p.Asset.GetType()
			assert.NoError(t, err)
			assert.Equal(t, AssetTypeCreditAlphanum4, assetType, "Asset type should match")
			assert.Equal(t, "XY", p.Asset.GetCode(), "Asset code should match")
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", p.Asset.GetIssuer(), "Asset issuer should match")
		}
	}
}

func TestPathPaymentFromXDR(t *testing.T) {
	txeB64 := "AAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAZAAAql0AAAADAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAAF9eEAAAAAAH4RyzTWNfXhqwLUoCw91aWkZtgIzY8SAVkIPc0uFVmYAAAAAAAAAAAAmJaAAAAAAQAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAAAAAAEuFVmYAAAAQF2kLUL/RoFIy1cmt+GXdWn2tDUjJYV3YwF4A82zIBhqYSO6ogOoLPNRt3w+IGCAgfR4Q9lpax+wCXWoQERHSw4="

	xdrEnv, err := unmarshalBase64(txeB64)
	if assert.NoError(t, err) {
		var pp PathPayment
		err = pp.FromXDR(xdrEnv.Tx.Operations[0])
		if assert.NoError(t, err) {
			assert.Equal(t, nil, pp.SourceAccount, "source accounts should match")
			assert.Equal(t, "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H", pp.Destination, "destination should match")
			assert.Equal(t, "1.0000000", pp.DestAmount, "DestAmount should match")
			assert.Equal(t, "10.0000000", pp.SendMax, "SendMax should match")
			assert.Equal(t, true, pp.DestAsset.IsNative(), "DestAsset should be native")
			assert.Equal(t, 1, len(pp.Path), "Number of paths should be 1")
			assetType, err := pp.Path[0].GetType()
			assert.NoError(t, err)
			assert.Equal(t, AssetTypeCreditAlphanum4, assetType, "Asset type should match")
			assert.Equal(t, "ABCD", pp.Path[0].GetCode(), "Asset code should match")
			assert.Equal(t, "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3", pp.Path[0].GetIssuer(), "Asset issuer should match")
		}
	}
}

func TestManageSellOfferFromXDR(t *testing.T) {
	txeB64 := "AAAAAGigiN2q4qBXAERImNEncpaADylyBRtzdqpEsku6CN0xAAABkAAADXYAAAABAAAAAAAAAAAAAAACAAAAAQAAAABooIjdquKgVwBESJjRJ3KWgA8pcgUbc3aqRLJLugjdMQAAAAMAAAAAAAAAAkFCQ1hZWgAAAAAAAAAAAABooIjdquKgVwBESJjRJ3KWgA8pcgUbc3aqRLJLugjdMQAAAACy0F4AAAAABQAAAAEAAAAAAAAAAAAAAAAAAAADAAAAAUFCQwAAAAAAaKCI3arioFcAREiY0SdyloAPKXIFG3N2qkSyS7oI3TEAAAAAAAAAAO5rKAAAAAAFAAAAAQAAAAAAAAAAAAAAAAAAAAA="

	xdrEnv, err := unmarshalBase64(txeB64)
	if assert.NoError(t, err) {
		var mso ManageSellOffer
		err = mso.FromXDR(xdrEnv.Tx.Operations[0])
		if assert.NoError(t, err) {
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", mso.SourceAccount.GetAccountID(), "source accounts should match")
			assert.Equal(t, int64(0), mso.OfferID, "OfferID should match")
			assert.Equal(t, "300.0000000", mso.Amount, "Amount should match")
			assert.Equal(t, "5.0000000", mso.Price, "Price should match")
			assert.Equal(t, true, mso.Selling.IsNative(), "Selling should be native")
			assetType, err := mso.Buying.GetType()
			assert.NoError(t, err)
			assert.Equal(t, AssetTypeCreditAlphanum12, assetType, "Asset type should match")
			assert.Equal(t, "ABCXYZ", mso.Buying.GetCode(), "Asset code should match")
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", mso.Buying.GetIssuer(), "Asset issuer should match")
		}

		err = mso.FromXDR(xdrEnv.Tx.Operations[1])
		if assert.NoError(t, err) {
			assert.Equal(t, nil, mso.SourceAccount, "source accounts should match")
			assert.Equal(t, int64(0), mso.OfferID, "OfferID should match")
			assert.Equal(t, "400.0000000", mso.Amount, "Amount should match")
			assert.Equal(t, "5.0000000", mso.Price, "Price should match")
			assert.Equal(t, true, mso.Buying.IsNative(), "Buying should be native")
			assetType, err := mso.Selling.GetType()
			assert.NoError(t, err)
			assert.Equal(t, AssetTypeCreditAlphanum4, assetType, "Asset type should match")
			assert.Equal(t, "ABC", mso.Selling.GetCode(), "Asset code should match")
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", mso.Selling.GetIssuer(), "Asset issuer should match")
		}

	}
}

func TestManageBuyOfferFromXDR(t *testing.T) {
	txeB64 := "AAAAAGigiN2q4qBXAERImNEncpaADylyBRtzdqpEsku6CN0xAAABkAAADXYAAAABAAAAAAAAAAAAAAACAAAAAQAAAABooIjdquKgVwBESJjRJ3KWgA8pcgUbc3aqRLJLugjdMQAAAAwAAAAAAAAAAkFCQ1hZWgAAAAAAAAAAAABooIjdquKgVwBESJjRJ3KWgA8pcgUbc3aqRLJLugjdMQAAAAA7msoAAAAAAQAAAAIAAAAAAAAAAAAAAAAAAAAMAAAAAUFCQwAAAAAAaKCI3arioFcAREiY0SdyloAPKXIFG3N2qkSyS7oI3TEAAAAAAAAAALLQXgAAAAADAAAABQAAAAAAAAAAAAAAAAAAAAA="

	xdrEnv, err := unmarshalBase64(txeB64)
	if assert.NoError(t, err) {
		var mbo ManageBuyOffer
		err = mbo.FromXDR(xdrEnv.Tx.Operations[0])
		if assert.NoError(t, err) {
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", mbo.SourceAccount.GetAccountID(), "source accounts should match")
			assert.Equal(t, int64(0), mbo.OfferID, "OfferID should match")
			assert.Equal(t, "100.0000000", mbo.Amount, "Amount should match")
			assert.Equal(t, "0.5000000", mbo.Price, "Price should match")
			assert.Equal(t, true, mbo.Selling.IsNative(), "Selling should be native")
			assetType, err := mbo.Buying.GetType()
			assert.NoError(t, err)
			assert.Equal(t, AssetTypeCreditAlphanum12, assetType, "Asset type should match")
			assert.Equal(t, "ABCXYZ", mbo.Buying.GetCode(), "Asset code should match")
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", mbo.Buying.GetIssuer(), "Asset issuer should match")
		}

		err = mbo.FromXDR(xdrEnv.Tx.Operations[1])
		if assert.NoError(t, err) {
			assert.Equal(t, nil, mbo.SourceAccount, "source accounts should match")
			assert.Equal(t, int64(0), mbo.OfferID, "OfferID should match")
			assert.Equal(t, "300.0000000", mbo.Amount, "Amount should match")
			assert.Equal(t, "0.6000000", mbo.Price, "Price should match")
			assert.Equal(t, true, mbo.Buying.IsNative(), "Buying should be native")
			assetType, err := mbo.Selling.GetType()
			assert.NoError(t, err)
			assert.Equal(t, AssetTypeCreditAlphanum4, assetType, "Asset type should match")
			assert.Equal(t, "ABC", mbo.Selling.GetCode(), "Asset code should match")
			assert.Equal(t, "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU", mbo.Selling.GetIssuer(), "Asset issuer should match")
		}

	}
}

func TestCreatePassiveSellOfferFromXDR(t *testing.T) {
	txeB64 := "AAAAACXK8doPx27P6IReQlRRuweSSUiUfjqgyswxiu3Sh2R+AAAAZAAAJWoAAAANAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAEAAAAAAAAAAFBQkNEAAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAAAX14QAAAAABAAAAAQAAAAAAAAAB0odkfgAAAEAgUD7M1UL7x2m2m26ySzcSHxIneOT7/r+s/HLsgWDj6CmpSi1GZrlvtBH+CNuegCwvW09TRZJhp7bLywkaFCoK"

	xdrEnv, err := unmarshalBase64(txeB64)
	if assert.NoError(t, err) {
		var cpo CreatePassiveSellOffer
		err = cpo.FromXDR(xdrEnv.Tx.Operations[0])
		if assert.NoError(t, err) {
			assert.Equal(t, nil, cpo.SourceAccount, "source accounts should match")
			assert.Equal(t, "10.0000000", cpo.Amount, "Amount should match")
			assert.Equal(t, "1.0000000", cpo.Price, "Price should match")
			assert.Equal(t, true, cpo.Selling.IsNative(), "Selling should be native")
			assetType, err := cpo.Buying.GetType()
			assert.NoError(t, err)
			assert.Equal(t, AssetTypeCreditAlphanum4, assetType, "Asset type should match")
			assert.Equal(t, "ABCD", cpo.Buying.GetCode(), "Asset code should match")
			assert.Equal(t, "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3", cpo.Buying.GetIssuer(), "Asset issuer should match")
		}
	}
}
