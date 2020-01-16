package txnbuild

import (
	"testing"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestHandleSetFlagsThreeDifferent(t *testing.T) {
	options := SetOptions{}
	options.SetFlags = []AccountFlag{1, 2, 4}

	options.handleSetFlags()

	expected := xdr.Uint32(7)
	assert.Equal(t, expected, *options.xdrOp.SetFlags, "three different valid flags are ok")
}

func TestHandleSetFlagsThreeSame(t *testing.T) {
	options := SetOptions{}
	options.SetFlags = []AccountFlag{1, 1, 1}

	options.handleSetFlags()

	expected := xdr.Uint32(1)
	assert.Equal(t, expected, *options.xdrOp.SetFlags, "three of the same valid flags are ok")
}

func TestHandleSetFlagsRedundantFlagsAllowed(t *testing.T) {
	options := SetOptions{}
	options.SetFlags = []AccountFlag{1, 2, 4, 2, 4, 1}

	options.handleSetFlags()

	expected := xdr.Uint32(7)
	assert.Equal(t, expected, *options.xdrOp.SetFlags, "additional redundant flags are allowed")
}

func TestHandleSetFlagsLessThanThreeAreOK(t *testing.T) {
	options := SetOptions{}
	options.SetFlags = []AccountFlag{1, 2}

	options.handleSetFlags()

	expected := xdr.Uint32(3)
	assert.Equal(t, expected, *options.xdrOp.SetFlags, "less than three flags are ok")
}

func TestHandleSetFlagsInvalidFlagsAllowed(t *testing.T) {
	options := SetOptions{}
	options.SetFlags = []AccountFlag{3, 3, 3}

	options.handleSetFlags()

	expected := xdr.Uint32(3)
	assert.Equal(t, expected, *options.xdrOp.SetFlags, "invalid flags are allowed")
}

func TestHandleSetFlagsZeroFlagsAreOK(t *testing.T) {
	options := SetOptions{}
	options.SetFlags = []AccountFlag{0, 2, 0}

	options.handleSetFlags()

	expected := xdr.Uint32(2)
	assert.Equal(t, expected, *options.xdrOp.SetFlags, "zero flags are ok")
}

func TestHandleClearFlagsThreeDifferent(t *testing.T) {
	options := SetOptions{}
	options.ClearFlags = []AccountFlag{1, 2, 4}

	options.handleClearFlags()

	expected := xdr.Uint32(7)
	assert.Equal(t, expected, *options.xdrOp.ClearFlags, "three different valid flags are ok")
}

func TestHandleClearFlagsThreeSame(t *testing.T) {
	options := SetOptions{}
	options.ClearFlags = []AccountFlag{1, 1, 1}

	options.handleClearFlags()

	expected := xdr.Uint32(1)
	assert.Equal(t, expected, *options.xdrOp.ClearFlags, "three of the same valid flags are ok")
}

func TestHandleClearFlagsRedundantFlagsAllowed(t *testing.T) {
	options := SetOptions{}
	options.ClearFlags = []AccountFlag{1, 2, 4, 2, 4, 1}

	options.handleClearFlags()

	expected := xdr.Uint32(7)
	assert.Equal(t, expected, *options.xdrOp.ClearFlags, "additional redundant flags are allowed")
}

func TestHandleClearFlagsLessThanThreeAreOK(t *testing.T) {
	options := SetOptions{}
	options.ClearFlags = []AccountFlag{1, 2}

	options.handleClearFlags()

	expected := xdr.Uint32(3)
	assert.Equal(t, expected, *options.xdrOp.ClearFlags, "less than three flags are ok")
}

func TestHandleClearFlagsInvalidFlagsAllowed(t *testing.T) {
	options := SetOptions{}
	options.ClearFlags = []AccountFlag{3, 3, 3}

	options.handleClearFlags()

	expected := xdr.Uint32(3)
	assert.Equal(t, expected, *options.xdrOp.ClearFlags, "invalid flags are allowed")
}

func TestHandleClearFlagsZeroFlagsAreOK(t *testing.T) {
	options := SetOptions{}
	options.ClearFlags = []AccountFlag{0, 2, 0}

	options.handleClearFlags()

	expected := xdr.Uint32(2)
	assert.Equal(t, expected, *options.xdrOp.ClearFlags, "zero flags are ok")
}

func TestEmptyHomeDomainOK(t *testing.T) {
	options := SetOptions{
		HomeDomain: NewHomeDomain(""),
	}
	options.BuildXDR()

	assert.Equal(t, string(*options.xdrOp.HomeDomain), "", "empty string home domain is set")

}

func TestSignerFromHorizon(t *testing.T) {
	horizonSigner := hProtocol.Signer{Key: "GAABGBW5DINUS456OTHH6IUPTQSQZVVFCZGAO467OLIPFUWTMV6XR5XS", Weight: 10}
	wantSigner := Signer{Address: "GAABGBW5DINUS456OTHH6IUPTQSQZVVFCZGAO467OLIPFUWTMV6XR5XS", Weight: 10}
	signer := SignerFromHorizon(horizonSigner)
	assert.Equal(t, wantSigner, signer)
}

func TestSignersFromHorizon(t *testing.T) {
	horizonSigners := []hProtocol.Signer{
		{Key: "GAABGBW5DINUS456OTHH6IUPTQSQZVVFCZGAO467OLIPFUWTMV6XR5XS", Weight: 0},
		{Key: "GAT4FUGQNKOIDLOIXCJJYFSAFJHQY5MZEZLRBXXFKDCXGUHJQ63XZFTD", Weight: 10},
		{Key: "GCB35H32SU5ME4OALQUPOM4AADJICL2H2NLWAGLOMMTSYOVTXWYP6Q4T", Weight: 100},
		{Key: "GAVJHRCK5CEFE3MHL4JALMNX35Z5NLUIODSJIC44VRRLQQZGTDJWANV4", Weight: 255},
	}
	wantSigners := []Signer{
		{Address: "GAABGBW5DINUS456OTHH6IUPTQSQZVVFCZGAO467OLIPFUWTMV6XR5XS", Weight: 0},
		{Address: "GAT4FUGQNKOIDLOIXCJJYFSAFJHQY5MZEZLRBXXFKDCXGUHJQ63XZFTD", Weight: 10},
		{Address: "GCB35H32SU5ME4OALQUPOM4AADJICL2H2NLWAGLOMMTSYOVTXWYP6Q4T", Weight: 100},
		{Address: "GAVJHRCK5CEFE3MHL4JALMNX35Z5NLUIODSJIC44VRRLQQZGTDJWANV4", Weight: 255},
	}
	signers := SignersFromHorizon(horizonSigners)
	assert.Equal(t, wantSigners, signers)
}
