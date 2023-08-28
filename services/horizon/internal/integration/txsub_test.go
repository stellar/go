package integration

import (
	"sync"
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
		keys, accounts := itest.CreateAccounts(1, "1000")

		var wg sync.WaitGroup
		wg.Add(1)

		seq, err := accounts[0].GetSequenceNumber()
		assert.NoError(t, err)

		account := txnbuild.SimpleAccount{
			AccountID: keys[0].Address(),
			Sequence:  seq,
		}

		go func(account txnbuild.SimpleAccount) {
			defer wg.Done()

			op := txnbuild.Payment{
				Destination: master.Address(),
				Amount:      "10",
				Asset:       txnbuild.NativeAsset{},
			}

			txResp := itest.MustSubmitOperations(&account, keys[0], &op)

			tt.Equal(accounts[0].GetAccountID(), txResp.Account)
			seq, err := account.GetSequenceNumber()
			assert.NoError(t, err)
			tt.Equal(seq, txResp.AccountSequence)
			t.Logf("Done")
		}(account)

		wg.Wait()
	})

	t.Run("transaction submission is not successful when DISABLE_TX_SUB=true", func(t *testing.T) {
		itest := integration.NewTest(t, integration.Config{
			HorizonEnvironment: map[string]string{
				"DISABLE_TX_SUB": "true",
			},
		})
		assert.PanicsWithError(t, "horizon error: \"Transaction Submission Disabled\" - check horizon.Error.Problem for more information", func() {
			itest.CreateAccounts(1, "1000")
		})
	})
}
