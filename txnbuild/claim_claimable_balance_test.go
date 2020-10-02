package txnbuild

import (
	"testing"
)

func TestClaimClaimableBalanceRoundTrip(t *testing.T) {
	claimClaimableBalance := &ClaimClaimableBalance{
		BalanceID: "00000000929b20b72e5890ab51c24f1cc46fa01c4f318d8d33367d24dd614cfdf5491072",
	}

	roundTrip(t, []Operation{claimClaimableBalance})
}
