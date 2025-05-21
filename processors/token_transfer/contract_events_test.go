package token_transfer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

var (
	randomAccount     = keypair.MustRandom().Address()
	someContractHash1 = xdr.ContractId([32]byte{1, 2, 3, 4})
	someContract1     = strkey.MustEncode(strkey.VersionByteContract, someContractHash1[:])
	someContractHash2 = xdr.ContractId([32]byte{4, 3, 2, 1})
	someContract2     = strkey.MustEncode(strkey.VersionByteContract, someContractHash2[:])

	processor = &EventsProcessor{
		networkPassphrase: someNetworkPassphrase,
	}
	contractIdFromAsset = func(asset xdr.Asset) *xdr.ContractId {
		contractId, _ := asset.ContractID(someNetworkPassphrase)
		hash := xdr.ContractId(contractId)
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

func createAddress(address string) xdr.ScVal {
	scAddress := xdr.ScAddress{}

	switch {
	case strkey.IsValidContractAddress(address) == true:
		scAddress.Type = xdr.ScAddressTypeScAddressTypeContract
		contractHash := strkey.MustDecode(strkey.VersionByteContract, address)
		contractId := xdr.ContractId(contractHash)
		scAddress.ContractId = &contractId

	case strkey.IsValidEd25519PublicKey(address) == true:
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
	amount128 *xdr.Int128Parts,
	assetStr string,
	contractId *xdr.ContractId,
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

	var data xdr.ScVal
	if amount128 != nil {
		data = xdr.ScVal{
			Type: xdr.ScValTypeScvI128,
			I128: amount128,
		}

	} else {
		data = createInt128(amount)
	}

	return xdr.ContractEvent{
		Type:       xdr.ContractEventTypeContract,
		ContractId: contractId,
		Body: xdr.ContractEventBody{
			V: 0,
			V0: &xdr.ContractEventV0{
				Topics: topics,
				Data:   data,
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
		amount128     *xdr.Int128Parts
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
				assert.Nil(t, event.GetAsset()) // asset is nil for non-SAC events
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
				assert.NotNil(t, event.GetAsset())

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetAsset().ToXdrAsset().Equals(asset))
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
				assert.Nil(t, event.GetAsset()) // asset is nil for non-SAC events
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
				assert.NotNil(t, event.GetAsset())

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetAsset().ToXdrAsset().Equals(asset))
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
				assert.Nil(t, event.GetAsset()) // asset is nil for non-SAC events
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
				assert.NotNil(t, event.GetAsset())

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetAsset().ToXdrAsset().Equals(asset))
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
				assert.Nil(t, event.GetAsset()) // asset is nil for non-SAC events
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
				assert.NotNil(t, event.GetAsset())

				asset := assetItem.(xdr.Asset)
				assert.True(t, event.GetAsset().ToXdrAsset().Equals(asset))
			},
		},
		{
			name:       "Transfer SEP-41 Token Event - Amount is 128bits",
			eventType:  TransferEvent,
			addr1:      someContract1, // from
			addr2:      someContract2, // to
			amount128:  &xdr.Int128Parts{Hi: 5000, Lo: 1000},
			isSacEvent: false,
			validateEvent: func(t *testing.T, event *TokenTransferEvent, from, to string, amount string, _ interface{}) {
				assert.NotNil(t, event.GetTransfer())
				assert.Equal(t, from, event.GetTransfer().From)
				assert.Equal(t, to, event.GetTransfer().To)
				assert.Equal(t, amount, event.GetTransfer().Amount)
				assert.Nil(t, event.GetAsset()) // asset is nil for non-SAC events
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with multiple assets
			var assets []interface{}
			var contractId *xdr.ContractId

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
					contractId = contractIdFromAsset(asset)
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
						tc.amount128,
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
						tc.amount128,
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
						tc.amount128,
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
						tc.amount128,
						assetStr,
						contractId,
					)
				}

				event, err := processor.parseEvent(someTx, &someOperationIndex, contractEvent)

				require.NoError(t, err)
				require.NotNil(t, event)

				var amountStr string
				if tc.amount128 != nil {
					amountStr = amount.String128Raw(*tc.amount128)
				} else {
					amountStr = fmt.Sprintf("%d", tc.amount)
				}
				tc.validateEvent(t, event, tc.addr1, tc.addr2, amountStr, assetItem)
			}
		})
	}
}

func TestInvalidEvents(t *testing.T) {
	testCases := []struct {
		name           string
		setupEvent     func() xdr.ContractEvent
		expectedErrMsg string
	}{
		{
			name: "Invalid contract event type",
			setupEvent: func() xdr.ContractEvent {
				// Use a non-contract event type
				event := createContractEvent(TransferEvent, randomAccount, someContract1, 1000, nil, "asset", &someContractHash1)
				event.Type = xdr.ContractEventTypeSystem // Invalid type
				return event
			},
			expectedErrMsg: "invalid contractEvent",
		},
		{
			name: "Missing contract ID",
			setupEvent: func() xdr.ContractEvent {
				event := createContractEvent(TransferEvent, randomAccount, someContract1, 1000, nil, "asset", nil)
				return event
			},
			expectedErrMsg: "invalid contractEvent",
		},
		{
			name: "Invalid body version",
			setupEvent: func() xdr.ContractEvent {
				event := createContractEvent(TransferEvent, randomAccount, someContract1, 1000, nil, "asset", &someContractHash1)
				event.Body.V = 1 // Invalid version
				return event
			},
			expectedErrMsg: "invalid contractEvent",
		},
		{
			name: "Insufficient topics",
			setupEvent: func() xdr.ContractEvent {
				// Create event with only one topic (the function name)
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "insufficient topics in contract event",
		},
		{
			name: "Invalid function name type",
			setupEvent: func() xdr.ContractEvent {
				// Use a string instead of a symbol for function name
				topics := []xdr.ScVal{
					createString(TransferEvent), // Should be a symbol
					createAddress(randomAccount),
					createAddress(someContract1),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid function name",
		},
		{
			name: "Invalid amount format",
			setupEvent: func() xdr.ContractEvent {
				event := createContractEvent(TransferEvent, randomAccount, someContract1, 1000, nil, "asset", &someContractHash1)
				// Replace the amount with a string value instead of Int128
				event.Body.V0.Data = createString("1000")
				return event
			},
			expectedErrMsg: "invalid event amount",
		},
		{
			name: "Transfer: Too few topics",
			setupEvent: func() xdr.ContractEvent {
				// Only include function name and from address, missing to address
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createAddress(randomAccount),
					// Missing to address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "transfer event requires minimum 3 topics, found: 2",
		},
		{
			name: "Transfer: Invalid from address",
			setupEvent: func() xdr.ContractEvent {
				// Use an invalid value for from address (not an address)
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createString("not an address"), // Invalid from address
					createAddress(someContract1),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid fromAddress",
		},
		{
			name: "Transfer: Invalid to address",
			setupEvent: func() xdr.ContractEvent {
				// Use an invalid value for to address (not an address)
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createAddress(randomAccount),
					createString("not an address"), // Invalid to address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid toAddress",
		},
		{
			name: "Mint: Too few topics",
			setupEvent: func() xdr.ContractEvent {
				// Only include function name and admin, missing to address
				topics := []xdr.ScVal{
					createSymbol(MintEvent),
					createAddress(randomAccount),
					// Missing to address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "mint event requires minimum 3 topics",
		},
		{
			name: "Mint: Invalid admin address",
			setupEvent: func() xdr.ContractEvent {
				// Use an invalid value for admin address (not an address)
				topics := []xdr.ScVal{
					createSymbol(MintEvent),
					createString("not an address"), // Invalid admin address
					createAddress(someContract1),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid adminAddress",
		},
		{
			name: "Mint: Invalid to address",
			setupEvent: func() xdr.ContractEvent {
				// Use an invalid value for to address (not an address)
				topics := []xdr.ScVal{
					createSymbol(MintEvent),
					createAddress(randomAccount),
					createString("not an address"), // Invalid to address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid toAddress",
		},
		{
			name: "Clawback: Too few topics",
			setupEvent: func() xdr.ContractEvent {
				// Only include function name and admin, missing from address
				topics := []xdr.ScVal{
					createSymbol(ClawbackEvent),
					createAddress(randomAccount),
					// Missing from address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "clawback event requires minimum 3 topics",
		},
		{
			name: "Clawback: Invalid admin address",
			setupEvent: func() xdr.ContractEvent {
				// Use an invalid value for admin address (not an address)
				topics := []xdr.ScVal{
					createSymbol(ClawbackEvent),
					createString("not an address"), // Invalid admin address
					createAddress(randomAccount),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid adminAddress",
		},
		{
			name: "Clawback: Invalid from address",
			setupEvent: func() xdr.ContractEvent {
				// Use an invalid value for from address (not an address)
				topics := []xdr.ScVal{
					createSymbol(ClawbackEvent),
					createAddress(someContract1),
					createString("not an address"), // Invalid from address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid fromAddress",
		},
		{
			name: "Burn: Too few topics",
			// this is true for any event really. Minimum number of topics is 2.
			setupEvent: func() xdr.ContractEvent {
				// Only include function name, missing from address
				topics := []xdr.ScVal{
					createSymbol(BurnEvent),
					// Missing from address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "insufficient topics in contract event",
		},
		{
			name: "Burn: Invalid from address",
			setupEvent: func() xdr.ContractEvent {
				// Use an invalid value for from address (not an address)
				topics := []xdr.ScVal{
					createSymbol(BurnEvent),
					createString("not an address"), // Invalid from address
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "invalid fromAddress",
		},
		{
			name: "Unsupported event type",
			setupEvent: func() xdr.ContractEvent {
				return createContractEvent(
					"unknown_event", // Unsupported event type
					randomAccount,
					someContract1,
					1000,
					nil,
					"asset",
					&someContractHash1,
				)
			},
			expectedErrMsg: "unsupported custom token event type",
		},
		{
			name: "Invalid SAC asset string",
			setupEvent: func() xdr.ContractEvent {
				// Create transfer event with invalid asset string that can't be parsed
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createAddress(randomAccount),
					createAddress(someContract1),
					createString("not:a:valid:asset"), // Invalid asset string
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: &someContractHash1,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			expectedErrMsg: "", // This should not error as SAC validation is only for enhancement
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contractEvent := tc.setupEvent()
			event, err := processor.parseEvent(someTx, &someOperationIndex, contractEvent)

			if tc.expectedErrMsg == "" {
				// If no error is expected, the test should pass
				require.NoError(t, err)
				require.NotNil(t, event)
			} else {
				// If an error is expected
				require.Error(t, err)
				assert.Nil(t, event)
				assert.Contains(t, err.Error(), tc.expectedErrMsg, "Error message should contain expected text")

				// Verify it's the right error type
				_, ok := err.(ErrNotSep41TokenEvent)
				assert.True(t, ok, "Error should be of type ErrNotSep41TokenEvent")
			}
		})
	}
}

func TestSacAssetValidation(t *testing.T) {
	// None of the test fixtures here will result in errors
	// They simply might not pass the additional validation required to qualify as a SAC event, is all
	xlmContractId := contractIdFromAsset(xlmAsset)

	testCases := []struct {
		name              string
		setupEvent        func() xdr.ContractEvent
		isAssetSetInEvent bool // Whether to check if the asset is set in the event
	}{
		{
			name: "Valid SAC asset with matching contract ID",
			setupEvent: func() xdr.ContractEvent {
				return createContractEvent(
					TransferEvent,
					randomAccount,
					someContract1,
					1000,
					nil,
					xlmAsset.StringCanonical(),
					xlmContractId,
				)
			},
			isAssetSetInEvent: true,
		},
		{
			name: "Valid SAC asset string but mismatched contract ID",
			setupEvent: func() xdr.ContractEvent {
				// Use valid asset string but wrong contract ID
				return createContractEvent(
					TransferEvent,
					randomAccount,
					someContract1,
					1000,
					nil,
					xlmAsset.StringCanonical(),
					&someContractHash2, // Different from xlmContractId
				)
			},
			isAssetSetInEvent: false, // Asset should not be set due to mismatch
		},
		{
			name: "Valid SAC asset with correct format but not exactly 4 topics",
			setupEvent: func() xdr.ContractEvent {
				// Add an extra topic to make it 5 instead of 4
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createAddress(randomAccount),
					createAddress(someContract1),
					createString(xlmAsset.StringCanonical()),
					createString("extra topic"),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: xlmContractId,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			isAssetSetInEvent: false, // Should not set asset due to wrong topic count
		},
		{
			name: "Valid burn event with exactly 3 topics and valid asset",
			setupEvent: func() xdr.ContractEvent {
				topics := []xdr.ScVal{
					createSymbol(BurnEvent),
					createAddress(randomAccount),
					createString(xlmAsset.StringCanonical()),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: xlmContractId,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			isAssetSetInEvent: true, // Should set asset for burn with 3 topics
		},
		{
			name: "Valid token event but last topic is not a string",
			setupEvent: func() xdr.ContractEvent {
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createAddress(randomAccount),
					createAddress(someContract1),
					createInt128(12345), // Not a string
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: xlmContractId,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			isAssetSetInEvent: false, // Should not set asset due to non-string
		},
		{
			name: "Valid token event but asset string is empty",
			setupEvent: func() xdr.ContractEvent {
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createAddress(randomAccount),
					createAddress(someContract1),
					createString(""), // Empty string
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: xlmContractId,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			isAssetSetInEvent: false, // Should not set asset due to empty string
		},
		{
			name: "Valid token event but BuildAssets returns multiple assets",
			setupEvent: func() xdr.ContractEvent {
				// Make it so that some custom token emits a string that is a SEP-11 representation of multiple assets
				topics := []xdr.ScVal{
					createSymbol(TransferEvent),
					createAddress(randomAccount),
					createAddress(someContract1),
					createString(
						fmt.Sprintf("%s,%s",
							xlmAsset.StringCanonical(), xlmAsset.StringCanonical()),
					),
				}

				return xdr.ContractEvent{
					Type:       xdr.ContractEventTypeContract,
					ContractId: xlmContractId,
					Body: xdr.ContractEventBody{
						V: 0,
						V0: &xdr.ContractEventV0{
							Topics: topics,
							Data:   createInt128(1000),
						},
					},
				}
			},
			isAssetSetInEvent: false, // Asset shouldn't be set due to buildAssets returning multiple assets
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contractEvent := tc.setupEvent()
			event, err := processor.parseEvent(someTx, &someOperationIndex, contractEvent)

			require.NoError(t, err, "Should not error for this test case")
			require.NotNil(t, event, "Event should be returned")

			// Check if we need to verify asset is set or not set
			if tc.isAssetSetInEvent {
				eventAsset := event.GetAsset()

				assert.NotNil(t, eventAsset, "Asset should be set for this event")
				assert.True(t, eventAsset.ToXdrAsset().Equals(xlmAsset))
			} else {
				eventAsset := event.GetAsset()
				assert.Nil(t, eventAsset, "Asset should not be set for this event")
			}

		})
	}
}
