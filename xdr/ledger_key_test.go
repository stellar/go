package xdr_test

import (
	"encoding/base64"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestLedgerKeyTrustLineBinaryMaxLength(t *testing.T) {
	key := &xdr.LedgerKey{}
	err := key.SetTrustline(
		xdr.MustAddress("GBFLTCDLOE6YQ74B66RH3S2UW5I2MKZ5VLTM75F4YMIWUIXRIFVNRNIF"),
		xdr.MustNewCreditAsset("123456789012", "GBFLTCDLOE6YQ74B66RH3S2UW5I2MKZ5VLTM75F4YMIWUIXRIFVNRNIF"),
	)
	assert.NoError(t, err)

	compressed, err := key.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, len(compressed), 92)
	bcompressed := base64.StdEncoding.EncodeToString(compressed)
	assert.Equal(t, len(bcompressed), 124)
}
