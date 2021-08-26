package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
)

func NewProtocol18Test(t *testing.T) *integration.Test {
	config := integration.Config{ProtocolVersion: 18}
	return integration.NewTest(t, config)
}

func TestProtocol18Basics(t *testing.T) {
	tt := assert.New(t)
	itest := NewProtocol18Test(t)
	master := itest.Master()

	t.Run("Sanity", func(t *testing.T) {
		root, err := itest.Client().Root()
		tt.NoError(err)
		tt.LessOrEqual(int32(18), root.CoreSupportedProtocolVersion)
		tt.Equal(int32(18), root.CurrentProtocolVersion)

		// Submit a simple tx
		op := txnbuild.Payment{
			Destination: master.Address(),
			Amount:      "10",
			Asset:       txnbuild.NativeAsset{},
		}

		txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &op)
		tt.Equal(master.Address(), txResp.Account)
		tt.Equal("1", txResp.AccountSequence)
	})
}
