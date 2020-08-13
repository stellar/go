package integration

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

var protocol14Config = test.IntegrationConfig{ProtocolVersion: 14}

func TestProtocol14Sample(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	root, err := itest.Client().Root()
	assert.NoError(t, err)
	assert.Equal(t, int32(14), root.CoreSupportedProtocolVersion)
	assert.Equal(t, int32(14), root.CurrentProtocolVersion)

	// Submit a simple tx
	master := itest.Master()
	op := txnbuild.Payment{
		Destination: master.Address(),
		Amount:      "10",
		Asset:       txnbuild.NativeAsset{},
	}
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: master.Address(),
				Sequence:  1,
			},
			Operations: []txnbuild.Operation{&op},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	assert.NoError(t, err)
	tx, err = tx.Sign(test.IntegrationNetworkPassphrase, itest.Master().(*keypair.Full))
	assert.NoError(t, err)

	txResp, err := itest.Client().SubmitTransaction(tx)
	assert.NoError(t, err)
	assert.Equal(t, itest.Master().Address(), txResp.Account)
	assert.Equal(t, "1", txResp.AccountSequence)
}
