package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowTrustValidateAsset(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484366))

	issuedAsset := CreditAsset{"", kp1.Address()}
	allowTrust := AllowTrust{
		Trustor:   kp1.Address(),
		Type:      issuedAsset,
		Authorize: true,
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{&allowTrust},
			Timebounds:    NewInfiniteTimeout(),
			BaseFee:       MinBaseFee,
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.AllowTrust operation: Field: Type, Error: asset code length must be between 1 and 12 characters"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestAllowTrustValidateTrustor(t *testing.T) {
	kp0 := newKeypair0()
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp0.Address(), int64(40385577484366))

	issuedAsset := CreditAsset{"ABCD", kp1.Address()}
	allowTrust := AllowTrust{
		Trustor:   "",
		Type:      issuedAsset,
		Authorize: true,
	}

	_, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []Operation{&allowTrust},
			Timebounds:    NewInfiniteTimeout(),
			BaseFee:       MinBaseFee,
		},
	)
	if assert.Error(t, err) {
		expected := "validation failed for *txnbuild.AllowTrust operation: Field: Trustor, Error: public key is undefined"
		assert.Contains(t, err.Error(), expected)
	}
}

func TestAllowTrustRoundtrip(t *testing.T) {
	allowTrust := AllowTrust{
		SourceAccount: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Trustor:       "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Type:          CreditAsset{"USD", "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H"},
		Authorize:     true,
	}
	testOperationsMarshallingRoundtrip(t, []Operation{&allowTrust}, false)

	// with muxed accounts
	allowTrust = AllowTrust{
		SourceAccount: "MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK",
		Trustor:       "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		Type:          CreditAsset{"USD", "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"},
		Authorize:     true,
	}

	testOperationsMarshallingRoundtrip(t, []Operation{&allowTrust}, true)
}

// Ensures that the issuer is back-filled from the source account correctly.
// See https://github.com/stellar/go/pull/3636 for more context.
func TestDecoderIncludesIssuer(t *testing.T) {
	testCases := []struct {
		source  string
		trustor string
		issuer  string
	}{
		{
			"GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
			"GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
			"GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		},
		{
			"MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK",
			"GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
			"GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ",
		},
	}
	for _, testCase := range testCases {
		op := AllowTrust{
			SourceAccount: testCase.source,
			Trustor:       testCase.trustor,
			Type:          CreditAsset{"CODE", testCase.issuer},
			Authorize:     true,
		}

		encodedOp, err := op.BuildXDR(true)
		assert.NoError(t, err)

		decodedOp := AllowTrust{}
		err = decodedOp.FromXDR(encodedOp, true)
		assert.NoError(t, err)
		assert.Equal(t, testCase.issuer, decodedOp.Type.GetIssuer())
	}
}
