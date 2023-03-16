package integration

import (
	"context"
	"crypto/sha256"

	"testing"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sac_contract = "soroban_sac_test.wasm"

// Tests use precompiled wasm bin files that are added to the testdata directory.
// Refer to ./services/horizon/internal/integration/contracts/README.md on how to recompile
// contract code if needed to new wasm.
//
// `test_add_u64.wasm` is compiled from ./serivces/horizon/internal/integration/contracts/sac_test
//

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
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("20"), stellarAssetContractID(itest, asset))

	fx := getTxEffects(itest, mintTx, asset)
	assert.Len(t, fx, 1)
	debitEffect := assertContainsEffect(t, fx, effects.EffectAccountCredited)[0].(effects.AccountCredited)
	assert.Equal(t, recipientKp.Address(), debitEffect.Account)
	assert.Equal(t, issuer, debitEffect.Asset.Issuer)
	assert.Equal(t, code, debitEffect.Asset.Code)
	assert.Equal(t, "20.0000000", debitEffect.Amount)

	otherRecipientKp, otherRecipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(otherRecipientKp, otherRecipient, txnbuild.MustAssetFromXDR(asset))

	// calling xfer from the issuer account will also mint the asset
	_, xferTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		xfer(itest, issuer, asset, "30", accountAddressParam(otherRecipient.GetAccountID())),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("20"))
	assertContainsBalance(itest, otherRecipientKp, issuer, code, amount.MustParse("30"))
	assertAssetStats(itest, issuer, code, 2, amount.MustParse("50"), stellarAssetContractID(itest, asset))

	fx = getTxEffects(itest, xferTx, asset)
	assert.Len(t, fx, 2)
	assertContainsEffect(t, fx,
		effects.EffectAccountCredited,
		effects.EffectAccountDebited)
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
	recipientContractID := mustCreateAndInstallContract(itest, itest.Master(), "a1", add_u64_contract)

	_, mintTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "20", contractAddressParam(recipientContractID)),
	)
	assert.Empty(t, getTxEffects(itest, mintTx, asset))

	balanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		balance(itest, issuer, asset, contractAddressParam(recipientContractID)),
	)
	assert.Equal(itest.CurrentTest(), xdr.ScValTypeScvObject, balanceAmount.Type)
	assert.Equal(itest.CurrentTest(), xdr.ScObjectTypeScoI128, (*balanceAmount.Obj).Type)

	// The quantities are correct, (they are multiplied by 10^7 because we converted the amounts to stroops)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(200000000), (*balanceAmount.Obj).I128.Lo)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(0), (*balanceAmount.Obj).I128.Hi)

	// calling xfer from the issuer account will also mint the asset
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		xfer(itest, issuer, asset, "30", contractAddressParam(recipientContractID)),
	)

	balanceAmount, _ = assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		balance(itest, issuer, asset, contractAddressParam(recipientContractID)),
	)

	assert.Equal(itest.CurrentTest(), xdr.Uint64(500000000), (*balanceAmount.Obj).I128.Lo)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(0), (*balanceAmount.Obj).I128.Hi)
	assertAssetStats(itest, issuer, code, 0, amount.MustParse("0"), stellarAssetContractID(itest, asset))
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
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1000"), stellarAssetContractID(itest, asset))

	otherRecipientKp, otherRecipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(otherRecipientKp, otherRecipient, txnbuild.MustAssetFromXDR(asset))

	_, xferTx := assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		xfer(itest, recipientKp.Address(), asset, "30", accountAddressParam(otherRecipient.GetAccountID())),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("970"))
	assertContainsBalance(itest, otherRecipientKp, issuer, code, amount.MustParse("30"))
	assertAssetStats(itest, issuer, code, 2, amount.MustParse("1000"), stellarAssetContractID(itest, asset))

	fx := getTxEffects(itest, xferTx, asset)
	assert.NotEmpty(t, fx)
	assertContainsEffect(t, fx, effects.EffectAccountCredited, effects.EffectAccountDebited)
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
	recipientContractID := mustCreateAndInstallContract(itest, itest.Master(), "a1", sac_contract)

	// init recipient contract with the asset contract id
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		initAssetContract(itest, issuer, asset, recipientContractID),
	)

	// Add funds to recipient contract
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "1000", contractAddressParam(recipientContractID)),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("1000"))
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1000"), stellarAssetContractID(itest, asset))

	// transfer from account to contract
	assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		xfer(itest, recipientKp.Address(), asset, "30", contractAddressParam(recipientContractID)),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("970"))
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("970"), stellarAssetContractID(itest, asset))

	// transfer from contract to account
	_, xferTx := assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		xferFromContract(itest,
			recipientKp.Address(), recipientContractID,
			asset, "500",
			accountAddressParam(recipient.GetAccountID())),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("1470"))
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1470"), stellarAssetContractID(itest, asset))

	balanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		balance(itest, issuer, asset, contractAddressParam(recipientContractID)),
	)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(5300000000), (*balanceAmount.Obj).I128.Lo)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(0), (*balanceAmount.Obj).I128.Hi)

	assert.Empty(t, getTxEffects(itest, xferTx, asset))
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
	recipientContractID := mustCreateAndInstallContract(itest, itest.Master(), "a1", sac_contract)

	// Create emitter contract
	emitterContractID := mustCreateAndInstallContract(itest, itest.Master(), "a2", sac_contract)

	// init emitter contract with the asset contract id
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		initAssetContract(itest, issuer, asset, emitterContractID),
	)

	// Add funds to emitter contract
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "1000", contractAddressParam(emitterContractID)),
	)

	// Transfer funds from emitter to recipient
	_, xferTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		xferFromContract(itest, issuer, emitterContractID, asset, "10", contractAddressParam(recipientContractID)),
	)
	assert.Empty(t, getTxEffects(itest, xferTx, asset))

	// Check balances of emitter and recipient
	emitterBalanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		balance(itest, issuer, asset, contractAddressParam(emitterContractID)),
	)

	assert.Equal(itest.CurrentTest(), xdr.Uint64(9900000000), (*emitterBalanceAmount.Obj).I128.Lo)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(0), (*emitterBalanceAmount.Obj).I128.Hi)

	recipientBalanceAmount, _ := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		balance(itest, issuer, asset, contractAddressParam(recipientContractID)),
	)

	assert.Equal(itest.CurrentTest(), xdr.Uint64(100000000), (*recipientBalanceAmount.Obj).I128.Lo)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(0), (*recipientBalanceAmount.Obj).I128.Hi)

	assertAssetStats(itest, issuer, code, 0, amount.MustParse("0"), stellarAssetContractID(itest, asset))
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
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1000"), stellarAssetContractID(itest, asset))

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
	assert.Equal(t, "20.0000000", burnEffect.Amount)
	assert.Equal(t, recipientKp.Address(), burnEffect.Account)
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
	recipientContractID := mustCreateAndInstallContract(itest, itest.Master(), "a1", sac_contract)

	// init contract with asset contract id
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		initAssetContract(itest, issuer, asset, recipientContractID),
	)

	// Add funds to recipient contract
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "1000", contractAddressParam(recipientContractID)),
	)

	// Burn funds
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		burnSelf(itest, issuer, recipientContractID, "10"),
	)

	balanceAmount, burnTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		balance(itest, issuer, asset, contractAddressParam(recipientContractID)),
	)

	assert.Equal(itest.CurrentTest(), xdr.Uint64(9900000000), (*balanceAmount.Obj).I128.Lo)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(0), (*balanceAmount.Obj).I128.Hi)
	assertAssetStats(itest, issuer, code, 0, amount.MustParse("0"), stellarAssetContractID(itest, asset))

	// Burn transactions across contracts generate burn events, but these
	// shouldn't be included as account-related effects.
	assert.Empty(t, getTxEffects(itest, burnTx, asset))
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
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1000"), stellarAssetContractID(itest, asset))

	_, clawTx := assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		clawback(itest, issuer, asset, "1000", accountAddressParam(recipientKp.Address())),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("0"))
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("0"), stellarAssetContractID(itest, asset))

	assertContainsEffect(t, getTxEffects(itest, clawTx, asset), effects.EffectAccountDebited)
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
	recipientContractID := mustCreateAndInstallContract(itest, itest.Master(), "a2", sac_contract)

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
		balance(itest, issuer, asset, contractAddressParam(recipientContractID)),
	)

	assert.Equal(itest.CurrentTest(), xdr.Uint64(9900000000), (*balanceAmount.Obj).I128.Lo)
	assert.Equal(itest.CurrentTest(), xdr.Uint64(0), (*balanceAmount.Obj).I128.Hi)
	assertAssetStats(itest, issuer, code, 0, amount.MustParse("0"), stellarAssetContractID(itest, asset))

	// clawbacks between contracts generate events but not effects
	assert.Empty(t, getTxEffects(itest, clawTx, asset))
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

func assertAssetStats(itest *integration.Test, issuer, code string, numAccounts int32, amt xdr.Int64, contractID [32]byte) {
	assets, err := itest.Client().Assets(horizonclient.AssetRequest{
		ForAssetCode:   code,
		ForAssetIssuer: issuer,
		Limit:          1,
	})
	assert.NoError(itest.CurrentTest(), err)
	for _, asset := range assets.Embedded.Records {
		if asset.Issuer != issuer || asset.Code != code {
			continue
		}
		assert.Equal(itest.CurrentTest(), numAccounts, asset.NumAccounts)
		assert.Equal(itest.CurrentTest(), numAccounts, asset.Accounts.Authorized)
		assert.Equal(itest.CurrentTest(), amt, amount.MustParse(asset.Amount))
		assert.Equal(itest.CurrentTest(), strkey.MustEncode(strkey.VersionByteContract, contractID[:]), asset.ContractID)
		return
	}
	if numAccounts != 0 || amt != 0 {
		itest.CurrentTest().Fatalf("could not find balance for aset %s:%s", code, issuer)
	}
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
		Limit:          2, // no tx should have more than 2 effects
	})
	assert.NoError(t, err)

	return effects.Embedded.Records
}

func masterAccountIDEnumParam(itest *integration.Test) xdr.ScVal {
	root := keypair.Root(itest.GetPassPhrase())
	return accountAddressParam(root.Address())
}

func functionNameParam(name string) xdr.ScVal {
	contractFnParameterSym := xdr.ScSymbol(name)
	return xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &contractFnParameterSym,
	}
}

func contractIDParam(contractID xdr.Hash) xdr.ScVal {
	contractIdBytes := contractID[:]
	contractIdParameterObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &contractIdBytes,
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &contractIdParameterObj,
	}
}

func accountAddressParam(accountID string) xdr.ScVal {
	accountObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoAddress,
		Address: &xdr.ScAddress{
			Type:      xdr.ScAddressTypeScAddressTypeAccount,
			AccountId: xdr.MustAddressPtr(accountID),
		},
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &accountObj,
	}
}

func contractAddressParam(contractID xdr.Hash) xdr.ScVal {
	contractObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoAddress,
		Address: &xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: &contractID,
		},
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &contractObj,
	}
}

func i128Param(hi, lo uint64) xdr.ScVal {
	i128Obj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoI128,
		I128: &xdr.Int128Parts{
			Hi: xdr.Uint64(hi),
			Lo: xdr.Uint64(lo),
		},
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &i128Obj,
	}
}

func createSAC(itest *integration.Test, sourceAccount string, asset xdr.Asset) *txnbuild.InvokeHostFunction {
	return addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
			CreateContractArgs: &xdr.CreateContractArgs{
				ContractId: xdr.ContractId{
					Type:  xdr.ContractIdTypeContractIdFromAsset,
					Asset: &asset,
				},
				Source: xdr.ScContractCode{
					Type: xdr.ScContractCodeTypeSccontractCodeToken,
				},
			},
		},
		SourceAccount: sourceAccount,
	})
}

func mint(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	invokeHostFn := addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("mint"),
				accountAddressParam(sourceAccount),
				recipient,
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"mint",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			accountAddressParam(sourceAccount),
			recipient,
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func initAssetContract(itest *integration.Test, sourceAccount string, asset xdr.Asset, sacTestcontractID xdr.Hash) *txnbuild.InvokeHostFunction {
	invokeHostFn := addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(sacTestcontractID),
				functionNameParam("init"),
				contractIDParam(stellarAssetContractID(itest, asset)),
			},
		},
		SourceAccount: sourceAccount,
	})

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"init",
		sacTestcontractID,
		xdr.ScVec{
			contractIDParam(stellarAssetContractID(itest, asset)),
		})

	return invokeHostFn
}

func clawback(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	invokeHostFn := addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("clawback"),
				accountAddressParam(sourceAccount),
				recipient,
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"clawback",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			accountAddressParam(sourceAccount),
			recipient,
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func balance(itest *integration.Test, sourceAccount string, asset xdr.Asset, holder xdr.ScVal) *txnbuild.InvokeHostFunction {
	return addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("balance"),
				holder,
			},
		},
		SourceAccount: sourceAccount,
	})
}

func xfer(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	invokeHostFn := addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("xfer"),
				accountAddressParam(sourceAccount),
				recipient,
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"xfer",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			accountAddressParam(sourceAccount),
			recipient,
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

// Invokes burn_self from the sac_test contract (which just burns assets from itself)
func burnSelf(itest *integration.Test, sourceAccount string, sacTestcontractID xdr.Hash, assetAmount string) *txnbuild.InvokeHostFunction {
	invokeHostFn := addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(sacTestcontractID),
				functionNameParam("burn_self"),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"burn_self",
		sacTestcontractID,
		xdr.ScVec{
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func xferFromContract(itest *integration.Test, sourceAccount string, sacTestcontractID xdr.Hash, asset xdr.Asset, assetAmount string, recipient xdr.ScVal) *txnbuild.InvokeHostFunction {
	invokeHostFn := addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(sacTestcontractID),
				functionNameParam("xfer"),
				recipient,
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"xfer",
		sacTestcontractID,
		xdr.ScVec{
			recipient,
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func burn(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string) *txnbuild.InvokeHostFunction {
	invokeHostFn := addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest, asset)),
				functionNameParam("burn"),
				accountAddressParam(sourceAccount),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})

	invokeHostFn.Auth = addAuthNextInvokerFlow(
		"burn",
		stellarAssetContractID(itest, asset),
		xdr.ScVec{
			accountAddressParam(sourceAccount),
			i128Param(0, uint64(amount.MustParse(assetAmount))),
		})

	return invokeHostFn
}

func addFootprint(itest *integration.Test, invokeHostFn *txnbuild.InvokeHostFunction) *txnbuild.InvokeHostFunction {
	opXDR, err := invokeHostFn.BuildXDR()
	require.NoError(itest.CurrentTest(), err)

	invokeHostFunctionOp := opXDR.Body.MustInvokeHostFunctionOp()

	// clear footprint so we can verify preflight response
	response, err := itest.CoreClient().Preflight(
		context.Background(),
		invokeHostFn.SourceAccount,
		invokeHostFunctionOp,
	)
	require.NoError(itest.CurrentTest(), err)
	require.Equal(itest.CurrentTest(), stellarcore.PreflightStatusOk, response.Status, response.Detail)
	err = xdr.SafeUnmarshalBase64(response.Footprint, &invokeHostFn.Footprint)
	require.NoError(itest.CurrentTest(), err)
	return invokeHostFn
}

func assertInvokeHostFnSucceeds(itest *integration.Test, signer *keypair.Full, op *txnbuild.InvokeHostFunction) (*xdr.ScVal, string) {
	acc := itest.MustGetAccount(signer)
	tx, err := itest.SubmitOperations(&acc, signer, op)
	require.NoError(itest.CurrentTest(), err)

	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	require.NoError(itest.CurrentTest(), err)

	assert.Equal(itest.CurrentTest(), tx.Hash, clientTx.Hash)
	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	require.NoError(itest.CurrentTest(), err)

	opResults, ok := txResult.OperationResults()
	assert.True(itest.CurrentTest(), ok)
	assert.Equal(itest.CurrentTest(), len(opResults), 1)
	invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	assert.True(itest.CurrentTest(), ok)
	assert.Equal(itest.CurrentTest(), invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)
	return invokeHostFunctionResult.Success, tx.Hash
}

func stellarAssetContractID(itest *integration.Test, asset xdr.Asset) xdr.Hash {
	networkId := xdr.Hash(sha256.Sum256([]byte(itest.GetPassPhrase())))
	preImage := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypeContractIdFromAsset,
		FromAsset: &xdr.HashIdPreimageFromAsset{
			NetworkId: networkId,
			Asset:     asset,
		},
	}
	xdrPreImageBytes, err := preImage.MarshalBinary()
	require.NoError(itest.CurrentTest(), err)
	return sha256.Sum256(xdrPreImageBytes)
}

func addAuthNextInvokerFlow(fnName string, contractId xdr.Hash, args xdr.ScVec) []xdr.ContractAuth {
	return []xdr.ContractAuth{
		{
			RootInvocation: xdr.AuthorizedInvocation{
				ContractId:     contractId,
				FunctionName:   xdr.ScSymbol(fnName),
				Args:           args,
				SubInvocations: nil,
			},
			SignatureArgs: nil,
		},
	}
}

func mustCreateAndInstallContract(itest *integration.Test, signer *keypair.Full, contractSalt string, wasmFileName string) xdr.Hash {
	installContractOp := assembleInstallContractCodeOp(itest.CurrentTest(), itest.Master().Address(), wasmFileName)
	assertInvokeHostFnSucceeds(itest, signer, installContractOp)
	createContractOp := assembleCreateContractOp(itest.CurrentTest(), itest.Master().Address(), wasmFileName, contractSalt, itest.GetPassPhrase())
	assertInvokeHostFnSucceeds(itest, signer, createContractOp)
	return createContractOp.Footprint.ReadWrite[0].MustContractData().ContractId
}
