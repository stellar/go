package ingest

import (
	"crypto/rand"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/require"
)

const passphrase = "passphrase"

func TestSACTransferEvent(t *testing.T) {
	randomIssuer := keypair.MustRandom()
	randomAsset := xdr.MustNewCreditAsset("TESTING", randomIssuer.Address())

	rawContractId, err := randomAsset.ContractID(passphrase)
	contractId := xdr.Hash(rawContractId)
	require.NoError(t, err)

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
		Topics: makeTransferTopic(randomAsset),
		Data:   makeAmount(10000),
	}

	// Ensure the happy path for transfer events works
	sacEvent, err := NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.NotNil(t, sacEvent)
	require.Equal(t, EventTypeTransfer, sacEvent.Type)

	require.NotNil(t, sacEvent.From)
	require.NotNil(t, sacEvent.To)
	require.NotNil(t, sacEvent.Amount)

	require.Equal(t, randomIssuer.Address(), sacEvent.From)
	require.Equal(t, randomIssuer.Address(), sacEvent.To)
	require.EqualValues(t, 10000, sacEvent.Amount.Lo)
	require.EqualValues(t, 0, sacEvent.Amount.Hi)

	// Ensure that changing the passphrase invalidates the event
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, "different")
	require.Error(t, err)

	// Ensure that it works for the native asset
	oldAsset := *(*baseXdrEvent.Body.V0.Topics[3].Obj).Bin
	*(*baseXdrEvent.Body.V0.Topics[3].Obj).Bin = []byte("native")
	sacEvent, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.Equal(t, xdr.AssetTypeAssetTypeNative, sacEvent.Asset.Type)

	// Ensure that invalid asset binaries are rejected
	bsAsset := make([]byte, len(oldAsset))
	rand.Read(bsAsset)
	(*baseXdrEvent.Body.V0.Topics[3].Obj).Bin = &bsAsset
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.Error(t, err)
	(*baseXdrEvent.Body.V0.Topics[3].Obj).Bin = &oldAsset

	// Ensure that system events are invalid
	baseXdrEvent.Type = xdr.ContractEventTypeSystem
	_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.Error(t, err)
	baseXdrEvent.Type = xdr.ContractEventTypeContract
}

func makeTransferTopic(asset xdr.Asset) xdr.ScVec {
	accountId, _ := xdr.AddressToAccountId(asset.GetIssuer())

	fnName := xdr.ScSymbol("transfer")
	account := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoAddress,
		Address: &xdr.ScAddress{
			Type:      xdr.ScAddressTypeScAddressTypeAccount,
			AccountId: &accountId,
		},
	}

	slice := []byte(asset.StringCanonical())
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
