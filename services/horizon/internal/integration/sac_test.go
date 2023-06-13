package integration

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

const sac_contract = "soroban_sac_test.wasm"

// Tests use precompiled wasm bin files that are added to the testdata directory.
// Refer to ./services/horizon/internal/integration/contracts/README.md on how to recompile
// contract code if needed to new wasm.

func TestContractMintToAccount(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	recipientKp, recipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(recipientKp, recipient, txnbuild.MustAssetFromXDR(asset))

	_, mintTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "20", accountAddressParam(recipient.GetAccountID())),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("20"))
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("20"),
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})

	fx := getTxEffects(itest, mintTx, asset)
	require.Len(t, fx, 1)
	creditEffect := assertContainsEffect(t, fx,
		effects.EffectAccountCredited)[0].(effects.AccountCredited)
	assert.Equal(t, recipientKp.Address(), creditEffect.Account)
	assert.Equal(t, issuer, creditEffect.Asset.Issuer)
	assert.Equal(t, code, creditEffect.Asset.Code)
	assert.Equal(t, "20.0000000", creditEffect.Amount)
	assertEventPayments(itest, mintTx, asset, "", recipient.GetAccountID(), "mint", "20.0000000")

	otherRecipientKp, otherRecipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(otherRecipientKp, otherRecipient, txnbuild.MustAssetFromXDR(asset))

	// calling transfer from the issuer account will also mint the asset
	_, transferTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		transfer(itest, issuer, asset, "30", accountAddressParam(otherRecipient.GetAccountID())),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("20"))
	assertContainsBalance(itest, otherRecipientKp, issuer, code, amount.MustParse("30"))

	fx = getTxEffects(itest, transferTx, asset)
	assert.Len(t, fx, 2)
	assertContainsEffect(t, fx,
		effects.EffectAccountCredited,
		effects.EffectAccountDebited)
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      2,
		balanceAccounts:  amount.MustParse("50"),
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})
}

func TestContractMintToContract(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	// Create recipient contract
	recipientContractID, _ := mustCreateAndInstallContract(itest, itest.Master(), "a1", add_u64_contract)
	strkeyRecipientContractID, err := strkey.Encode(strkey.VersionByteContract, recipientContractID[:])
	assert.NoError(t, err)

	mintAmount := xdr.Int128Parts{Lo: math.MaxUint64 - 3, Hi: math.MaxInt64}
	_, mintTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mintWithAmt(
			itest,
			issuer, asset,
			i128Param(int64(mintAmount.Hi), uint64(mintAmount.Lo)),
			contractAddressParam(recipientContractID)),
	)
	assertContainsEffect(t, getTxEffects(itest, mintTx, asset),
		effects.EffectContractCredited)

	balanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		contractBalance(itest, issuer, asset, recipientContractID),
	)
	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvI128, balanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(math.MaxUint64-3), (*balanceAmount.I128).Lo)
	assert.Equal(itest.CurrentTest(), xdr.Int64(math.MaxInt64), (*balanceAmount.I128).Hi)
	assertEventPayments(itest, mintTx, asset, "", strkeyRecipientContractID, "mint", amount.String128(mintAmount))

	// calling transfer from the issuer account will also mint the asset
	_, transferTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		transferWithAmount(itest, issuer, asset, i128Param(0, 3), contractAddressParam(recipientContractID)),
	)

	assertContainsEffect(t, getTxEffects(itest, transferTx, asset),
		effects.EffectAccountDebited,
		effects.EffectContractCredited)

	balanceAmount, _ = assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		contractBalance(itest, issuer, asset, recipientContractID),
	)
	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvI128, balanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(math.MaxUint64), (*balanceAmount.I128).Lo)
	assert.Equal(itest.CurrentTest(), xdr.Int64(math.MaxInt64), (*balanceAmount.I128).Hi)

	// 2^127 - 1
	balanceContracts := new(big.Int).Lsh(big.NewInt(1), 127)
	balanceContracts.Sub(balanceContracts, big.NewInt(1))
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      0,
		balanceAccounts:  0,
		numContracts:     1,
		balanceContracts: balanceContracts,
		contractID:       stellarAssetContractID(itest, asset),
	})
}

func TestContractTransferBetweenAccounts(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	recipientKp, recipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(recipientKp, recipient, txnbuild.MustAssetFromXDR(asset))

	itest.MustSubmitOperations(
		itest.MasterAccount(),
		itest.Master(),
		&txnbuild.Payment{
			SourceAccount: issuer,
			Destination:   recipient.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   code,
				Issuer: issuer,
			},
			Amount: "1000",
		},
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("1000"))
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("1000"),
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})

	otherRecipientKp, otherRecipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(otherRecipientKp, otherRecipient, txnbuild.MustAssetFromXDR(asset))

	_, transferTx := assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		transfer(itest, recipientKp.Address(), asset, "30", accountAddressParam(otherRecipient.GetAccountID())),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("970"))
	assertContainsBalance(itest, otherRecipientKp, issuer, code, amount.MustParse("30"))

	fx := getTxEffects(itest, transferTx, asset)
	assert.NotEmpty(t, fx)
	assertContainsEffect(t, fx, effects.EffectAccountCredited, effects.EffectAccountDebited)
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      2,
		balanceAccounts:  amount.MustParse("1000"),
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, transferTx, asset, recipientKp.Address(), otherRecipient.GetAccountID(), "transfer", "30.0000000")
}

func TestContractTransferBetweenAccountAndContract(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	issuer := itest.Master().Address()
	code := "USDLONG"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	recipientKp, recipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(recipientKp, recipient, txnbuild.MustAssetFromXDR(asset))

	itest.MustSubmitOperations(
		itest.MasterAccount(),
		itest.Master(),
		&txnbuild.Payment{
			SourceAccount: issuer,
			Destination:   recipient.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   code,
				Issuer: issuer,
			},
			Amount: "1000",
		},
	)

	// Create recipient contract
	recipientContractID, recipientContractHash := mustCreateAndInstallContract(itest, itest.Master(), "a1", sac_contract)
	strkeyRecipientContractID, err := strkey.Encode(strkey.VersionByteContract, recipientContractID[:])
	assert.NoError(t, err)

	// init recipient contract with the asset contract id
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		initAssetContract(itest, issuer, asset, recipientContractID, recipientContractHash),
	)

	// Add funds to recipient contract
	_, mintTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "1000", contractAddressParam(recipientContractID)),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("1000"))
	assertContainsEffect(t, getTxEffects(itest, mintTx, asset),
		effects.EffectContractCredited)

	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("1000"),
		numContracts:     1,
		balanceContracts: big.NewInt(int64(amount.MustParse("1000"))),
		contractID:       stellarAssetContractID(itest, asset),
	})

	// transfer from account to contract
	_, transferTx := assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		transfer(itest, recipientKp.Address(), asset, "30", contractAddressParam(recipientContractID)),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("970"))
	assertContainsEffect(t, getTxEffects(itest, transferTx, asset),
		effects.EffectAccountDebited, effects.EffectContractCredited)
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("970"),
		numContracts:     1,
		balanceContracts: big.NewInt(int64(amount.MustParse("1030"))),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, transferTx, asset, recipientKp.Address(), strkeyRecipientContractID, "transfer", "30.0000000")

	// transfer from contract to account
	_, transferTx = assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		transferFromContract(itest, recipientKp.Address(), asset, recipientContractID, recipientContractHash, "500", accountAddressParam(recipient.GetAccountID())),
	)
	assertContainsEffect(t, getTxEffects(itest, transferTx, asset),
		effects.EffectContractDebited, effects.EffectAccountCredited)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("1470"))
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("1470"),
		numContracts:     1,
		balanceContracts: big.NewInt(int64(amount.MustParse("530"))),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, transferTx, asset, strkeyRecipientContractID, recipientKp.Address(), "transfer", "500.0000000")

	balanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		contractBalance(itest, issuer, asset, recipientContractID),
	)
	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvI128, balanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(5300000000), (*balanceAmount.I128).Lo)
	assert.Equal(itest.CurrentTest(), xdr.Int64(0), (*balanceAmount.I128).Hi)
}

func TestContractTransferBetweenContracts(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the token contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	// Create recipient contract
	recipientContractID, _ := mustCreateAndInstallContract(itest, itest.Master(), "a1", sac_contract)
	strkeyRecipientContractID, err := strkey.Encode(strkey.VersionByteContract, recipientContractID[:])
	assert.NoError(t, err)

	// Create emitter contract
	emitterContractID, emitterContractHash := mustCreateAndInstallContract(itest, itest.Master(), "a2", sac_contract)
	strkeyEmitterContractID, err := strkey.Encode(strkey.VersionByteContract, emitterContractID[:])
	assert.NoError(t, err)

	// init emitter contract with the asset contract id
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		initAssetContract(itest, issuer, asset, emitterContractID, emitterContractHash),
	)

	// Add funds to emitter contract
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "1000", contractAddressParam(emitterContractID)),
	)

	// Transfer funds from emitter to recipient
	_, transferTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		transferFromContract(itest, issuer, asset, emitterContractID, emitterContractHash, "10", contractAddressParam(recipientContractID)),
	)
	assertContainsEffect(t, getTxEffects(itest, transferTx, asset),
		effects.EffectContractCredited, effects.EffectContractDebited)

	// Check balances of emitter and recipient
	emitterBalanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		contractBalance(itest, issuer, asset, emitterContractID),
	)
	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvI128, emitterBalanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(9900000000), (*emitterBalanceAmount.I128).Lo)
	assert.Equal(itest.CurrentTest(), xdr.Int64(0), (*emitterBalanceAmount.I128).Hi)

	recipientBalanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		contractBalance(itest, issuer, asset, recipientContractID),
	)
	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvI128, recipientBalanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(100000000), (*recipientBalanceAmount.I128).Lo)
	assert.Equal(itest.CurrentTest(), xdr.Int64(0), (*recipientBalanceAmount.I128).Hi)

	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      0,
		balanceAccounts:  0,
		numContracts:     2,
		balanceContracts: big.NewInt(int64(amount.MustParse("1000"))),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, transferTx, asset, strkeyEmitterContractID, strkeyRecipientContractID, "transfer", "10.0000000")
}

func TestContractBurnFromAccount(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	recipientKp, recipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(recipientKp, recipient, txnbuild.MustAssetFromXDR(asset))

	itest.MustSubmitOperations(
		itest.MasterAccount(),
		itest.Master(),
		&txnbuild.Payment{
			SourceAccount: issuer,
			Destination:   recipient.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   code,
				Issuer: issuer,
			},
			Amount: "1000",
		},
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("1000"))
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("1000"),
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})

	_, burnTx := assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		burn(itest, recipientKp.Address(), asset, "500"),
	)

	fx := getTxEffects(itest, burnTx, asset)
	assert.Len(t, fx, 1)
	burnEffect := assertContainsEffect(t, fx,
		effects.EffectAccountDebited)[0].(effects.AccountDebited)

	assert.Equal(t, issuer, burnEffect.Asset.Issuer)
	assert.Equal(t, code, burnEffect.Asset.Code)
	assert.Equal(t, "500.0000000", burnEffect.Amount)
	assert.Equal(t, recipientKp.Address(), burnEffect.Account)
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("500"),
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, burnTx, asset, recipientKp.Address(), "", "burn", "500.0000000")
}

func TestContractBurnFromContract(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	// Create recipient contract
	recipientContractID, recipientContractHash := mustCreateAndInstallContract(itest, itest.Master(), "a1", sac_contract)
	strkeyRecipientContractID, err := strkey.Encode(strkey.VersionByteContract, recipientContractID[:])
	assert.NoError(t, err)
	// init contract with asset contract id
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		initAssetContract(itest, issuer, asset, recipientContractID, recipientContractHash),
	)

	// Add funds to recipient contract
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "1000", contractAddressParam(recipientContractID)),
	)

	// Burn funds
	_, burnTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		burnSelf(itest, issuer, asset, recipientContractID, recipientContractHash, "10"),
	)

	balanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		contractBalance(itest, issuer, asset, recipientContractID),
	)

	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvI128, balanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(9900000000), (*balanceAmount.I128).Lo)
	assert.Equal(itest.CurrentTest(), xdr.Int64(0), (*balanceAmount.I128).Hi)

	assertContainsEffect(t, getTxEffects(itest, burnTx, asset),
		effects.EffectContractDebited)

	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      0,
		balanceAccounts:  0,
		numContracts:     1,
		balanceContracts: big.NewInt(int64(amount.MustParse("990"))),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, burnTx, asset, strkeyRecipientContractID, "", "burn", "10.0000000")
}

func TestContractClawbackFromAccount(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	// Give the master account the revocable flag (needed to set the clawback flag)
	// and the clawback flag
	setRevocableFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
			txnbuild.AuthClawbackEnabled,
		},
	}
	itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(), &setRevocableFlag)

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	recipientKp, recipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(recipientKp, recipient, txnbuild.MustAssetFromXDR(asset))

	itest.MustSubmitOperations(
		itest.MasterAccount(),
		itest.Master(),
		&txnbuild.Payment{
			SourceAccount: issuer,
			Destination:   recipient.GetAccountID(),
			Asset: txnbuild.CreditAsset{
				Code:   code,
				Issuer: issuer,
			},
			Amount: "1000",
		},
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("1000"))
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  amount.MustParse("1000"),
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})

	_, clawTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		clawback(itest, issuer, asset, "1000", accountAddressParam(recipientKp.Address())),
	)

	assertContainsEffect(t, getTxEffects(itest, clawTx, asset), effects.EffectAccountDebited)
	assertContainsBalance(itest, recipientKp, issuer, code, 0)
	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      1,
		balanceAccounts:  0,
		numContracts:     0,
		balanceContracts: big.NewInt(0),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, clawTx, asset, recipientKp.Address(), "", "clawback", "1000.0000000")
}

func TestContractClawbackFromContract(t *testing.T) {
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	// Give the master account the revocable flag (needed to set the clawback flag)
	// and the clawback flag
	setRevocableFlag := txnbuild.SetOptions{
		SetFlags: []txnbuild.AccountFlag{
			txnbuild.AuthRevocable,
			txnbuild.AuthClawbackEnabled,
		},
	}
	itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(), &setRevocableFlag)

	issuer := itest.Master().Address()
	code := "USD"
	asset := xdr.MustNewCreditAsset(code, issuer)

	// Create the contract
	assertInvokeHostFnSucceeds(itest, itest.Master(), createSAC(itest, issuer, asset))

	// Create recipient contract
	recipientContractID, _ := mustCreateAndInstallContract(itest, itest.Master(), "a2", sac_contract)
	strkeyRecipientContractID, err := strkey.Encode(strkey.VersionByteContract, recipientContractID[:])
	assert.NoError(itest.CurrentTest(), err)

	// Add funds to recipient contract
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "1000", contractAddressParam(recipientContractID)),
	)

	// Clawback funds
	_, clawTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		clawback(itest, issuer, asset, "10", contractAddressParam(recipientContractID)),
	)

	balanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		contractBalance(itest, issuer, asset, recipientContractID),
	)
	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvI128, balanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(9900000000), (*balanceAmount.I128).Lo)
	assert.Equal(itest.CurrentTest(), xdr.Int64(0), (*balanceAmount.I128).Hi)

	assertContainsEffect(t, getTxEffects(itest, clawTx, asset),
		effects.EffectContractDebited)

	assertAssetStats(itest, assetStats{
		code:             code,
		issuer:           issuer,
		numAccounts:      0,
		balanceAccounts:  0,
		numContracts:     1,
		balanceContracts: big.NewInt(int64(amount.MustParse("990"))),
		contractID:       stellarAssetContractID(itest, asset),
	})
	assertEventPayments(itest, clawTx, asset, strkeyRecipientContractID, "", "clawback", "10.0000000")
}

func assertContainsBalance(itest *integration.Test, acct *keypair.Full, issuer, code string, amt xdr.Int64) {
	accountResponse := itest.MustGetAccount(acct)
	if issuer == "" && code == "" {
		xlmBalance, err := accountResponse.GetNativeBalance()
		assert.NoError(itest.CurrentTest(), err)
		assert.Equal(itest.CurrentTest(), amt, amount.MustParse(xlmBalance))
	} else {
		assetBalance := accountResponse.GetCreditBalance(code, issuer)
		assert.Equal(itest.CurrentTest(), amt, amount.MustParse(assetBalance))
	}
}

type assetStats struct {
	code             string
	issuer           string
	numAccounts      int32
	balanceAccounts  xdr.Int64
	numContracts     int32
	balanceContracts *big.Int
	contractID       [32]byte
}

func assertAssetStats(itest *integration.Test, expected assetStats) {
	assets, err := itest.Client().Assets(horizonclient.AssetRequest{
		ForAssetCode:   expected.code,
		ForAssetIssuer: expected.issuer,
		Limit:          1,
	})
	assert.NoError(itest.CurrentTest(), err)

	if expected.numContracts == 0 && expected.numAccounts == 0 &&
		expected.balanceContracts.Cmp(big.NewInt(0)) == 0 && expected.balanceAccounts == 0 {
		assert.Empty(itest.CurrentTest(), assets)
		return
	}

	assert.Len(itest.CurrentTest(), assets.Embedded.Records, 1)
	asset := assets.Embedded.Records[0]
	assert.Equal(itest.CurrentTest(), expected.code, asset.Code)
	assert.Equal(itest.CurrentTest(), expected.issuer, asset.Issuer)
	assert.Equal(itest.CurrentTest(), expected.numAccounts, asset.NumAccounts)
	assert.Equal(itest.CurrentTest(), expected.numAccounts, asset.Accounts.Authorized)
	assert.Equal(itest.CurrentTest(), expected.balanceAccounts, amount.MustParse(asset.Amount))
	assert.Equal(itest.CurrentTest(), expected.numContracts, asset.NumContracts)
	parts := strings.Split(asset.ContractsAmount, ".")
	assert.Len(itest.CurrentTest(), parts, 2)
	contractsAmount, ok := new(big.Int).SetString(parts[0]+parts[1], 10)
	assert.True(itest.CurrentTest(), ok)
	assert.Equal(itest.CurrentTest(), expected.balanceContracts.String(), contractsAmount.String())
	assert.Equal(itest.CurrentTest(), strkey.MustEncode(strkey.VersionByteContract, expected.contractID[:]), asset.ContractID)
}

// assertContainsEffect checks that the list of json effects contains the given
// effect type(s) by name (no other details are checked). It returns the last
// effect matching each given type.
func assertContainsEffect(t *testing.T, fx []effects.Effect, effectTypes ...effects.EffectType) []effects.Effect {
	found := map[string]int{}
	for idx, effect := range fx {
		found[effect.GetType()] = idx
	}

	for _, type_ := range effectTypes {
		assert.Containsf(t, found, effects.EffectTypeNames[type_], "effects: %v", fx)
	}

	var rv []effects.Effect
	for _, i := range found {
		rv = append(rv, fx[i])
	}

	return rv
}

// getTxEffects returns a transaction's effects, limited to 2 because it's to be
// used for checking SAC effects.
func getTxEffects(itest *integration.Test, txHash string, asset xdr.Asset) []effects.Effect {
	t := itest.CurrentTest()
	effects, err := itest.Client().Effects(horizonclient.EffectRequest{
		ForTransaction: txHash,
		Order:          horizonclient.OrderDesc,
	})
	assert.NoError(t, err)
	result := effects.Embedded.Records

	assert.LessOrEqualf(t, len(result), 2, "txhash: %s", txHash)
	return result
}

func assertEventPayments(itest *integration.Test, txHash string, asset xdr.Asset, from string, to string, evtType string, amount string) {
	ops, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: txHash,
		Limit:          1,
	})
	assert.NoError(itest.CurrentTest(), err)
	assert.Equal(itest.CurrentTest(), 1, len(ops.Embedded.Records))
	assert.Equal(itest.CurrentTest(), ops.Embedded.Records[0].GetType(), operations.TypeNames[xdr.OperationTypeInvokeHostFunction])

	invokeHostFn := ops.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.Equal(itest.CurrentTest(), 1, len(invokeHostFn.AssetBalanceChanges))
	assetBalanceChange := invokeHostFn.AssetBalanceChanges[0]
	assert.Equal(itest.CurrentTest(), assetBalanceChange.Amount, amount)
	assert.Equal(itest.CurrentTest(), assetBalanceChange.From, from)
	assert.Equal(itest.CurrentTest(), assetBalanceChange.To, to)
	assert.Equal(itest.CurrentTest(), assetBalanceChange.Type, evtType)
	assert.Equal(itest.CurrentTest(), assetBalanceChange.Asset.Code, strings.TrimRight(asset.GetCode(), "\x00"))
	assert.Equal(itest.CurrentTest(), assetBalanceChange.Asset.Issuer, asset.GetIssuer())
}

func functionNameParam(name string) xdr.ScVal {
	contractFnParameterSym := xdr.ScSymbol(name)
	return xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &contractFnParameterSym,
	}
}

func contractIDParam(contractID xdr.Hash) xdr.ScVal {
	contractBytes := contractID[:]
	return xdr.ScVal{
		Type:  xdr.ScValTypeScvBytes,
		Bytes: (*xdr.ScBytes)(&contractBytes),
	}
}

func accountAddressParam(accountID string) xdr.ScVal {
	address := xdr.ScAddress{
		Type:      xdr.ScAddressTypeScAddressTypeAccount,
		AccountId: xdr.MustAddressPtr(accountID),
	}
	return xdr.ScVal{
		Type:    xdr.ScValTypeScvAddress,
		Address: &address,
	}
}

func contractAddressParam(contractID xdr.Hash) xdr.ScVal {
	address := xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: &contractID,
	}
	return xdr.ScVal{
		Type:    xdr.ScValTypeScvAddress,
		Address: &address,
	}
}

func i128Param(hi int64, lo uint64) xdr.ScVal {
	i128 := &xdr.Int128Parts{
		Hi: xdr.Int64(hi),
		Lo: xdr.Uint64(lo),
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvI128,
		I128: i128,
	}
}

func baseSACFootPrint(itest *integration.Test, asset xdr.Asset) []xdr.LedgerKey {
	contractID := stellarAssetContractID(itest, asset)
	metadata := xdr.ScSymbol("METADATA")
	admin := xdr.ScSymbol("Admin")
	assetInfo := xdr.ScSymbol("AssetInfo")
	adminVec := &xdr.ScVec{
		{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &admin,
		},
	}
	assetInfoVec := &xdr.ScVec{
		{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &assetInfo,
		},
	}
	masterAddress := xdr.MustAddress(itest.MasterAccount().GetAccountID())
	return []xdr.LedgerKey{
		{
			Type:    xdr.LedgerEntryTypeAccount,
			Account: &xdr.LedgerKeyAccount{AccountId: masterAddress},
		},
		{
			Type: xdr.LedgerEntryTypeContractData,
			ContractData: &xdr.LedgerKeyContractData{
				ContractId: contractID,
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvSymbol,
					Sym:  &metadata,
				},
			},
		},
		{
			Type: xdr.LedgerEntryTypeContractData,
			ContractData: &xdr.LedgerKeyContractData{
				ContractId: contractID,
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvVec,
					Vec:  &adminVec,
				},
			},
		},
		{
			Type: xdr.LedgerEntryTypeContractData,
			ContractData: &xdr.LedgerKeyContractData{
				ContractId: contractID,
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvVec,
					Vec:  &assetInfoVec,
				},
			},
		},
		{
			Type: xdr.LedgerEntryTypeContractData,
			ContractData: &xdr.LedgerKeyContractData{
				ContractId: contractID,
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvLedgerKeyContractExecutable,
				},
			},
		},
	}
}

func tokenLedgerKey(contractID xdr.Hash) xdr.LedgerKey {
	token := xdr.ScSymbol("Token")
	tokenVec := &xdr.ScVec{
		{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &token,
		},
	}
	return xdr.LedgerKey{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.LedgerKeyContractData{
			ContractId: contractID,
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvVec,
				Vec:  &tokenVec,
			},
		},
	}
}

func sacTestContractCodeLedgerKeys(contractID xdr.Hash, contractHash xdr.Hash) []xdr.LedgerKey {
	return []xdr.LedgerKey{
		{
			Type: xdr.LedgerEntryTypeContractData,
			ContractData: &xdr.LedgerKeyContractData{
				ContractId: contractID,
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvLedgerKeyContractExecutable,
				},
			},
		},
		{
			Type: xdr.LedgerEntryTypeContractCode,
			ContractCode: &xdr.LedgerKeyContractCode{
				Hash: contractHash,
			},
		},
	}
}

func addressLedgerKeys(itest *integration.Test, contractAsset xdr.Asset, address xdr.ScAddress) []xdr.LedgerKey {
	contractID := stellarAssetContractID(itest, contractAsset)
	switch address.Type {
	case xdr.ScAddressTypeScAddressTypeAccount:
		// FIXME: if I comment out this "if" , tx_internal_error is returned due to a Core bug
		//        we should be able to uncomment it in the future
		if address.AccountId.Address() == contractAsset.GetIssuer() {
			return nil
		}
		return []xdr.LedgerKey{
			{
				Type: xdr.LedgerEntryTypeTrustline,
				TrustLine: &xdr.LedgerKeyTrustLine{
					AccountId: *address.AccountId,
					Asset:     contractAsset.ToTrustLineAsset(),
				},
			},
		}
	case xdr.ScAddressTypeScAddressTypeContract:
		balance := xdr.ScSymbol("Balance")
		vec := &xdr.ScVec{
			{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &balance,
			},
			{
				Type: xdr.ScValTypeScvAddress,
				Address: &xdr.ScAddress{
					Type:       xdr.ScAddressTypeScAddressTypeContract,
					ContractId: address.ContractId,
				},
			},
		}
		return []xdr.LedgerKey{
			{
				Type: xdr.LedgerEntryTypeContractData,
				ContractData: &xdr.LedgerKeyContractData{
					ContractId: contractID,
					Key: xdr.ScVal{
						Type: xdr.ScValTypeScvVec,
						Vec:  &vec,
					},
				},
			},
		}
	}

	return nil
}

func footprintForRecipientAddress(itest *integration.Test, contractAsset xdr.Asset, address xdr.ScAddress) xdr.LedgerFootprint {
	footPrint := xdr.LedgerFootprint{
		ReadOnly: baseSACFootPrint(itest, contractAsset),
	}
	footPrint.ReadWrite = addressLedgerKeys(itest, contractAsset, address)
	return footPrint
}

func createSAC(itest *integration.Test, sourceAccount string, asset xdr.Asset) *txnbuild.InvokeHostFunction {
	footPrint := xdr.LedgerFootprint{
		ReadWrite: baseSACFootPrint(itest, asset),
	}

	invokeHostFunction := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
			CreateContract: &xdr.CreateContractArgs{
				ContractIdPreimage: xdr.ContractIdPreimage{
					Type:      xdr.ContractIdPreimageTypeContractIdPreimageFromAsset,
					FromAsset: &asset,
				},
				Executable: xdr.ScContractExecutable{
					Type: xdr.ScContractExecutableTypeSccontractExecutableToken,
				},
			},
		},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	return invokeHostFunction
}

func mint(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	return mintWithAmt(itest, sourceAccount, asset, i128Param(0, uint64(amount.MustParse(assetAmount))), recipient)
}

func mintWithAmt(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount xdr.ScVal, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	footPrint := footprintForRecipientAddress(itest, asset, *recipient.Address)
	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("mint"),
				accountAddressParam(sourceAccount),
				recipient,
				assetAmount,
			},
		},
		Auth:          []xdr.SorobanAuthorizationEntry{},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"mint",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			recipient,
			assetAmount,
		})

	return invokeHostFn
}

func initAssetContract(itest *integration.Test, sourceAccount string, asset xdr.Asset, sacTestcontractID, sacTestcontractHash xdr.Hash) *txnbuild.InvokeHostFunction {
	footPrint := xdr.LedgerFootprint{
		ReadOnly:  sacTestContractCodeLedgerKeys(sacTestcontractID, sacTestcontractHash),
		ReadWrite: []xdr.LedgerKey{tokenLedgerKey(sacTestcontractID)},
	}
	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(sacTestcontractID),
				functionNameParam("init"),
				contractIDParam(stellarAssetContractID(itest, asset)),
			},
		},
		Auth:          []xdr.SorobanAuthorizationEntry{},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"init",
		sacTestcontractID,
		xdr.ScVec{
			contractIDParam(stellarAssetContractID(itest, asset)),
		})

	return invokeHostFn
}

func clawback(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	footPrint := footprintForRecipientAddress(itest, asset, *recipient.Address)
	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("clawback"),
				accountAddressParam(sourceAccount),
				recipient,
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		Auth:          []xdr.SorobanAuthorizationEntry{},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"clawback",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			recipient,
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func contractBalance(itest *integration.Test, sourceAccount string, asset xdr.Asset, sacTestcontractID xdr.Hash) *txnbuild.InvokeHostFunction {
	contractID := stellarAssetContractID(itest, asset)
	footPrint := xdr.LedgerFootprint{
		ReadOnly: append(
			addressLedgerKeys(itest, asset, *contractAddressParam(sacTestcontractID).Address),
			xdr.LedgerKey{
				Type: xdr.LedgerEntryTypeContractData,
				ContractData: &xdr.LedgerKeyContractData{
					ContractId: contractID,
					Key: xdr.ScVal{
						Type: xdr.ScValTypeScvLedgerKeyContractExecutable,
					},
				},
			},
		),
	}
	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("balance"),
				contractAddressParam(sacTestcontractID),
			},
		},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	return invokeHostFn
}

func transfer(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	return transferWithAmount(itest, sourceAccount, asset, i128Param(0, uint64(amount.MustParse(assetAmount))), recipient)
}

func transferWithAmount(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount xdr.ScVal, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	footPrint := footprintForRecipientAddress(itest, asset, *recipient.Address)
	footPrint.ReadWrite = append(footPrint.ReadWrite, addressLedgerKeys(itest, asset, *accountAddressParam(sourceAccount).Address)...)
	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("transfer"),
				accountAddressParam(sourceAccount),
				recipient,
				assetAmount,
			},
		},
		Auth:          []xdr.SorobanAuthorizationEntry{},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"transfer",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			accountAddressParam(sourceAccount),
			recipient,
			assetAmount,
		})

	return invokeHostFn
}

func transferFromContract(itest *integration.Test, sourceAccount string, asset xdr.Asset, sacTestcontractID xdr.Hash, sacTestContractHash xdr.Hash, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	footPrint := footprintForRecipientAddress(itest, asset, *recipient.Address)
	footPrint.ReadOnly = append(footPrint.ReadOnly, tokenLedgerKey(sacTestcontractID))
	footPrint.ReadOnly = append(footPrint.ReadOnly, sacTestContractCodeLedgerKeys(sacTestcontractID, sacTestContractHash)...)
	footPrint.ReadWrite = append(footPrint.ReadWrite, addressLedgerKeys(itest, asset, *contractAddressParam(sacTestcontractID).Address)...)

	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(sacTestcontractID),
				functionNameParam("transfer"),
				recipient,
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	return invokeHostFn
}

// Invokes burn_self from the sac_test contract (which just burns assets from itself)
func burnSelf(itest *integration.Test, sourceAccount string, asset xdr.Asset, sacTestcontractID xdr.Hash, sacTestContractHash xdr.Hash, assetAmount string) *txnbuild.InvokeHostFunction {
	footPrint := footprintForRecipientAddress(itest, asset, *contractAddressParam(sacTestcontractID).Address)
	footPrint.ReadOnly = append(footPrint.ReadOnly, tokenLedgerKey(sacTestcontractID))
	footPrint.ReadOnly = append(footPrint.ReadOnly, sacTestContractCodeLedgerKeys(sacTestcontractID, sacTestContractHash)...)
	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(sacTestcontractID),
				functionNameParam("burn_self"),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		Auth:          []xdr.SorobanAuthorizationEntry{},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"burn_self",
		sacTestcontractID,
		xdr.ScVec{
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func burn(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string) *txnbuild.InvokeHostFunction {
	footPrint := footprintForRecipientAddress(itest, asset, *accountAddressParam(sourceAccount).Address)
	invokeHostFn := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("burn"),
				accountAddressParam(sourceAccount),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		Auth:          []xdr.SorobanAuthorizationEntry{},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: getMaxSorobanTransactionData(footPrint),
		},
	}

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"burn",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			accountAddressParam(sourceAccount),
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func assertInvokeHostFnSucceeds(itest *integration.Test, signer *keypair.Full, op *txnbuild.InvokeHostFunction) (*xdr.ScVal, string) {
	acc := itest.MustGetAccount(signer)
	tx, err := itest.SubmitOperationsWithFee(&acc, signer, 10*stroopsIn1XLM, op)
	require.NoError(itest.CurrentTest(), err)

	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	require.NoError(itest.CurrentTest(), err)

	assert.Equal(itest.CurrentTest(), tx.Hash, clientTx.Hash)
	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	require.NoError(itest.CurrentTest(), err)

	var txMetaResult xdr.TransactionMeta
	err = xdr.SafeUnmarshalBase64(clientTx.ResultMetaXdr, &txMetaResult)
	require.NoError(itest.CurrentTest(), err)

	opResults, ok := txResult.OperationResults()
	assert.True(itest.CurrentTest(), ok)
	assert.Equal(itest.CurrentTest(), len(opResults), 1)
	invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	assert.True(itest.CurrentTest(), ok)
	assert.Equal(itest.CurrentTest(), invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)

	returnValue := txMetaResult.MustV3().ReturnValue

	return &returnValue, tx.Hash
}

func stellarAssetContractID(itest *integration.Test, asset xdr.Asset) xdr.Hash {
	contractID, err := asset.ContractID(itest.GetPassPhrase())
	require.NoError(itest.CurrentTest(), err)
	return contractID
}

func addAuthNextInvokerFlow(fnName string, contractId xdr.Hash, args xdr.ScVec) []xdr.SorobanAuthorizationEntry {
	return []xdr.SorobanAuthorizationEntry{
		{
			Credentials: xdr.SorobanCredentials{
				Type:    xdr.SorobanCredentialsTypeSorobanCredentialsAddress,
				Address: &xdr.SorobanAddressCredentials{},
			},
			RootInvocation: xdr.SorobanAuthorizedInvocation{
				Function: xdr.SorobanAuthorizedFunction{
					Type: xdr.SorobanAuthorizedFunctionTypeSorobanAuthorizedFunctionTypeContractFn,
					ContractFn: &xdr.SorobanAuthorizedContractFunction{
						ContractAddress: xdr.ScAddress{
							Type:       xdr.ScAddressTypeScAddressTypeContract,
							ContractId: &contractId,
						},
						FunctionName: xdr.ScSymbol(fnName),
						Args:         args,
					},
				},
				SubInvocations: nil,
			},
		},
	}
}

func mustCreateAndInstallContract(itest *integration.Test, signer *keypair.Full, contractSalt string, wasmFileName string) (xdr.Hash, xdr.Hash) {
	installContractOp := assembleInstallContractCodeOp(itest.CurrentTest(), itest.Master().Address(), wasmFileName)
	assertInvokeHostFnSucceeds(itest, signer, installContractOp)
	createContractOp := assembleCreateContractOp(itest.CurrentTest(), itest.Master().Address(), wasmFileName, contractSalt, itest.GetPassPhrase())
	assertInvokeHostFnSucceeds(itest, signer, createContractOp)
	contractHash := createContractOp.Ext.SorobanData.Resources.Footprint.ReadOnly[0].MustContractCode().Hash
	contractID := createContractOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().ContractId
	return contractID, contractHash
}
