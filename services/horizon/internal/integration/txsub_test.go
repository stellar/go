package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
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

func TestTxSubLimitsBodySize(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion:  20,
		EnableSorobanRPC: true,
		HorizonEnvironment: map[string]string{
			"MAX_HTTP_REQUEST_SIZE": "5000",
		},
	})

	// establish which account will be contract owner, and load it's current seq
	sourceAccount, err := itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})
	require.NoError(t, err)

	contract := []byte(strings.Repeat("a", 10*1024))
	installContractOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
			Wasm: &contract,
		},
		SourceAccount: itest.Master().Address(),
	}
	preFlightOp, minFee := itest.PreflightHostFunctions(&sourceAccount, *installContractOp)
	_, err = itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)
	assert.EqualError(
		t, err,
		"horizon error: \"Transaction Malformed\" - check horizon.Error.Problem for more information",
	)

	sourceAccount, err = itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})
	require.NoError(t, err)

	contract = []byte(strings.Repeat("a", 2*1024))
	installContractOp = &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
			Wasm: &contract,
		},
		SourceAccount: itest.Master().Address(),
	}
	preFlightOp, minFee = itest.PreflightHostFunctions(&sourceAccount, *installContractOp)
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)
	require.NoError(t, err)
	require.True(t, tx.Successful)
}
