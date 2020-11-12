package txnbuild

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/stellar/go/network"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestFeeBumpMissingInner(t *testing.T) {
	_, err := NewFeeBumpTransaction(FeeBumpTransactionParams{})
	assert.EqualError(t, err, "inner transaction is missing")
}

func TestFeeBumpInvalidFeeSource(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), 1)

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{&Inflation{}},
			BaseFee:       MinBaseFee,
			Timebounds:    NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	_, err = NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: "/.','ml",
			BaseFee:    MinBaseFee,
			Inner:      tx,
		},
	)
	assert.Contains(t, err.Error(), "fee account is not a valid address")
}

func TestFeeBumpUpgradesV0Transaction(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), 1)

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&Inflation{}},
			BaseFee:              2 * MinBaseFee,
			Memo:                 MemoText("test-memo"),
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	convertToV0(tx)

	feeBump, err := NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: newKeypair1().Address(),
			BaseFee:    3 * MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)

	assert.Equal(t, xdr.EnvelopeTypeEnvelopeTypeTx, feeBump.InnerTransaction().envelope.Type)
	assert.Equal(t, xdr.EnvelopeTypeEnvelopeTypeTxV0, tx.envelope.Type)

	innerHash, err := feeBump.InnerTransaction().HashHex(network.TestNetworkPassphrase)
	assert.NoError(t, err)
	originalHash, err := tx.HashHex(network.TestNetworkPassphrase)
	assert.NoError(t, err)
	assert.Equal(t, originalHash, innerHash)

	assert.Equal(t, tx.Signatures(), feeBump.InnerTransaction().Signatures())
	assert.Equal(t, tx.Operations(), feeBump.InnerTransaction().Operations())
	assert.Equal(t, tx.MaxFee(), feeBump.InnerTransaction().MaxFee())
	assert.Equal(t, tx.BaseFee(), feeBump.InnerTransaction().BaseFee())
	assert.Equal(t, tx.SourceAccount(), feeBump.InnerTransaction().SourceAccount())
	assert.Equal(t, tx.Memo(), feeBump.InnerTransaction().Memo())
	assert.Equal(t, tx.Timebounds(), feeBump.InnerTransaction().Timebounds())

	innerBase64, err := feeBump.InnerTransaction().Base64()
	assert.NoError(t, err)
	originalBase64, err := tx.Base64()
	assert.NoError(t, err)
	assert.NotEqual(t, innerBase64, originalBase64)
}

func TestFeeBumpInvalidInnerTransactionType(t *testing.T) {
	kp0 := newKeypair0()
	sourceAccount := NewSimpleAccount(kp0.Address(), 1)

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: false,
			Operations:           []Operation{&Inflation{}},
			BaseFee:              2 * MinBaseFee,
			Memo:                 MemoText("test-memo"),
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	aid := xdr.MustAddress(kp0.Address())
	tx.envelope.Type = xdr.EnvelopeTypeEnvelopeTypeTxFeeBump
	tx.envelope.FeeBump = &xdr.FeeBumpTransactionEnvelope{
		Tx: xdr.FeeBumpTransaction{
			FeeSource: aid.ToMuxedAccount(),
			InnerTx: xdr.FeeBumpTransactionInnerTx{
				Type: xdr.EnvelopeTypeEnvelopeTypeTx,
				V1:   tx.envelope.V1,
			},
		},
		Signatures: nil,
	}
	_, err = NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: newKeypair1().Address(),
			BaseFee:    3 * MinBaseFee,
			Inner:      tx,
		},
	)
	assert.EqualError(t, err, "EnvelopeTypeEnvelopeTypeTxFeeBump transactions cannot be fee bumped")
}

// There is a use case for having a fee bump tx where the fee account is equal to the
// source account of the inner transaction. Consider the case where the signers of the
// inner transaction could be different (which is the case when dealing with operations
// on different source accounts).
func TestFeeBumpAllowsFeeAccountToEqualInnerSourceAccount(t *testing.T) {
	sourceAccount := NewSimpleAccount("GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3", 1)
	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{&Inflation{}},
			BaseFee:       MinBaseFee,
			Timebounds:    NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)

	_, err = NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: sourceAccount.AccountID,
			BaseFee:    MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)

	muxedAccount := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0,
			Ed25519: xdr.Uint256{1, 2, 3},
		},
	}
	tx.envelope.V1.Tx.SourceAccount = muxedAccount

	otherAccount := xdr.AccountId{
		Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
		Ed25519: &xdr.Uint256{1, 2, 3},
	}
	_, err = NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: otherAccount.Address(),
			BaseFee:    MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)

	otherAccount = xdr.AccountId{
		Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
		Ed25519: &xdr.Uint256{1, 2, 3},
	}
	_, err = NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: otherAccount.Address(),
			BaseFee:    MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)
}

func TestFeeBumpSignWithKeyString(t *testing.T) {
	kp0, kp1 := newKeypair0(), newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), 1)

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{&Inflation{}},
			BaseFee:       MinBaseFee,
			Timebounds:    NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	feeBumpTx, err := NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: kp1.Address(),
			BaseFee:    2 * MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)
	feeBumpTx, err = feeBumpTx.Sign(network.TestNetworkPassphrase, kp1)
	assert.NoError(t, err)
	expectedBase64, err := feeBumpTx.Base64()
	assert.NoError(t, err)

	feeBumpTx, err = NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: kp1.Address(),
			BaseFee:    2 * MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)
	feeBumpTx, err = feeBumpTx.SignWithKeyString(network.TestNetworkPassphrase, kp1.Seed())
	assert.NoError(t, err)
	base64, err := feeBumpTx.Base64()
	assert.NoError(t, err)

	assert.Equal(t, expectedBase64, base64)
}

func TestFeeBumpSignHashX(t *testing.T) {
	// 256 bit preimage
	preimage := "this is a preimage for hashx transactions on the stellar network"
	preimageHash := sha256.Sum256([]byte(preimage))

	kp0, kp1 := newKeypair0(), newKeypair1()
	payment := Payment{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       NativeAsset{},
	}
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(4353383146192899))

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []Operation{&payment},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	feeBumpTx, err := NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: kp1.Address(),
			BaseFee:    2 * MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)
	feeBumpTx, err = feeBumpTx.SignHashX([]byte(preimage))
	assert.NoError(t, err)

	signatures := feeBumpTx.Signatures()
	assert.Len(t, signatures, 1)
	assert.Equal(t, xdr.Signature(preimage), signatures[0].Signature)
	var expectedHint [4]byte
	copy(expectedHint[:], preimageHash[28:])
	assert.Equal(t, xdr.SignatureHint(expectedHint), signatures[0].Hint)
}

func TestFeeBumpAddSignatureBase64(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	kp2 := newKeypair2()
	txSource := NewSimpleAccount(kp0.Address(), int64(9605939170639897))
	opSource := NewSimpleAccount(kp1.Address(), 0)
	createAccount := CreateAccount{
		Destination:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:        "10",
		SourceAccount: &opSource,
	}

	inner, err := NewTransaction(
		TransactionParams{
			SourceAccount:        &txSource,
			IncrementSequenceNum: true,
			Operations:           []Operation{&createAccount},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	inner, err = inner.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)

	tx, err := NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: kp1.Address(),
			BaseFee:    2 * MinBaseFee,
			Inner:      inner,
		},
	)
	assert.NoError(t, err)
	tx, err = tx.Sign(network.TestNetworkPassphrase, kp1, kp2)
	assert.NoError(t, err)
	expected, err := tx.Base64()
	assert.NoError(t, err)
	signatures := tx.Signatures()

	otherTx, err := NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: kp1.Address(),
			BaseFee:    2 * MinBaseFee,
			Inner:      inner,
		},
	)
	assert.NoError(t, err)
	otherTx, err = otherTx.AddSignatureBase64(
		network.TestNetworkPassphrase,
		kp1.Address(),
		base64.StdEncoding.EncodeToString(signatures[0].Signature),
	)
	assert.NoError(t, err)
	otherTx, err = otherTx.AddSignatureBase64(
		network.TestNetworkPassphrase,
		kp2.Address(),
		base64.StdEncoding.EncodeToString(signatures[1].Signature),
	)
	assert.NoError(t, err)
	b64, err := tx.Base64()
	assert.NoError(t, err)

	assert.Equal(t, expected, b64)
}

func TestFeeBumpRoundTrip(t *testing.T) {
	kp0, kp1 := newKeypair0(), newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), 1)

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{&Inflation{}},
			BaseFee:       MinBaseFee,
			Timebounds:    NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	tx, err = tx.Sign(network.TestNetworkPassphrase, kp0)
	assert.NoError(t, err)
	expectedInnerB64, err := tx.Base64()
	assert.NoError(t, err)

	feeBumpTx, err := NewFeeBumpTransaction(
		FeeBumpTransactionParams{
			FeeAccount: kp1.Address(),
			BaseFee:    2 * MinBaseFee,
			Inner:      tx,
		},
	)
	assert.NoError(t, err)
	feeBumpTx, err = feeBumpTx.Sign(network.TestNetworkPassphrase, kp1)
	assert.NoError(t, err)

	innerB64, err := feeBumpTx.InnerTransaction().Base64()
	assert.NoError(t, err)
	assert.Equal(t, expectedInnerB64, innerB64)

	assert.Equal(t, kp1.Address(), feeBumpTx.FeeAccount())
	assert.Equal(t, int64(2*MinBaseFee), feeBumpTx.BaseFee())
	assert.Equal(t, int64(4*MinBaseFee), feeBumpTx.MaxFee())

	outerHash, err := feeBumpTx.HashHex(network.TestNetworkPassphrase)
	assert.NoError(t, err)

	env, err := feeBumpTx.TxEnvelope()
	assert.NoError(t, err)
	assert.Equal(t, xdr.EnvelopeTypeEnvelopeTypeTxFeeBump, env.Type)
	assert.Equal(t, xdr.MustAddress(kp1.Address()), env.FeeBumpAccount().ToAccountId())
	assert.Equal(t, int64(4*MinBaseFee), env.FeeBumpFee())
	assert.Equal(t, feeBumpTx.Signatures(), env.FeeBumpSignatures())
	innerB64, err = xdr.MarshalBase64(xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1:   env.FeeBump.Tx.InnerTx.V1,
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedInnerB64, innerB64)

	expectedFeeBumpB64, err := xdr.MarshalBase64(env)
	assert.NoError(t, err)

	b64, err := feeBumpTx.Base64()
	assert.NoError(t, err)
	assert.Equal(t, expectedFeeBumpB64, b64)

	binary, err := feeBumpTx.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, expectedFeeBumpB64, base64.StdEncoding.EncodeToString(binary))

	parsed, err := TransactionFromXDR(expectedFeeBumpB64)
	assert.NoError(t, err)
	parsedFeeBump, ok := parsed.FeeBump()
	assert.True(t, ok)
	_, ok = parsed.Transaction()
	assert.False(t, ok)

	parsedHash, err := parsedFeeBump.HashHex(network.TestNetworkPassphrase)
	assert.NoError(t, err)

	assert.Equal(t, feeBumpTx.Signatures(), parsedFeeBump.Signatures())
	assert.Equal(t, kp1.Address(), parsedFeeBump.FeeAccount())
	assert.Equal(t, int64(2*MinBaseFee), parsedFeeBump.BaseFee())
	assert.Equal(t, int64(4*MinBaseFee), parsedFeeBump.MaxFee())
	innerB64, err = xdr.MarshalBase64(xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1:   parsedFeeBump.envelope.FeeBump.Tx.InnerTx.V1,
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedInnerB64, innerB64)
	b64, err = parsedFeeBump.Base64()
	assert.NoError(t, err)
	assert.Equal(t, expectedFeeBumpB64, b64)
	assert.Equal(t, outerHash, parsedHash)
}
