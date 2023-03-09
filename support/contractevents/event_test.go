package contractevents

import (
	"crypto/rand"
	"testing"

	"github.com/stellar/go/gxdr"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/randxdr"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const passphrase = "passphrase"

var (
	randomIssuer     = keypair.MustRandom()
	randomAsset      = xdr.MustNewCreditAsset("TESTING", randomIssuer.Address())
	randomAccount    = keypair.MustRandom().Address()
	zeroContractHash = xdr.Hash([32]byte{})
	zeroContract     = strkey.MustEncode(strkey.VersionByteContract, zeroContractHash[:])
)

func TestSACTransferEvent(t *testing.T) {
	rawNativeContractId, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	require.NoError(t, err)
	nativeContractId := xdr.Hash(rawNativeContractId)

	var baseXdrEvent xdr.ContractEvent
	resetEvent := func() {
		baseXdrEvent = makeEvent()
		baseXdrEvent.Body.V0 = &xdr.ContractEventV0{
			Topics: makeTransferTopic(randomAsset),
			Data:   makeAmount(10000),
		}

	}
	resetEvent()

	// Ensure the happy path for transfer events works
	sacEvent, err := NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.NotNil(t, sacEvent)
	require.Equal(t, EventTypeTransfer, sacEvent.GetType())

	xferEvent := sacEvent.(*TransferEvent)
	require.Equal(t, randomAccount, xferEvent.From)
	require.Equal(t, zeroContract, xferEvent.To)
	require.EqualValues(t, 10000, xferEvent.Amount.Lo)
	require.EqualValues(t, 0, xferEvent.Amount.Hi)

	// Ensure that changing the passphrase invalidates the event
	t.Run("wrong passphrase", func(t *testing.T) {
		_, err = NewStellarAssetContractEvent(&baseXdrEvent, "different")
		require.Error(t, err)
	})

	// Ensure that the native asset still works
	t.Run("native transfer", func(t *testing.T) {
		resetEvent()
		baseXdrEvent.ContractId = &nativeContractId
		baseXdrEvent.Body.V0.Topics = makeTransferTopic(xdr.MustNewNativeAsset())
		sacEvent, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
		require.NoError(t, err)
		require.Equal(t, xdr.AssetTypeAssetTypeNative, sacEvent.GetAsset().Type)
	})

	// Ensure that invalid asset binaries are rejected
	t.Run("bad asset binary", func(t *testing.T) {
		resetEvent()
		bsAsset := make([]byte, 42)
		rand.Read(bsAsset)
		(*baseXdrEvent.Body.V0.Topics[3].Obj).Bin = &bsAsset
		_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
		require.Error(t, err)
	})

	// Ensure that valid asset binaries that mismatch the contract are rejected
	t.Run("mismatching ID", func(t *testing.T) {
		resetEvent()
		// change the ID but keep the asset
		baseXdrEvent.ContractId = &nativeContractId
		_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
		require.Error(t, err)

		// now change the asset but keep the ID
		resetEvent()
		diffRandomAsset := xdr.MustNewCreditAsset("TESTING", keypair.MustRandom().Address())
		baseXdrEvent.Body.V0.Topics = makeTransferTopic(diffRandomAsset)
		_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
		require.Error(t, err)
	})

	// Ensure that system events are rejected
	t.Run("system events", func(t *testing.T) {
		resetEvent()
		baseXdrEvent.Type = xdr.ContractEventTypeSystem
		_, err = NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
		require.Error(t, err)
		baseXdrEvent.Type = xdr.ContractEventTypeContract
	})
}

func TestSACMintEvent(t *testing.T) {
	baseXdrEvent := makeEvent()
	baseXdrEvent.Body.V0 = &xdr.ContractEventV0{
		Topics: makeMintTopic(randomAsset),
		Data:   makeAmount(10000),
	}

	// Ensure the happy path for mint events works
	sacEvent, err := NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.NotNil(t, sacEvent)
	require.Equal(t, EventTypeMint, sacEvent.GetType())

	mintEvent := sacEvent.(*MintEvent)
	require.Equal(t, randomAccount, mintEvent.Admin)
	require.Equal(t, zeroContract, mintEvent.To)
	require.EqualValues(t, 10000, mintEvent.Amount.Lo)
	require.EqualValues(t, 0, mintEvent.Amount.Hi)
}

func TestSACClawbackEvent(t *testing.T) {
	baseXdrEvent := makeEvent()
	baseXdrEvent.Body.V0 = &xdr.ContractEventV0{
		Topics: makeClawbackTopic(randomAsset),
		Data:   makeAmount(10000),
	}

	// Ensure the happy path for clawback events works
	sacEvent, err := NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.NotNil(t, sacEvent)
	require.Equal(t, EventTypeClawback, sacEvent.GetType())

	clawEvent := sacEvent.(*ClawbackEvent)
	require.Equal(t, randomAccount, clawEvent.Admin)
	require.Equal(t, zeroContract, clawEvent.From)
	require.EqualValues(t, 10000, clawEvent.Amount.Lo)
	require.EqualValues(t, 0, clawEvent.Amount.Hi)
}

func TestSACBurnEvent(t *testing.T) {
	baseXdrEvent := makeEvent()
	baseXdrEvent.Body.V0 = &xdr.ContractEventV0{
		Topics: makeBurnTopic(randomAsset),
		Data:   makeAmount(10000),
	}

	// Ensure the happy path for burn events works
	sacEvent, err := NewStellarAssetContractEvent(&baseXdrEvent, passphrase)
	require.NoError(t, err)
	require.NotNil(t, sacEvent)
	require.Equal(t, EventTypeBurn, sacEvent.GetType())

	burnEvent := sacEvent.(*BurnEvent)
	require.Equal(t, randomAccount, burnEvent.From)
	require.EqualValues(t, 10000, burnEvent.Amount.Lo)
	require.EqualValues(t, 0, burnEvent.Amount.Hi)
}

func TestFuzzingSACEventParser(t *testing.T) {
	gen := randxdr.NewGenerator()
	for i := 0; i < 100_000; i++ {
		event, shape := xdr.ContractEvent{}, &gxdr.ContractEvent{}

		gen.Next(
			shape,
			[]randxdr.Preset{},
		)
		assert.NoError(t, gxdr.Convert(shape, &event))

		// return values are ignored, but this should never panic
		NewStellarAssetContractEvent(&event, "passphrase")
	}
}

//
// Test suite helpers below
//

func makeEvent() xdr.ContractEvent {
	rawContractId, err := randomAsset.ContractID(passphrase)
	if err != nil {
		panic(err)
	}
	contractId := xdr.Hash(rawContractId)

	baseXdrEvent := xdr.ContractEvent{
		Ext:        xdr.ExtensionPoint{V: 0},
		ContractId: &contractId,
		Type:       xdr.ContractEventTypeContract,
		Body: xdr.ContractEventBody{
			V:  0,
			V0: &xdr.ContractEventV0{},
		},
	}

	return baseXdrEvent
}

func makeTransferTopic(asset xdr.Asset) xdr.ScVec {
	accountId := xdr.MustAddress(randomAccount)

	fnName := xdr.ScSymbol("transfer")
	account := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoAddress,
		Address: &xdr.ScAddress{
			Type:      xdr.ScAddressTypeScAddressTypeAccount,
			AccountId: &accountId,
		},
	}
	contract := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoAddress,
		Address: &xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: &zeroContractHash,
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
			Obj:  &contract,
		},
		// asset details
		{
			Type: xdr.ScValTypeScvObject,
			Obj:  &assetDetails,
		},
	})
}

func makeMintTopic(asset xdr.Asset) xdr.ScVec {
	// mint is just transfer but with an admin instead of a from... nice
	fnName := xdr.ScSymbol("mint")
	topics := makeTransferTopic(asset)
	topics[0].Sym = &fnName
	return topics
}

func makeClawbackTopic(asset xdr.Asset) xdr.ScVec {
	// clawback is just mint but with a from instead of a to
	fnName := xdr.ScSymbol("clawback")
	topics := makeTransferTopic(asset)
	topics[0].Sym = &fnName
	return topics
}

func makeBurnTopic(asset xdr.Asset) xdr.ScVec {
	// burn is like clawback but without a "to", so we drop that topic
	fnName := xdr.ScSymbol("burn")
	topics := makeTransferTopic(asset)
	topics[0].Sym = &fnName
	topics = append(topics[:2], topics[3:]...)
	return topics
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
