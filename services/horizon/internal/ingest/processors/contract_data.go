package processors

import (
	"math/big"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

const (
	scDecimalPrecision = 7
)

var (
	// https://github.com/stellar/rs-soroban-env/blob/v0.0.16/soroban-env-host/src/native_contract/token/public_types.rs#L22
	nativeAssetSym = xdr.ScSymbol("Native")
	// these are storage DataKey enum
	// https://github.com/stellar/rs-soroban-env/blob/v0.0.16/soroban-env-host/src/native_contract/token/storage_types.rs#L23
	balanceMetadataSym = xdr.ScSymbol("Balance")
	metadataSym        = xdr.ScSymbol("METADATA")
	metadataNameSym    = xdr.ScSymbol("name")
	metadataSymbolSym  = xdr.ScSymbol("symbol")
	adminSym           = xdr.ScSymbol("Admin")
	issuerSym          = xdr.ScSymbol("issuer")
	assetCodeSym       = xdr.ScSymbol("asset_code")
	alphaNum4Sym       = xdr.ScSymbol("AlphaNum4")
	alphaNum12Sym      = xdr.ScSymbol("AlphaNum12")
	decimalSym         = xdr.ScSymbol("decimal")
	assetInfoSym       = xdr.ScSymbol("AssetInfo")
	decimalVal         = xdr.Uint32(scDecimalPrecision)
	assetInfoVec       = &xdr.ScVec{
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &assetInfoSym,
		},
	}
	assetInfoKey = xdr.ScVal{
		Type: xdr.ScValTypeScvVec,
		Vec:  &assetInfoVec,
	}
)

// AssetFromContractData takes a ledger entry and verifies if the ledger entry
// corresponds to the asset info entry written to contract storage by the Stellar
// Asset Contract upon initialization.
//
// Note that AssetFromContractData will ignore forged asset info entries by
// deriving the Stellar Asset Contract ID from the asset info entry and comparing
// it to the contract ID found in the ledger entry.
//
// If the given ledger entry is a verified asset info entry,
// AssetFromContractData will return the corresponding Stellar asset. Otherwise,
// it returns nil.
//
// References:
// https://github.com/stellar/rs-soroban-env/blob/v0.0.16/soroban-env-host/src/native_contract/token/public_types.rs#L21
// https://github.com/stellar/rs-soroban-env/blob/v0.0.16/soroban-env-host/src/native_contract/token/asset_info.rs#L6
// https://github.com/stellar/rs-soroban-env/blob/v0.0.16/soroban-env-host/src/native_contract/token/contract.rs#L115
//
// The asset info in `ContractData` entry takes the following form:
//
//   - Instance storage - it's part of contract instance data storage
//
//   - Key: a vector with one element, which is the symbol "AssetInfo"
//
//     ScVal{ Vec: ScVec({ ScVal{ Sym: ScSymbol("AssetInfo") }})}
//
//   - Value: a map with two key-value pairs: code and issuer
//
//     ScVal{ Map: ScMap(
//     { ScVal{ Sym: ScSymbol("asset_code") } -> ScVal{ Str: ScString(...) } },
//     { ScVal{ Sym: ScSymbol("issuer") } -> ScVal{ Bytes: ScBytes(...) } }
//     )}
func AssetFromContractData(ledgerEntry xdr.LedgerEntry, passphrase string) *xdr.Asset {
	contractData, ok := ledgerEntry.Data.GetContractData()
	if !ok {
		return nil
	}
	if contractData.Key.Type != xdr.ScValTypeScvLedgerKeyContractInstance {
		return nil
	}
	contractInstanceData, ok := contractData.Val.GetInstance()
	if !ok || contractInstanceData.Storage == nil {
		return nil
	}

	// we don't support asset stats for lumens
	nativeAssetContractID, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	if err != nil || (contractData.Contract.ContractId != nil && (*contractData.Contract.ContractId) == nativeAssetContractID) {
		return nil
	}

	var assetInfo *xdr.ScVal
	for _, mapEntry := range *contractInstanceData.Storage {
		if mapEntry.Key.Equals(assetInfoKey) {
			// clone the map entry to avoid reference to loop iterator
			mapValXdr, cloneErr := mapEntry.Val.MarshalBinary()
			if cloneErr != nil {
				return nil
			}
			assetInfo = &xdr.ScVal{}
			cloneErr = assetInfo.UnmarshalBinary(mapValXdr)
			if cloneErr != nil {
				return nil
			}
			break
		}
	}

	if assetInfo == nil {
		return nil
	}

	vecPtr, ok := assetInfo.GetVec()
	if !ok || vecPtr == nil || len(*vecPtr) != 2 {
		return nil
	}
	vec := *vecPtr

	sym, ok := vec[0].GetSym()
	if !ok {
		return nil
	}
	switch sym {
	case "AlphaNum4":
	case "AlphaNum12":
	default:
		return nil
	}

	var assetCode, assetIssuer string
	assetMapPtr, ok := vec[1].GetMap()
	if !ok || assetMapPtr == nil || len(*assetMapPtr) != 2 {
		return nil
	}
	assetMap := *assetMapPtr

	assetCodeEntry, assetIssuerEntry := assetMap[0], assetMap[1]
	if sym, ok = assetCodeEntry.Key.GetSym(); !ok || sym != assetCodeSym {
		return nil
	}
	assetCodeSc, ok := assetCodeEntry.Val.GetStr()
	if !ok {
		return nil
	}
	if assetCode = string(assetCodeSc); assetCode == "" {
		return nil
	}

	if sym, ok = assetIssuerEntry.Key.GetSym(); !ok || sym != issuerSym {
		return nil
	}
	assetIssuerSc, ok := assetIssuerEntry.Val.GetBytes()
	if !ok {
		return nil
	}
	assetIssuer, err = strkey.Encode(strkey.VersionByteAccountID, assetIssuerSc)
	if err != nil {
		return nil
	}

	asset, err := xdr.NewCreditAsset(assetCode, assetIssuer)
	if err != nil {
		return nil
	}

	expectedID, err := asset.ContractID(passphrase)
	if err != nil {
		return nil
	}
	if contractData.Contract.ContractId == nil || expectedID != *(contractData.Contract.ContractId) {
		return nil
	}

	return &asset
}

// ContractBalanceFromContractData takes a ledger entry and verifies that the
// ledger entry corresponds to the balance entry written to contract storage by
// the Stellar Asset Contract.
//
// Reference:
//
//	https://github.com/stellar/rs-soroban-env/blob/da325551829d31dcbfa71427d51c18e71a121c5f/soroban-env-host/src/native_contract/token/storage_types.rs#L11-L24
func ContractBalanceFromContractData(ledgerEntry xdr.LedgerEntry, passphrase string) ([32]byte, *big.Int, bool) {
	contractData, ok := ledgerEntry.Data.GetContractData()
	if !ok {
		return [32]byte{}, nil, false
	}

	// we don't support asset stats for lumens
	nativeAssetContractID, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	if err != nil || (contractData.Contract.ContractId != nil && *contractData.Contract.ContractId == nativeAssetContractID) {
		return [32]byte{}, nil, false
	}

	keyEnumVecPtr, ok := contractData.Key.GetVec()
	if !ok || keyEnumVecPtr == nil {
		return [32]byte{}, nil, false
	}
	keyEnumVec := *keyEnumVecPtr
	if len(keyEnumVec) != 2 || !keyEnumVec[0].Equals(
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &balanceMetadataSym,
		},
	) {
		return [32]byte{}, nil, false
	}

	scAddress, ok := keyEnumVec[1].GetAddress()
	if !ok {
		return [32]byte{}, nil, false
	}

	holder, ok := scAddress.GetContractId()
	if !ok {
		return [32]byte{}, nil, false
	}

	balanceMapPtr, ok := contractData.Val.GetMap()
	if !ok || balanceMapPtr == nil {
		return [32]byte{}, nil, false
	}
	balanceMap := *balanceMapPtr
	if !ok || len(balanceMap) != 3 {
		return [32]byte{}, nil, false
	}

	var keySym xdr.ScSymbol
	if keySym, ok = balanceMap[0].Key.GetSym(); !ok || keySym != "amount" {
		return [32]byte{}, nil, false
	}
	if keySym, ok = balanceMap[1].Key.GetSym(); !ok || keySym != "authorized" ||
		!balanceMap[1].Val.IsBool() {
		return [32]byte{}, nil, false
	}
	if keySym, ok = balanceMap[2].Key.GetSym(); !ok || keySym != "clawback" ||
		!balanceMap[2].Val.IsBool() {
		return [32]byte{}, nil, false
	}
	amount, ok := balanceMap[0].Val.GetI128()
	if !ok {
		return [32]byte{}, nil, false
	}

	// amount cannot be negative
	// https://github.com/stellar/rs-soroban-env/blob/a66f0815ba06a2f5328ac420950690fd1642f887/soroban-env-host/src/native_contract/token/balance.rs#L92-L93
	if int64(amount.Hi) < 0 {
		return [32]byte{}, nil, false
	}
	amt := new(big.Int).Lsh(new(big.Int).SetInt64(int64(amount.Hi)), 64)
	amt.Add(amt, new(big.Int).SetUint64(uint64(amount.Lo)))
	return holder, amt, true
}

func metadataObjFromAsset(isNative bool, code, issuer string) (*xdr.ScMap, error) {
	assetInfoVecKey := &xdr.ScVec{
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &assetInfoSym,
		},
	}

	if isNative {
		nativeVec := &xdr.ScVec{
			xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &nativeAssetSym,
			},
		}
		return &xdr.ScMap{
			xdr.ScMapEntry{
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvVec,
					Vec:  &assetInfoVecKey,
				},
				Val: xdr.ScVal{
					Type: xdr.ScValTypeScvVec,
					Vec:  &nativeVec,
				},
			},
		}, nil
	}

	nameVal := xdr.ScString(code + ":" + issuer)
	symbolVal := xdr.ScString(code)
	metaDataMap := &xdr.ScMap{
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &decimalSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvU32,
				U32:  &decimalVal,
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &metadataNameSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvString,
				Str:  &nameVal,
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &metadataSymbolSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvString,
				Str:  &symbolVal,
			},
		},
	}

	adminVec := &xdr.ScVec{
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &adminSym,
		},
	}

	adminAccountId := xdr.MustAddress(issuer)
	assetCodeVal := xdr.ScString(code)
	issuerBytes, err := strkey.Decode(strkey.VersionByteAccountID, issuer)
	if err != nil {
		return nil, err
	}

	assetIssuerBytes := xdr.ScBytes(issuerBytes)
	assetInfoMap := &xdr.ScMap{
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &assetCodeSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvString,
				Str:  &assetCodeVal,
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &issuerSym,
			},
			Val: xdr.ScVal{
				Type:  xdr.ScValTypeScvBytes,
				Bytes: &assetIssuerBytes,
			},
		},
	}

	alphaNumSym := alphaNum4Sym
	if len(code) > 4 {
		alphaNumSym = alphaNum12Sym
	}
	assetInfoVecVal := &xdr.ScVec{
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &alphaNumSym,
		},
		xdr.ScVal{
			Type: xdr.ScValTypeScvMap,
			Map:  &assetInfoMap,
		},
	}

	storageMap := &xdr.ScMap{
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &metadataSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvMap,
				Map:  &metaDataMap,
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvVec,
				Vec:  &adminVec,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvAddress,
				Address: &xdr.ScAddress{
					AccountId: &adminAccountId,
				},
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvVec,
				Vec:  &assetInfoVecKey,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvVec,
				Vec:  &assetInfoVecVal,
			},
		},
	}

	return storageMap, nil
}

// AssetToContractData is the inverse of AssetFromContractData. It creates a
// ledger entry containing the asset info entry written to contract storage by the
// Stellar Asset Contract.
//
// Warning: Only for use in tests. This does not create the accompanied TTLEntry which would typically be created by core.
func AssetToContractData(isNative bool, code, issuer string, contractID [32]byte) (xdr.LedgerEntryData, error) {
	storageMap, err := metadataObjFromAsset(isNative, code, issuer)
	if err != nil {
		return xdr.LedgerEntryData{}, err
	}
	var ContractIDHash xdr.Hash = contractID

	return xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.ContractDataEntry{
			Contract: xdr.ScAddress{
				Type:       xdr.ScAddressTypeScAddressTypeContract,
				AccountId:  nil,
				ContractId: &ContractIDHash,
			},
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvLedgerKeyContractInstance,
			},
			Durability: xdr.ContractDataDurabilityPersistent,
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvContractInstance,
				Instance: &xdr.ScContractInstance{
					Executable: xdr.ContractExecutable{
						Type: xdr.ContractExecutableTypeContractExecutableStellarAsset,
					},
					Storage: storageMap,
				},
			},
		},
	}, nil
}

// BalanceToContractData is the inverse of ContractBalanceFromContractData. It
// creates a ledger entry containing the asset balance of a contract holder
// written to contract storage by the Stellar Asset Contract.
//
// Warning: Only for use in tests. This does not create the accompanied TTLEntry which would typically be created by core.
func BalanceToContractData(assetContractId, holderID [32]byte, amt uint64) xdr.LedgerEntryData {
	return balanceToContractData(assetContractId, holderID, xdr.Int128Parts{
		Lo: xdr.Uint64(amt),
		Hi: 0,
	})
}

// ContractBalanceLedgerKey constructs the ledger key corresponding to the
// asset balance of a contract holder written to contract storage by the
// Stellar Asset Contract.
func ContractBalanceLedgerKey(assetContractId, holderID [32]byte) xdr.LedgerKey {
	holder := xdr.Hash(holderID)
	scAddress := &xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: &holder,
	}
	keyVec := &xdr.ScVec{
		xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &balanceMetadataSym},
		xdr.ScVal{Type: xdr.ScValTypeScvAddress, Address: scAddress},
	}
	var contractIDHash xdr.Hash = assetContractId
	return xdr.LedgerKey{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.LedgerKeyContractData{
			Contract: xdr.ScAddress{
				Type:       xdr.ScAddressTypeScAddressTypeContract,
				ContractId: &contractIDHash,
			},
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvVec,
				Vec:  &keyVec,
			},
			Durability: xdr.ContractDataDurabilityPersistent,
		},
	}
}

func balanceToContractData(assetContractId, holderID [32]byte, amt xdr.Int128Parts) xdr.LedgerEntryData {
	ledgerKey := ContractBalanceLedgerKey(assetContractId, holderID)

	amountSym := xdr.ScSymbol("amount")
	authorizedSym := xdr.ScSymbol("authorized")
	clawbackSym := xdr.ScSymbol("clawback")
	trueIc := true
	dataMap := &xdr.ScMap{
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &amountSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvI128,
				I128: &amt,
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &authorizedSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvBool,
				B:    &trueIc,
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &clawbackSym,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvBool,
				B:    &trueIc,
			},
		},
	}

	contractData := ledgerKey.MustContractData()
	return xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.ContractDataEntry{
			Contract:   contractData.Contract,
			Key:        contractData.Key,
			Durability: contractData.Durability,
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvMap,
				Map:  &dataMap,
			},
		},
	}
}
