package processors

import (
	"math/big"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

var (
	nativeAssetSym     = xdr.ScSymbol("Native")
	balanceMetadataSym = xdr.ScSymbol("Balance")
	assetMetadataSym   = xdr.ScSymbol("Metadata")
	assetMetadataVec   = &xdr.ScVec{
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &assetMetadataSym,
		},
	}
	assetMetadataKey = xdr.ScVal{
		Type: xdr.ScValTypeScvVec,
		Vec:  &assetMetadataVec,
	}
)

// AssetFromContractData takes a ledger entry and verifies if the ledger entry
// corresponds to the asset metadata written to contract storage by the Stellar
// Asset Contract upon initialization.
//
// Note that AssetFromContractData will ignore forged asset metadata entries by
// deriving the Stellar Asset Contract ID from the asset metadata and comparing
// it to the contract ID found in the ledger entry.
//
// If the given ledger entry is a verified asset metadata entry,
// AssetFromContractData will return the corresponding Stellar asset. Otherwise,
// it returns nil.
//
// References:
//
//	https://github.com/stellar/rs-soroban-env/blob/da325551829d31dcbfa71427d51c18e71a121c5f/soroban-env-host/src/native_contract/token/public_types.rs#L21
//	https://github.com/stellar/rs-soroban-env/blob/da325551829d31dcbfa71427d51c18e71a121c5f/soroban-env-host/src/native_contract/token/metadata.rs#L8
//	https://github.com/stellar/rs-soroban-env/blob/da325551829d31dcbfa71427d51c18e71a121c5f/soroban-env-host/src/native_contract/token/contract.rs#L108
//
// The `ContractData` entry takes the following form:
//
//   - Key: a vector with one element, which is the symbol "Metadata"
//
//     ScVal{ Vec: ScVec({ ScVal{ Sym: ScSymbol("metadata") }})}
//
//   - Value: a map with two key-value pairs: code and issuer
//
//     ScVal{ Map: ScMap(
//     { ScVal{ Sym: ScSymbol("asset_code") } -> ScVal{ Bytes: ScBytes(...) } },
//     { ScVal{ Sym: ScSymbol("asset_code") } -> ScVal{ Bytes: ScBytes(...) } }
//     )}
func AssetFromContractData(ledgerEntry xdr.LedgerEntry, passphrase string) *xdr.Asset {
	contractData, ok := ledgerEntry.Data.GetContractData()
	if !ok {
		return nil
	}

	// we don't support asset stats for lumens
	nativeAssetContractID, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	if err != nil || contractData.ContractId == nativeAssetContractID {
		return nil
	}

	if !contractData.Key.Equals(assetMetadataKey) {
		return nil
	}

	vecPtr, ok := contractData.Val.GetVec()
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
	if sym, ok = assetCodeEntry.Key.GetSym(); !ok || sym != "asset_code" {
		return nil
	}
	bin, ok := assetCodeEntry.Val.GetBytes()
	if !ok || bin == nil {
		return nil
	}
	assetCode = string(bin)

	if sym, ok = assetIssuerEntry.Key.GetSym(); !ok || sym != "issuer" {
		return nil
	}
	bin, ok = assetIssuerEntry.Val.GetBytes()
	if !ok || bin == nil {
		return nil
	}
	assetIssuer, err = strkey.Encode(strkey.VersionByteAccountID, bin)
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
	if expectedID != contractData.ContractId {
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
	if err != nil || contractData.ContractId == nativeAssetContractID {
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

func metadataObjFromAsset(isNative bool, code, issuer string) (*xdr.ScVec, error) {
	if isNative {
		return &xdr.ScVec{
			xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &nativeAssetSym,
			},
		}, nil
	}

	var assetCodeLength int
	var symbol xdr.ScSymbol
	if len(code) <= 4 {
		symbol = "AlphaNum4"
		assetCodeLength = 4
	} else {
		symbol = "AlphaNum12"
		assetCodeLength = 12
	}

	assetCodeSymbol := xdr.ScSymbol("asset_code")
	assetCodeBytes := make([]byte, assetCodeLength)
	copy(assetCodeBytes, code)

	issuerSymbol := xdr.ScSymbol("issuer")
	issuerBytes, err := strkey.Decode(strkey.VersionByteAccountID, issuer)
	if err != nil {
		return nil, err
	}

	mapObj := &xdr.ScMap{
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &assetCodeSymbol,
			},
			Val: xdr.ScVal{
				Type:  xdr.ScValTypeScvBytes,
				Bytes: (*xdr.ScBytes)(&assetCodeBytes),
			},
		},
		xdr.ScMapEntry{
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &issuerSymbol,
			},
			Val: xdr.ScVal{
				Type:  xdr.ScValTypeScvBytes,
				Bytes: (*xdr.ScBytes)(&issuerBytes),
			},
		},
	}

	return &xdr.ScVec{
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &symbol,
		},
		xdr.ScVal{
			Type: xdr.ScValTypeScvMap,
			Map:  &mapObj,
		},
	}, nil
}

// AssetToContractData is the inverse of AssetFromContractData. It creates a
// ledger entry containing the asset metadata written to contract storage by the
// Stellar Asset Contract.
func AssetToContractData(isNative bool, code, issuer string, contractID [32]byte) (xdr.LedgerEntryData, error) {
	vec, err := metadataObjFromAsset(isNative, code, issuer)
	if err != nil {
		return xdr.LedgerEntryData{}, err
	}
	return xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.ContractDataEntry{
			ContractId: contractID,
			Key:        assetMetadataKey,
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvVec,
				Vec:  &vec,
			},
		},
	}, nil
}

// BalanceToContractData is the inverse of ContractBalanceFromContractData. It
// creates a ledger entry containing the asset balance of a contract holder
// written to contract storage by the Stellar Asset Contract.
func BalanceToContractData(assetContractId, holderID [32]byte, amt uint64) xdr.LedgerEntryData {
	return balanceToContractData(assetContractId, holderID, xdr.Int128Parts{
		Lo: xdr.Uint64(amt),
		Hi: 0,
	})
}

func balanceToContractData(assetContractId, holderID [32]byte, amt xdr.Int128Parts) xdr.LedgerEntryData {
	holder := xdr.Hash(holderID)
	scAddress := &xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: &holder,
	}
	keyVec := &xdr.ScVec{
		xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &balanceMetadataSym},
		xdr.ScVal{Type: xdr.ScValTypeScvAddress, Address: scAddress},
	}

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

	return xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.ContractDataEntry{
			ContractId: assetContractId,
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvVec,
				Vec:  &keyVec,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvMap,
				Map:  &dataMap,
			},
		},
	}
}
