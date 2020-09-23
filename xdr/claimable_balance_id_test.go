package xdr_test

import (
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestClaimableBalanceIdString(t *testing.T) {
	tt := assert.New(t)

	balanceID := xdr.ClaimableBalanceId{}
	err := xdr.SafeUnmarshalHex("00000000da0d57da7d4850e7fc10d2a9d0ebc731f7afb40574c03395b17d49149b91f5be", &balanceID)
	tt.NoError(err)
	expected := "0da0d57da7d4850e7fc10d2a9d0ebc731f7afb40574c03395b17d49149b91f5be"

	id, err := balanceID.String()
	tt.NoError(err)
	tt.Equal(expected, id)
}
