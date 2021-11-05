package strkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	cases := []struct {
		Name                string
		Address             string
		ExpectedVersionByte VersionByte
	}{
		{
			Name:                "AccountID",
			Address:             "GA3D5KRYM6CB7OWQ6TWYRR3Z4T7GNZLKERYNZGGA5SOAOPIFY6YQHES5",
			ExpectedVersionByte: VersionByteAccountID,
		},
		{
			Name:                "Seed",
			Address:             "SBU2RRGLXH3E5CQHTD3ODLDF2BWDCYUSSBLLZ5GNW7JXHDIYKXZWHOKR",
			ExpectedVersionByte: VersionByteSeed,
		},
		{
			Name:                "HashTx",
			Address:             "TBU2RRGLXH3E5CQHTD3ODLDF2BWDCYUSSBLLZ5GNW7JXHDIYKXZWHXL7",
			ExpectedVersionByte: VersionByteHashTx,
		},
		{
			Name:                "HashX",
			Address:             "XBU2RRGLXH3E5CQHTD3ODLDF2BWDCYUSSBLLZ5GNW7JXHDIYKXZWGTOG",
			ExpectedVersionByte: VersionByteHashX,
		},
		{
			Name:                "Other (0x60)",
			Address:             "MBU2RRGLXH3E5CQHTD3ODLDF2BWDCYUSSBLLZ5GNW7JXHDIYKXZWGTOG",
			ExpectedVersionByte: VersionByte(0x60),
		},
	}

	for _, kase := range cases {
		actual, err := Version(kase.Address)
		if assert.NoError(t, err, "An error occured decoding case %s", kase.Name) {
			assert.Equal(t, kase.ExpectedVersionByte, actual, "Output mismatch in case %s", kase.Name)
		}
	}
}

func TestIsValidEd25519PublicKey(t *testing.T) {
	validKey := "GDWZCOEQRODFCH6ISYQPWY67L3ULLWS5ISXYYL5GH43W7YFMTLB65PYM"
	isValid := IsValidEd25519PublicKey(validKey)
	assert.Equal(t, true, isValid)

	invalidKey := "GDWZCOEQRODFCH6ISYQPWY67L3ULLWS5ISXYYL5GH43W7Y"
	isValid = IsValidEd25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = ""
	isValid = IsValidEd25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = "SBCVMMCBEDB64TVJZFYJOJAERZC4YVVUOE6SYR2Y76CBTENGUSGWRRVO"
	isValid = IsValidEd25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = "MAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSAAAAAAAAAAE2LP26"
	isValid = IsValidEd25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)
}

func TestIsValidMuxedAccountMed25519PublicKey(t *testing.T) {
	validKey := "MAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSAAAAAAAAAAE2LP26"
	isValid := IsValidMuxedAccountMed25519PublicKey(validKey)
	assert.Equal(t, true, isValid)

	invalidKey := "MAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSAAAAAA"
	isValid = IsValidMuxedAccountMed25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = "GDWZCOEQRODFCH6ISYQPWY67L3ULLWS5ISXYYL5GH43W7YFMTLB65PYM"
	isValid = IsValidMuxedAccountMed25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = ""
	isValid = IsValidMuxedAccountMed25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = "SBCVMMCBEDB64TVJZFYJOJAERZC4YVVUOE6SYR2Y76CBTENGUSGWRRVO"
	isValid = IsValidMuxedAccountMed25519PublicKey(invalidKey)
	assert.Equal(t, false, isValid)
}

func TestIsValidEd25519SecretSeed(t *testing.T) {
	validKey := "SBCVMMCBEDB64TVJZFYJOJAERZC4YVVUOE6SYR2Y76CBTENGUSGWRRVO"
	isValid := IsValidEd25519SecretSeed(validKey)
	assert.Equal(t, true, isValid)

	invalidKey := "SBCVMMCBEDB64TVJZFYJOJAERZC4YVVUOE6SYR2Y76CBTENGUSG"
	isValid = IsValidEd25519SecretSeed(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = ""
	isValid = IsValidEd25519SecretSeed(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = "GDWZCOEQRODFCH6ISYQPWY67L3ULLWS5ISXYYL5GH43W7YFMTLB65PYM"
	isValid = IsValidEd25519SecretSeed(invalidKey)
	assert.Equal(t, false, isValid)

	invalidKey = "MAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSAAAAAAAAAAE2LP26"
	isValid = IsValidEd25519SecretSeed(invalidKey)
	assert.Equal(t, false, isValid)
}

func TestMuxedAccount_id(t *testing.T) {
	muxed := MuxedAccount{}
	assert.Equal(t, uint64(0), muxed.ID())

	muxed = MuxedAccount{id: uint64(9223372036854775808)}
	assert.Equal(t, uint64(9223372036854775808), muxed.ID())
}

func TestMuxedAccount_address(t *testing.T) {
	muxed := MuxedAccount{}
	publicKey, err := muxed.Address()
	assert.EqualError(t, err, "muxed account has no ed25519 key")
	assert.Empty(t, publicKey)

	muxed = MuxedAccount{ed25519: [32]byte{63, 12, 52, 191, 147, 173, 13, 153, 113, 208, 76, 204, 144, 247, 5, 81, 28, 131, 138, 173, 151, 52, 164, 162, 251, 13, 122, 3, 252, 127, 232, 154}}
	publicKey, err = muxed.Address()
	assert.NoError(t, err)
	assert.Equal(t, "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ", publicKey)
}

func TestMuxedAccount_muxedAddress(t *testing.T) {
	muxed := MuxedAccount{}
	publicKey, err := muxed.MuxedAddress()
	assert.EqualError(t, err, "muxed account has no ed25519 key")
	assert.Empty(t, publicKey)

	muxed = MuxedAccount{
		id:      uint64(9223372036854775808),
		ed25519: [32]byte{63, 12, 52, 191, 147, 173, 13, 153, 113, 208, 76, 204, 144, 247, 5, 81, 28, 131, 138, 173, 151, 52, 164, 162, 251, 13, 122, 3, 252, 127, 232, 154},
	}
	publicKey, err = muxed.MuxedAddress()
	assert.NoError(t, err)
	assert.Equal(t, "MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK", publicKey)
}
