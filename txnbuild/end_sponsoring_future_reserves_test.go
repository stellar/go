package txnbuild

import "testing"

func TestEndSponsoringFutureReservesRoundTrip(t *testing.T) {
	roundTrip(t, []Operation{&EndSponsoringFutureReserves{}})
}
