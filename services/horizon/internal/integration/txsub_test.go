package integration

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

func TestTxSub(t *testing.T) {
	tt := assert.New(t)

	t.Run("transaction submission is successful when DISABLE_TX_SUB=false", func(t *testing.T) {
		itest := integration.NewTest(t, integration.Config{})
		master := itest.Master()

		op := txnbuild.Payment{
			Destination: master.Address(),
			Amount:      "10",
			Asset:       txnbuild.NativeAsset{},
		}

		txResp, err := itest.SubmitOperations(itest.MasterAccount(), master, &op)
		assert.NoError(t, err)

		var seq int64
		tt.Equal(itest.MasterAccount().GetAccountID(), txResp.Account)
		seq, err = itest.MasterAccount().GetSequenceNumber()
		assert.NoError(t, err)
		tt.Equal(seq, txResp.AccountSequence)
		t.Logf("Done")
	})

	t.Run("transaction submission is not successful when DISABLE_TX_SUB=true", func(t *testing.T) {
		itest := integration.NewTest(t, integration.Config{
			HorizonEnvironment: map[string]string{
				"DISABLE_TX_SUB": "true",
			},
		})
		master := itest.Master()

		op := txnbuild.Payment{
			Destination: master.Address(),
			Amount:      "10",
			Asset:       txnbuild.NativeAsset{},
		}

		_, err := itest.SubmitOperations(itest.MasterAccount(), master, &op)
		assert.Error(t, err)
	})
}
