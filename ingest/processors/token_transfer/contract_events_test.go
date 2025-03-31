package token_transfer

import (
	"encoding/hex"
	"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	randomAccount     = keypair.MustRandom().Address()
	someContractHash1 = xdr.Hash([32]byte{1, 2, 3, 4})
	someContract1     = strkey.MustEncode(strkey.VersionByteContract, someContractHash1[:])
	someContractHash2 = xdr.Hash([32]byte{4, 3, 2, 1})
	someContract2     = strkey.MustEncode(strkey.VersionByteContract, someContractHash2[:])

	processor = &EventsProcessor{
		networkPassphrase: someNetworkPassphrase,
	}
	contractIdFromAsset = func(asset xdr.Asset, passphrase string) *xdr.Hash {
		contractId, _ := asset.ContractID(passphrase)
		hash := xdr.Hash(contractId)
		return &hash
	}
)

// Helper function to create a ScVal symbol
func createSymbol(sym string) xdr.ScVal {
	symStr := xdr.ScSymbol(sym)
	return xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &symStr,
	}
}

func contractIdToHash(contractId string) *xdr.Hash {
	idBytes := [32]byte{}
	rawBytes, err := hex.DecodeString(contractId)
	if err != nil {
		panic(fmt.Errorf("invalid contract id (%s): %v", contractId, err))
	}
	if copy(idBytes[:], rawBytes[:]) != 32 {
		panic("couldn't copy 32 bytes to contract hash")
	}

	hash := xdr.Hash(idBytes)
	return &hash
}

func createAddress(address string) xdr.ScVal {
	scAddress := xdr.ScAddress{}

	switch address[0] {
	case 'C':
		scAddress.Type = xdr.ScAddressTypeScAddressTypeContract
		contractHash := strkey.MustDecode(strkey.VersionByteContract, address)
		scAddress.ContractId = contractIdToHash(hex.EncodeToString(contractHash))

	case 'G':
		scAddress.Type = xdr.ScAddressTypeScAddressTypeAccount
		scAddress.AccountId = xdr.MustAddressPtr(address)

	default:
		panic(fmt.Errorf("unsupported address: %s", address))
	}

	return xdr.ScVal{
		Type:    xdr.ScValTypeScvAddress,
		Address: &scAddress,
	}
}

// Helper function to create a ScVal string
func createString(str string) xdr.ScVal {
	symStr := xdr.ScString(str)
	return xdr.ScVal{
		Type: xdr.ScValTypeScvString,
		Str:  &symStr,
	}
}

// Helper function to create an Int128 ScVal
func createInt128(val int64) xdr.ScVal {
	parts := xdr.Int128Parts{
		Lo: xdr.Uint64(val),
		Hi: 0,
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvI128,
		I128: &parts,
	}
}

// Helper function to create a mock ContractEvent
func createContractEvent(
	eventType string,
	from, to string,
	amount int64,
	assetStr string,
	contractId *xdr.Hash,
) xdr.ContractEvent {
	topics := []xdr.ScVal{
		createSymbol(eventType),
	}

	if from != "" {
		topics = append(topics, createAddress(from))
	}

	if to != "" {
		topics = append(topics, createAddress(to))
	}

	if assetStr != "" {
		topics = append(topics, createString(assetStr))
	}

	return xdr.ContractEvent{
		Type:       xdr.ContractEventTypeContract,
		ContractId: contractId,
		Body: xdr.ContractEventBody{
			V: 0,
			V0: &xdr.ContractEventV0{
				Topics: topics,
				Data:   createInt128(amount),
			},
		},
	}
}

func TestValidContractEvents(t *testing.T) {
	testCases := []struct {
		name          string
		eventType     string
		addr1         string // meaning depends on event type (from/admin)
		addr2         string // meaning depends on event type (to/from/empty)
		amount        int64
		isSacEvent    bool
		validateEvent func(t *testing.T, event *TokenTransferEvent, addr1, addr2 string, amount string, assetItem interface{})
	}{
		{
			name:       "Transfer SEP-41 Token Event",
			eventType:  TransferEvent,
			addr1:      someContract1, // from
			addr2:      someContract2, // to
			amount:     1000,
			isSacEvent: false,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, from, to string, amount string, _ interface{}) {
				assert.NotNil(t, event.GetTransfer())
				assert.Equal(t, from, event.GetTransfer().From)
				assert.Equal(t, to, event.GetTransfer().To)
				assert.Equal(t, amount, event.GetTransfer().Amount)
				assert.Nil(t, event.GetTransfer().Asset) // asset is nil for non-SAC events
			},
		},
		{
			name:       "Transfer SAC Event",
			eventType:  TransferEvent,
			addr1:      randomAccount, // from
			addr2:      someContract1, // to
			amount:     1000,
			isSacEvent: true,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, from, to string, amount string, assetItem interface{}) {
				assert.NotNil(t, event.GetTransfer())
				assert.Equal(t, from, event.GetTransfer().From)
				assert.Equal(t, to, event.GetTransfer().To)
				assert.Equal(t, amount, event.GetTransfer().Amount)
				assert.NotNil(t, event.GetTransfer().Asset)

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetTransfer().Asset.ToXdrAsset().Equals(asset))
			},
		},
		{
			name:       "Mint SEP-41 Token Event",
			eventType:  MintEvent,
			addr1:      someContract2, // admin
			addr2:      someContract1, // to
			amount:     500,
			isSacEvent: false,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, admin, to string, amount string, _ interface{}) {
				assert.NotNil(t, event.GetMint())
				assert.Equal(t, to, event.GetMint().To)
				assert.Equal(t, amount, event.GetMint().Amount)
				assert.Nil(t, event.GetMint().Asset) // asset is nil for non-SAC events
			},
		},
		{
			name:       "Mint SAC Event",
			eventType:  MintEvent,
			addr1:      randomAccount, // admin
			addr2:      someContract1, // to
			amount:     500,
			isSacEvent: true,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, admin, to string, amount string, assetItem interface{}) {
				assert.NotNil(t, event.GetMint())
				assert.Equal(t, to, event.GetMint().To)
				assert.Equal(t, amount, event.GetMint().Amount)
				assert.NotNil(t, event.GetMint().Asset)

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetMint().Asset.ToXdrAsset().Equals(asset))
			},
		},
		{
			name:       "Burn SEP-41 Token Event",
			eventType:  BurnEvent,
			addr1:      randomAccount, // from
			addr2:      "",            // no second address for burn
			amount:     300,
			isSacEvent: false,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, from, _ string, amount string, _ interface{}) {
				assert.NotNil(t, event.GetBurn())
				assert.Equal(t, from, event.GetBurn().From)
				assert.Equal(t, amount, event.GetBurn().Amount)
				assert.Nil(t, event.GetBurn().Asset) // asset is nil for non-SAC events
			},
		},
		{
			name:       "Burn SAC Event",
			eventType:  BurnEvent,
			addr1:      randomAccount, // from
			addr2:      "",            // no second address for burn
			amount:     300,
			isSacEvent: true,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, from, _ string, amount string, assetItem interface{}) {
				assert.NotNil(t, event.GetBurn())
				assert.Equal(t, from, event.GetBurn().From)
				assert.Equal(t, amount, event.GetBurn().Amount)
				assert.NotNil(t, event.GetBurn().Asset)

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetBurn().Asset.ToXdrAsset().Equals(asset))
			},
		},
		{
			name:       "Clawback SEP-41 Token Event",
			eventType:  ClawbackEvent,
			addr1:      someContract1, // admin
			addr2:      someContract2, // from (user whose tokens are clawed back)
			amount:     200,
			isSacEvent: false,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, admin, from string, amount string, _ interface{}) {
				assert.NotNil(t, event.GetClawback())
				assert.Equal(t, from, event.GetClawback().From)
				assert.Equal(t, amount, event.GetClawback().Amount)
				assert.Nil(t, event.GetClawback().Asset) // asset is nil for non-SAC events
			},
		},
		{
			name:       "Clawback SAC Event",
			eventType:  ClawbackEvent,
			addr1:      someContract1, // admin
			addr2:      randomAccount, // from (user whose tokens are clawed back)
			amount:     200,
			isSacEvent: true,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, admin, from string, amount string, assetItem interface{}) {
				assert.NotNil(t, event.GetClawback())
				assert.Equal(t, from, event.GetClawback().From)
				assert.Equal(t, amount, event.GetClawback().Amount)
				assert.NotNil(t, event.GetClawback().Asset)

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetClawback().Asset.ToXdrAsset().Equals(asset))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with multiple assets
			var assets []interface{}
			var contractId *xdr.Hash

			if tc.isSacEvent {
				// Test with SAC assets
				for _, asset := range []xdr.Asset{xlmAsset, usdcAsset, ethAsset} {
					assets = append(assets, asset)
				}
			} else {
				// Test with non-SAC assets
				for _, asset := range []string{"someNonSep11AssetString", xlmAsset.StringCanonical(), usdcAsset.StringCanonical()} {
					assets = append(assets, asset)
				}
				contractId = &someContractHash1
			}

			for _, assetItem := range assets {
				var assetStr string

				if tc.isSacEvent {
					asset := assetItem.(xdr.Asset)
					assetStr = asset.StringCanonical()
					contractId = contractIdFromAsset(asset, processor.networkPassphrase)
				} else {
					assetStr = assetItem.(string)
				}

				// Create the contract event based on event type
				var contractEvent xdr.ContractEvent

				switch tc.eventType {
				case TransferEvent:
					// from, to
					contractEvent = createContractEvent(
						tc.eventType,
						tc.addr1, // from
						tc.addr2, // to
						tc.amount,
						assetStr,
						contractId,
					)
				case MintEvent:
					// admin, to
					contractEvent = createContractEvent(
						tc.eventType,
						tc.addr1, // admin
						tc.addr2, // to
						tc.amount,
						assetStr,
						contractId,
					)
				case BurnEvent:
					// from, ""
					contractEvent = createContractEvent(
						tc.eventType,
						tc.addr1, // from
						"",       // toAddress is empty
						tc.amount,
						assetStr,
						contractId,
					)
				case ClawbackEvent:
					// admin, from
					contractEvent = createContractEvent(
						tc.eventType,
						tc.addr1, // admin
						tc.addr2, // address from which asset needs to be clawed back
						tc.amount,
						assetStr,
						contractId,
					)
				}

				event, err := processor.ParseEvent(someTx, &someOperationIndex, contractEvent)

				require.NoError(t, err)
				require.NotNil(t, event)

				amountStr := fmt.Sprintf("%d", tc.amount)
				tc.validateEvent(t, event, tc.addr1, tc.addr2, amountStr, assetItem)
			}
		})
	}
}
