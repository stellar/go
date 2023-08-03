package integration

import (
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/require"
)

func TestBumpFootPrintExpiration(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion:  20,
		EnableSorobanRPC: true,
	})

	// establish which account will be contract owner, and load it's current seq
	sourceAccount, err := itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})
	require.NoError(t, err)

	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), add_u64_contract)
	preFlightOp, minFee := itest.PreflightHostFunctions(&sourceAccount, *installContractOp)
	tx := itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)

	_, err = itest.Client().TransactionDetail(tx.Hash)
	require.NoError(t, err)

	sourceAccount, err = itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})
	require.NoError(t, err)

	bumpFootPrint := txnbuild.BumpFootprintExpiration{
		LedgersToExpire: 10000,
		SourceAccount:   "",
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: &xdr.SorobanTransactionData{
				Resources: xdr.SorobanResources{
					Footprint: xdr.LedgerFootprint{
						ReadOnly:  preFlightOp.Ext.SorobanData.Resources.Footprint.ReadWrite,
						ReadWrite: nil,
					},
				},
				RefundableFee: 0,
			},
		},
	}
	bumpFootPrint, minFee = itest.PreflightBumpFootprintExpiration(&sourceAccount, bumpFootPrint)
	tx = itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &bumpFootPrint)

	ops, err := itest.Client().Operations(horizonclient.OperationRequest{ForTransaction: tx.Hash})
	require.NoError(t, err)
	require.Len(t, ops.Embedded.Records, 1)

	op := ops.Embedded.Records[0].(operations.BumpFootprintExpiration)
	require.Equal(t, uint32(10000), op.LedgersToExpire)
}
