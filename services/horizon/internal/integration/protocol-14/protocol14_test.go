package integration

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

var protocol14Config = test.IntegrationConfig{ProtocolVersion: 14}

func TestProtocol14Basics(t *testing.T) {
	tt := assert.New(t)

	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()
	master := itest.Master()

	root, err := itest.Client().Root()
	tt.NoError(err)
	tt.Equal(int32(14), root.CoreSupportedProtocolVersion)
	tt.Equal(int32(14), root.CurrentProtocolVersion)

	// Submit a simple tx
	op := txnbuild.Payment{
		Destination: master.Address(),
		Amount:      "10",
		Asset:       txnbuild.NativeAsset{},
	}

	txResp := itest.MustSubmitOperations(itest.MasterAccount(), master, &op)
	tt.Equal(master.Address(), txResp.Account)
	tt.Equal("1", txResp.AccountSequence)
}
