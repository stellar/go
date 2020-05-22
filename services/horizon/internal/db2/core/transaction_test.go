package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestTransactionsQueries(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.CoreSession()}

	// Test TransactionsByLedger
	var txs []Transaction
	err := q.TransactionsByLedger(&txs, 2)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(txs, 3)
	}

	// Test TransactionByHashAfterLedger
	var tx Transaction
	err = q.TransactionByHashAfterLedger(&tx, "cebb875a00ff6e1383aef0fd251a76f22c1f9ab2a2dffcb077855736ade2659a", 2)

	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(3), tx.LedgerSequence)
	}

	err = q.TransactionByHashAfterLedger(&tx, "cebb875a00ff6e1383aef0fd251a76f22c1f9ab2a2dffcb077855736ade2659a", 3)

	if tt.Assert.Error(err) {
		tt.Assert.True(q.NoRows(err))
	}
}

func TestMemo(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	var tx Transaction

	xdr.SafeUnmarshalBase64("AAAAAMvoFDdcyQrJAcBmRdyEnW6047pvlk4MS/4r0n/1WH8VAAAAZAACnMAAAAACAAAAAAAAAAEAAAARADEuMC4xb3dlcnJpZGUgbWUAAAAAAAABAAAAAQAAAACJzogbLxrrmN7N5JVQceSxl8jkED26RGzbyyRIpwTh6wAAAAoAAAAWaSBzaG91bGQgYmUgb3dlcnJpZGRlbgAAAAAAAQAAABVpIHNob3VsZCBiZSBvd2VycmlkZW4AAAAAAAAAAAAAAacE4esAAABA0GuCIEmKyQ2DRqt5+BOIqjVlHisjY6rK1IcOtzjIKCDgSAoiv5yhYe09PohBH91TXvAQ/LZJj5hVMihfMjtgCw==", &tx.Envelope)

	tt.Assert.Equal("1.0.1owerride me", tx.Memo().String)
}

func TestSignatures(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	var tx Transaction

	// https://github.com/stellar/stellar-core/issues/1225
	xdr.SafeUnmarshalBase64("AAAAAMIK9djC7k75ziKOLJcvMAIBG7tnBuoeI34x+Pi6zqcZAAAAZAAZphYAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAynnCTTyw53VVRLOWX6XKTva63IM1LslPNW01YB0hz/8AAAAAAAAAAlQL5AAAAAAAAAAAAh0hz/8AAABA8qkkeKaKfsbgInyIkzXJhqJE5/Ufxri2LdxmyKkgkT6I3sPmvrs5cPWQSzEQyhV750IW2ds97xTHqTpOfuZCAnhSuFUAAAAA", &tx.Envelope)

	signatures := tx.Base64Signatures()

	tt.Assert.Equal(2, len(signatures))
	tt.Assert.Equal("8qkkeKaKfsbgInyIkzXJhqJE5/Ufxri2LdxmyKkgkT6I3sPmvrs5cPWQSzEQyhV750IW2ds97xTHqTpOfuZCAg==", signatures[0])
	tt.Assert.Equal("", signatures[1])
}

func TestTransaction_SourceAddress_MuxedAccount(t *testing.T) {
	aid := xdr.MustAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")

	muxed := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xcafebabe,
			Ed25519: *aid.Ed25519,
		},
	}
	var tx Transaction
	tx.Envelope = xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1: &xdr.TransactionV1Envelope{
			Tx: xdr.Transaction{
				SourceAccount: muxed,
				Operations: []xdr.Operation{
					{
						SourceAccount: &muxed,
						Body: xdr.OperationBody{
							Type: xdr.OperationTypePayment,
							PaymentOp: &xdr.PaymentOp{
								Destination: muxed,
								Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
								Amount:      100,
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ", tx.SourceAddress())
}
