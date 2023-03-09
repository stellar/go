package ingest

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/require"
)

const passphrase = "passphrase"

func TestSACTransferEvent(t *testing.T) {
	randomIssuer := keypair.MustRandom()
	randomAsset := xdr.MustNewCreditAsset("TESTING", randomIssuer.Address())
	randomAccount := keypair.MustRandom().Address()

	rawNativeContractId, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	require.NoError(t, err)
	rawContractId, err := randomAsset.ContractID(passphrase)
	require.NoError(t, err)

	nativeContractId := xdr.Hash(rawNativeContractId)
	contractId := xdr.Hash(rawContractId)

	baseXdrEvent := xdr.ContractEvent{
		Ext:        xdr.ExtensionPoint{V: 0},
		ContractId: &contractId,
		Type:       xdr.ContractEventTypeContract,
		Body: xdr.ContractEventBody{
			V:  0,
			V0: nil,
		},
	}

	baseXdrEvent.Body.V0 = &xdr.ContractEventV0{
		Topics: makeTransferTopic(randomAsset, randomAccount),
		Data:   makeAmount(10000),
	}

	// Ensure the happy path for transfer events works
	sacEvent, err := NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.NotNil(t, sacEvent)
	require.Equal(t, EventTypeTransfer, sacEvent.GetType())

	xferEvent := sacEvent.(*TransferEvent)
	require.Equal(t, randomAccount, xferEvent.From)
	require.Equal(t, randomAccount, xferEvent.To)
	require.EqualValues(t, 10000, xferEvent.Amount.Lo)
	require.EqualValues(t, 0, xferEvent.Amount.Hi)

	// Ensure that changing the passphrase invalidates the event
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, "different")
	require.Error(t, err)

	// Ensure that it works for the native asset
	baseXdrEvent.ContractId = &nativeContractId
	baseXdrEvent.Body.V0.Topics = makeTransferTopic(xdr.MustNewNativeAsset(), randomAccount)
	sacEvent, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.Equal(t, xdr.AssetTypeAssetTypeNative, sacEvent.GetAsset().Type)

	// Ensure that invalid asset binaries are rejected
	bsAsset := make([]byte, 42)
	rand.Read(bsAsset)
	(*baseXdrEvent.Body.V0.Topics[3].Obj).Bin = &bsAsset
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.Error(t, err)

	// Ensure that valid asset binaries that mismatch the contract are rejected
	baseXdrEvent.ContractId = &nativeContractId
	baseXdrEvent.Body.V0.Topics = makeTransferTopic(randomAsset, randomAccount)
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.Error(t, err)
	baseXdrEvent.ContractId = &contractId
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)

	// Ensure that system events are invalid
	baseXdrEvent.Type = xdr.ContractEventTypeSystem
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.Error(t, err)
	baseXdrEvent.Type = xdr.ContractEventTypeContract
}

func makeTransferTopic(asset xdr.Asset, participant string) xdr.ScVec {
	accountId, err := xdr.AddressToAccountId(participant)
	if err != nil {
		panic(fmt.Errorf("participant (%s) isn't an account ID: %v",
			participant, err))
	}

	fnName := xdr.ScSymbol("transfer")
	account := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoAddress,
		Address: &xdr.ScAddress{
			Type:      xdr.ScAddressTypeScAddressTypeAccount,
			AccountId: &accountId,
		},
	}

	slice := []byte("native")
	if asset.Type != xdr.AssetTypeAssetTypeNative {
		slice = []byte(asset.StringCanonical())
	}
	assetDetails := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &slice,
	}

	return xdr.ScVec([]xdr.ScVal{
		// event name
		{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &fnName,
		},
		// from
		{
			Type: xdr.ScValTypeScvObject,
			Obj:  &account,
		},
		// to
		{
			Type: xdr.ScValTypeScvObject,
			Obj:  &account,
		},
		// asset details
		{
			Type: xdr.ScValTypeScvObject,
			Obj:  &assetDetails,
		},
	})
}

func makeAmount(amount int) xdr.ScVal {
	amountObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoI128,
		I128: &xdr.Int128Parts{
			Lo: xdr.Uint64(amount),
			Hi: 0,
		},
	}

	return xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &amountObj,
	}
}
