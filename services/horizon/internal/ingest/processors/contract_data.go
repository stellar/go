package processors

import (
	"math/big"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

var (
	balanceMetadataSym = xdr.ScSymbol("Balance")
	assetMetadataSym   = xdr.ScSymbol("Metadata")
	assetMetadataObj   = &xdr.ScObject{
		Type: xdr.ScObjectTypeScoVec,
		Vec: &xdr.ScVec{
			xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &assetMetadataSym,
			},
		},
	}
	assetMetadataKey = xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &assetMetadataObj,
	}
)

// AssetFromContractData takes a ledger entry and verifies if the ledger entry corresponds
// to the asset metadata written to contract storage by the Stellar Asset Contract upon
// initialization. See:
// https://github.com/stellar/rs-soroban-env/blob/5695440da452837555d8f7f259cc33341fdf07b0/soroban-env-host/src/native_contract/token/public_types.rs#L21
// https://github.com/stellar/rs-soroban-env/blob/5695440da452837555d8f7f259cc33341fdf07b0/soroban-env-host/src/native_contract/token/metadata.rs#L8
// https://github.com/stellar/rs-soroban-env/blob/5695440da452837555d8f7f259cc33341fdf07b0/soroban-env-host/src/native_contract/token/contract.rs#L108
// Note that AssetFromContractData will ignore forged asset metadata entries by deriving
// the Stellar Asset Contract id from the asset metadata and comparing it to the contract
// id found in the ledger entry.
// If the given ledger entry is a verified asset metadata entry AssetFromContractData will
// return the corresponding Stellar asset. Otherwise, AssetFromContractData will return nil.
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
	obj, ok := contractData.Val.GetObj()
	if !ok || obj == nil {
		return nil
	}

	vec, ok := obj.GetVec()
	if !ok || len(vec) <= 0 {
		return nil
	}

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
	if len(vec) != 2 {
		return nil
	}

	var assetCode, assetIssuer string
	obj, ok = vec[1].GetObj()
	if !ok || obj == nil {
		return nil
	}
	assetMap, ok := obj.GetMap()
	if !ok || len(assetMap) != 2 {
		return nil
	}
	assetCodeEntry, assetIssuerEntry := assetMap[0], assetMap[1]
	if sym, ok = assetCodeEntry.Key.GetSym(); !ok || sym != "asset_code" {
		return nil
	}
	if obj, ok = assetCodeEntry.Val.GetObj(); !ok || obj == nil {
		return nil
	}
	bin, ok := obj.GetBin()
	if !ok {
		return nil
	}
	assetCode = string(bin)

	if sym, ok = assetIssuerEntry.Key.GetSym(); !ok || sym != "issuer" {
		return nil
	}
	if obj, ok = assetIssuerEntry.Val.GetObj(); !ok || obj == nil {
		return nil
	}
	bin, ok = obj.GetBin()
	if !ok {
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

// ContractBalanceFromContractData takes a ledger entry and verifies if the ledger entry corresponds
// to the balance entry written to contract storage by the Stellar Asset Contract. See:
// https://github.com/stellar/rs-soroban-env/blob/5695440da452837555d8f7f259cc33341fdf07b0/soroban-env-host/src/native_contract/token/storage_types.rs#L11-L24
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

	keyObj, ok := contractData.Key.GetObj()
	if !ok || keyObj == nil {
		return [32]byte{}, nil, false
	}
	keyEnumVec, ok := keyObj.GetVec()
	if !ok || len(keyEnumVec) != 2 || !keyEnumVec[0].Equals(
		xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &balanceMetadataSym},
	) {
		return [32]byte{}, nil, false
	}
	addressObj, ok := keyEnumVec[1].GetObj()
	if !ok || addressObj == nil {
		return [32]byte{}, nil, false
	}
	scAddress, ok := addressObj.GetAddress()
	if !ok {
		return [32]byte{}, nil, false
	}
	holder, ok := scAddress.GetContractId()
	if !ok {
		return [32]byte{}, nil, false
	}

	obj, ok := contractData.Val.GetObj()
	if !ok || obj == nil {
		return [32]byte{}, nil, false
	}
	balanceMap, ok := obj.GetMap()
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
	amountObj, ok := balanceMap[0].Val.GetObj()
	if !ok || amountObj == nil {
		return [32]byte{}, nil, false
	}
	amount, ok := amountObj.GetI128()
	if !ok {
		return [32]byte{}, nil, false
	}
	// amount cannot be negative
	// https://github.com/stellar/rs-soroban-env/blob/a66f0815ba06a2f5328ac420950690fd1642f887/soroban-env-host/src/native_contract/token/balance.rs#L92-L93
	if int64(amount.Hi) < 0 {
		return [32]byte{}, nil, false
	}
	amt := new(big.Int).Lsh(new(big.Int).SetUint64(uint64(amount.Hi)), 64)
	amt.Add(amt, new(big.Int).SetUint64(uint64(amount.Lo)))
	return holder, amt, true
}

func metadataObjFromAsset(isNative bool, code, issuer string) (*xdr.ScObject, error) {
	if isNative {
		symbol := xdr.ScSymbol("Native")
		metadataObj := &xdr.ScObject{
			Type: xdr.ScObjectTypeScoVec,
			Vec: &xdr.ScVec{
				xdr.ScVal{
					Type: xdr.ScValTypeScvSymbol,
					Sym:  &symbol,
				},
			},
		}
		return metadataObj, nil
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
	assetCodeBytes := make([]byte, assetCodeLength)
	copy(assetCodeBytes, code)
	assetCodeSymbol := xdr.ScSymbol("asset_code")
	issuerSymbol := xdr.ScSymbol("issuer")

	assetCodeObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &assetCodeBytes,
	}
	issuerBytes, err := strkey.Decode(strkey.VersionByteAccountID, issuer)
	if err != nil {
		return nil, err
	}
	issuerObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &issuerBytes,
	}

	mapObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoMap,
		Map: &xdr.ScMap{
			xdr.ScMapEntry{
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvSymbol,
					Sym:  &assetCodeSymbol,
				},
				Val: xdr.ScVal{
					Type: xdr.ScValTypeScvObject,
					Obj:  &assetCodeObj,
				},
			},
			xdr.ScMapEntry{
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvSymbol,
					Sym:  &issuerSymbol,
				},
				Val: xdr.ScVal{
					Type: xdr.ScValTypeScvObject,
					Obj:  &issuerObj,
				},
			},
		},
	}
	metadataObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoVec,
		Vec: &xdr.ScVec{
			xdr.ScVal{
				Type: xdr.ScValTypeScvSymbol,
				Sym:  &symbol,
			},
			xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &mapObj,
			},
		},
	}
	return metadataObj, nil
}

// AssetToContractData is the inverse of AssetFromContractData. It creates a ledger entry
// containing the asset metadata written to contract storage by the Stellar Asset Contract.
func AssetToContractData(isNative bool, code, issuer string, contractID [32]byte) (xdr.LedgerEntryData, error) {
	obj, err := metadataObjFromAsset(isNative, code, issuer)
	if err != nil {
		return xdr.LedgerEntryData{}, err
	}
	return xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.ContractDataEntry{
			ContractId: contractID,
			Key:        assetMetadataKey,
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &obj,
			},
		},
	}, nil
}

// BalanceToContractData is the inverse of ContractBalanceFromContractData. It creates a ledger entry
// containing the asset balance of a contract holder written to contract storage by the Stellar Asset Contract.
func BalanceToContractData(assetContractId, holderID [32]byte, amt uint64) xdr.LedgerEntryData {
	return balanceToContractData(assetContractId, holderID, xdr.Int128Parts{
		Lo: xdr.Uint64(amt),
		Hi: 0,
	})
}

func balanceToContractData(assetContractId, holderID [32]byte, amt xdr.Int128Parts) xdr.LedgerEntryData {
	holder := xdr.Hash(holderID)
	addressObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoAddress,
		Address: &xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: &holder,
		},
	}
	keyObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoVec,
		Vec: &xdr.ScVec{
			xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &balanceMetadataSym},
			xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &addressObj,
			},
		},
	}

	amountObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoI128,
		I128: &amt,
	}
	amountSym := xdr.ScSymbol("amount")
	authorizedSym := xdr.ScSymbol("authorized")
	clawbackSym := xdr.ScSymbol("clawback")
	trueIc := xdr.ScStaticScsTrue
	dataObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoMap,
		Map: &xdr.ScMap{
			xdr.ScMapEntry{
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvSymbol,
					Sym:  &amountSym,
				},
				Val: xdr.ScVal{
					Type: xdr.ScValTypeScvObject,
					Obj:  &amountObj,
				},
			},
			xdr.ScMapEntry{
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvSymbol,
					Sym:  &authorizedSym,
				},
				Val: xdr.ScVal{
					Type: xdr.ScValTypeScvStatic,
					Ic:   &trueIc,
				},
			},
			xdr.ScMapEntry{
				Key: xdr.ScVal{
					Type: xdr.ScValTypeScvSymbol,
					Sym:  &clawbackSym,
				},
				Val: xdr.ScVal{
					Type: xdr.ScValTypeScvStatic,
					Ic:   &trueIc,
				},
			},
		},
	}

	return xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.ContractDataEntry{
			ContractId: assetContractId,
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &keyObj,
			},
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &dataObj,
			},
		},
	}
}
