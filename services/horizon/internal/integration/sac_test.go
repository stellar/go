package integration

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMintToAccount(t *testing.T) {
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

	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		mint(itest, issuer, asset, "20", recipient.GetAccountID()),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("20"))
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("20"))

	otherRecipientKp, otherRecipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(otherRecipientKp, otherRecipient, txnbuild.MustAssetFromXDR(asset))

	// calling xfer from the issuer account will also mint the asset
	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		xfer(itest, issuer, asset, "30", otherRecipient.GetAccountID()),
	)
	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("20"))
	assertContainsBalance(itest, otherRecipientKp, issuer, code, amount.MustParse("30"))
	assertAssetStats(itest, issuer, code, 2, amount.MustParse("50"))
}

func TestTransferBetweenAccounts(t *testing.T) {
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
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1000"))

	otherRecipientKp, otherRecipient := itest.CreateAccount("100")
	itest.MustEstablishTrustline(otherRecipientKp, otherRecipient, txnbuild.MustAssetFromXDR(asset))

	assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		xfer(itest, recipientKp.Address(), asset, "30", otherRecipient.GetAccountID()),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("970"))
	assertContainsBalance(itest, otherRecipientKp, issuer, code, amount.MustParse("30"))
	assertAssetStats(itest, issuer, code, 2, amount.MustParse("1000"))
}

func TestBurnFromAccount(t *testing.T) {
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
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1000"))

	assertInvokeHostFnSucceeds(
		itest,
		recipientKp,
		burn(itest, recipientKp.Address(), asset, "500"),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("500"))
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("500"))
}

func TestClawbackFromAccount(t *testing.T) {
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
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("1000"))

	assertInvokeHostFnSucceeds(
		itest,
		itest.Master(),
		clawback(itest, issuer, asset, "1000", recipientKp.Address()),
	)

	assertContainsBalance(itest, recipientKp, issuer, code, amount.MustParse("0"))
	assertAssetStats(itest, issuer, code, 1, amount.MustParse("0"))
}

func assertContainsBalance(itest *integration.Test, acct *keypair.Full, issuer, code string, amt xdr.Int64) {
	for _, b := range itest.MustGetAccount(acct).Balances {
		if b.Issuer == issuer && b.Code == code && amount.MustParse(b.Balance) == amt {
			return
		}
	}
	itest.CurrentTest().Fatalf("could not find balance for aset %s:%s", code, issuer)
}

func assertAssetStats(itest *integration.Test, issuer, code string, numAccounts int32, amt xdr.Int64) {
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
		return
	}
	itest.CurrentTest().Fatalf("could not find balance for aset %s:%s", code, issuer)
}

func invokerSignatureParam() xdr.ScVal {
	invokerSym := xdr.ScSymbol("Invoker")
	obj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoVec,
		Vec: &xdr.ScVec{
			xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &invokerSym,
			},
		},
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &obj,
	}
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

func accountIDParam(accountID string) xdr.ScVal {
	accountObj := &xdr.ScObject{
		Type:      xdr.ScObjectTypeScoAccountId,
		AccountId: xdr.MustAddressPtr(accountID),
	}
	accountSym := xdr.ScSymbol("Account")
	accountEnum := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoVec,
		Vec: &xdr.ScVec{
			xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &accountSym,
			},
			xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &accountObj,
			},
		},
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &accountEnum,
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

func mint(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient string) *txnbuild.InvokeHostFunction {
	return addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest.CurrentTest(), itest.GetPassPhrase(), asset)),
				functionNameParam("mint"),
				invokerSignatureParam(),
				i128Param(0, 0),
				accountIDParam(recipient),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})
}

func clawback(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient string) *txnbuild.InvokeHostFunction {
	return addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest.CurrentTest(), itest.GetPassPhrase(), asset)),
				functionNameParam("clawback"),
				invokerSignatureParam(),
				i128Param(0, 0),
				accountIDParam(recipient),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})
}

func xfer(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string, recipient string) *txnbuild.InvokeHostFunction {
	return addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest.CurrentTest(), itest.GetPassPhrase(), asset)),
				functionNameParam("xfer"),
				invokerSignatureParam(),
				i128Param(0, 0),
				accountIDParam(recipient),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})
}

func burn(itest *integration.Test, sourceAccount string, asset xdr.Asset, assetAmount string) *txnbuild.InvokeHostFunction {
	return addFootprint(itest, &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeArgs: &xdr.ScVec{
				contractIDParam(stellarAssetContractID(itest.CurrentTest(), itest.GetPassPhrase(), asset)),
				functionNameParam("burn"),
				invokerSignatureParam(),
				i128Param(0, 0),
				i128Param(0, uint64(amount.MustParse(assetAmount))),
			},
		},
		SourceAccount: sourceAccount,
	})
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
	err = xdr.SafeUnmarshalBase64(response.Footprint, &invokeHostFn.Footprint)
	require.NoError(itest.CurrentTest(), err)
	require.Equal(itest.CurrentTest(), stellarcore.PreflightStatusOk, response.Status)
	return invokeHostFn
}

func assertInvokeHostFnSucceeds(itest *integration.Test, signer *keypair.Full, op *txnbuild.InvokeHostFunction) {
	acc := itest.MustGetAccount(signer)
	tx, err := itest.SubmitOperations(&acc, signer, op)
	require.NoError(itest.CurrentTest(), err)

	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	require.NoError(itest.CurrentTest(), err)

	effects, err := itest.Client().Effects(horizonclient.EffectRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(itest.CurrentTest(), err)
	// Horizon currently does not support effects for smart contract invocations
	require.Empty(itest.CurrentTest(), effects.Embedded.Records)

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
}

func stellarAssetContractID(t *testing.T, passPhrase string, asset xdr.Asset) xdr.Hash {
	networkId := xdr.Hash(sha256.Sum256([]byte(passPhrase)))
	preImage := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypeContractIdFromAsset,
		FromAsset: &xdr.HashIdPreimageFromAsset{
			NetworkId: networkId,
			Asset:     asset,
		},
	}
	xdrPreImageBytes, err := preImage.MarshalBinary()
	require.NoError(t, err)
	return sha256.Sum256(xdrPreImageBytes)
}
