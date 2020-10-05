package txnbuild

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func roundTrip(t *testing.T, operations []Operation) {
	kp1 := newKeypair1()
	sourceAccount := NewSimpleAccount(kp1.Address(), int64(9606132444168199))

	tx, err := NewTransaction(
		TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    operations,
			Timebounds:    NewInfiniteTimeout(),
			BaseFee:       MinBaseFee,
		},
	)
	assert.NoError(t, err)

	var b64 string
	b64, err = tx.Base64()
	assert.NoError(t, err)

	var parsedTx *GenericTransaction
	parsedTx, err = TransactionFromXDR(b64)
	assert.NoError(t, err)
	var ok bool
	tx, ok = parsedTx.Transaction()
	assert.True(t, ok)

	for i := 0; i < len(operations); i++ {
		assert.Equal(t, operations[i], tx.Operations()[i])
	}
}

func TestBeginSponsoringFutureReservesRoundTrip(t *testing.T) {
	beginSponsoring := &BeginSponsoringFutureReserves{
		SponsoredID: newKeypair1().Address(),
	}

	roundTrip(t, []Operation{beginSponsoring})
}
