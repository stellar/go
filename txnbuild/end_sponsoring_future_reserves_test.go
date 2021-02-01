package txnbuild

import "testing"

func TestEndSponsoringFutureReservesRoundTrip(t *testing.T) {
	testOperationsMarshallingRoundtrip(t, []Operation{&EndSponsoringFutureReserves{}})
}
