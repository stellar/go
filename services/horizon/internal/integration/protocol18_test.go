package integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
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

func TestLiquidityPoolHappyPath(t *testing.T) {
	tt := assert.New(t)
	itest := NewProtocol18Test(t)
	master := itest.Master()

	keys, accounts := itest.CreateAccounts(1, "1000")
	shareKeys, shareAccount := keys[0], accounts[0]

	itest.MustSubmitMultiSigOperations(shareAccount, []*keypair.Full{shareKeys, master},
		&txnbuild.ChangeTrust{
			Line: txnbuild.ChangeTrustAssetWrapper{
				Asset: txnbuild.CreditAsset{
					Code:   "USD",
					Issuer: master.Address(),
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.ChangeTrust{
			Line: txnbuild.LiquidityPoolShareChangeTrustAsset{
				LiquidityPoolParameters: txnbuild.LiquidityPoolParameters{
					AssetA: txnbuild.NativeAsset{},
					AssetB: txnbuild.CreditAsset{
						Code:   "USD",
						Issuer: master.Address(),
					},
					Fee: 30,
				},
			},
			Limit: txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.Payment{
			SourceAccount: master.Address(),
			Destination:   shareAccount.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   "USD",
				Issuer: master.Address(),
			},
			Amount: "1000",
		},
	)

	poolID, err := xdr.NewPoolId(
		xdr.MustNewNativeAsset(),
		xdr.MustNewCreditAsset("USD", master.Address()),
		30,
	)
	tt.NoError(err)
	poolIDHexString := xdr.Hash(poolID).HexString()

	pools, err := itest.Client().LiquidityPools(horizonclient.LiquidityPoolsRequest{})
	tt.NoError(err)
	tt.Len(pools.Embedded.Records, 1)

	pool := pools.Embedded.Records[0]
	tt.Equal(poolIDHexString, pool.ID)
	tt.Equal(uint32(30), pool.FeeBP)
	tt.Equal("constant_product", pool.Type)
	tt.Equal(uint64(0), pool.TotalShares)
	tt.Equal(uint64(1), pool.TotalTrustlines)

	tt.Equal("0.0000000", pool.Reserves[0].Amount)
	tt.Equal("native", pool.Reserves[0].Asset)
	tt.Equal("0.0000000", pool.Reserves[1].Amount)
	tt.Equal(fmt.Sprintf("USD:%s", master.Address()), pool.Reserves[1].Asset)

	itest.MustSubmitOperations(shareAccount, shareKeys,
		&txnbuild.LiquidityPoolDeposit{
			LiquidityPoolID: [32]byte(poolID),
			MaxAmountA:      "400",
			MaxAmountB:      "777",
			MinPrice:        "0.5",
			MaxPrice:        "2",
		},
	)

	pool, err = itest.Client().LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: poolIDHexString,
	})
	tt.NoError(err)

	tt.Equal(poolIDHexString, pool.ID)
	tt.Equal(uint64(1), pool.TotalTrustlines)

	tt.Equal("400.0000000", pool.Reserves[0].Amount)
	tt.Equal("native", pool.Reserves[0].Asset)
	tt.Equal("777.0000000", pool.Reserves[1].Amount)
	tt.Equal(fmt.Sprintf("USD:%s", master.Address()), pool.Reserves[1].Asset)

	itest.MustSubmitOperations(shareAccount, shareKeys,
		&txnbuild.LiquidityPoolWithdraw{
			LiquidityPoolID: [32]byte(poolID),
			Amount:          amount.StringFromInt64(int64(pool.TotalShares)),
			MinAmountA:      "10",
			MinAmountB:      "20",
		},
	)

	itest.MustSubmitOperations(shareAccount, shareKeys,
		// Clear trustline...
		&txnbuild.Payment{
			Asset: txnbuild.CreditAsset{
				Code:   "USD",
				Issuer: master.Address(),
			},
			Amount:      "1000",
			Destination: master.Address(),
		},
		// ...and remove it. It should also remove LP.
		&txnbuild.ChangeTrust{
			Line: txnbuild.LiquidityPoolShareChangeTrustAsset{
				LiquidityPoolParameters: txnbuild.LiquidityPoolParameters{
					AssetA: txnbuild.NativeAsset{},
					AssetB: txnbuild.CreditAsset{
						Code:   "USD",
						Issuer: master.Address(),
					},
					Fee: 30,
				},
			},
			Limit: "0",
		},
	)

	ops, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForLiquidityPool: poolIDHexString,
	})
	tt.NoError(err)

	// We expect 4 ops for this liquidity pool:
	// 1. change_trust creating a trust to LP.
	// 2. liquidity_pool_deposit.
	// 3. liquidity_pool_withdraw.
	// 4. change_trust removing a trust to LP.
	tt.Len(ops.Embedded.Records, 4)

	op1 := (ops.Embedded.Records[0]).(operations.ChangeTrust)
	tt.Equal("change_trust", op1.Type)
	tt.Equal("liquidity_pool_shares", op1.Asset.Type)
	tt.Equal(poolIDHexString, op1.LiquidityPoolID)
	tt.Equal("922337203685.4775807", op1.Limit)

	op2 := (ops.Embedded.Records[1]).(operations.LiquidityPoolDeposit)
	tt.Equal("liquidity_pool_deposit", op2.Type)
	tt.Equal(poolIDHexString, op2.LiquidityPoolID)
	tt.Equal("0.5000000", op2.MinPrice)
	tt.Equal("2.0000000", op2.MaxPrice)
	tt.Equal("native", op2.ReservesDeposited[0].Asset)
	tt.Equal("400.0000000", op2.ReservesDeposited[0].Amount)
	tt.Equal(fmt.Sprintf("USD:%s", master.Address()), op2.ReservesDeposited[1].Asset)
	tt.Equal("777.0000000", op2.ReservesDeposited[1].Amount)
	tt.Equal(uint64(5574943946), op2.SharesReceived)

	op3 := (ops.Embedded.Records[2]).(operations.LiquidityPoolWithdraw)
	tt.Equal("liquidity_pool_withdraw", op3.Type)
	tt.Equal(poolIDHexString, op3.LiquidityPoolID)

	tt.Equal("native", op3.ReservesMin[0].Asset)
	tt.Equal("10.0000000", op3.ReservesMin[0].Amount)
	tt.Equal(fmt.Sprintf("USD:%s", master.Address()), op3.ReservesMin[1].Asset)
	tt.Equal("20.0000000", op3.ReservesMin[1].Amount)

	tt.Equal("native", op3.ReservesReceived[0].Asset)
	tt.Equal("400.0000000", op3.ReservesReceived[0].Amount)
	tt.Equal(fmt.Sprintf("USD:%s", master.Address()), op3.ReservesReceived[1].Asset)
	tt.Equal("777.0000000", op3.ReservesReceived[1].Amount)

	tt.Equal(uint64(5574943946), op3.Shares)

	op4 := (ops.Embedded.Records[3]).(operations.ChangeTrust)
	tt.Equal("change_trust", op4.Type)
	tt.Equal("liquidity_pool_shares", op4.Asset.Type)
	tt.Equal(poolIDHexString, op4.LiquidityPoolID)
	tt.Equal("0.0000000", op4.Limit)
}
