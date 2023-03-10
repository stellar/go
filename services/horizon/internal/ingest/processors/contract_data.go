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
	if ledgerEntry.Data.Type != xdr.LedgerEntryTypeContractData {
		return nil
	}

	contractData := ledgerEntry.Data.MustContractData()
	// we don't support asset stats for lumens
	nativeAssetContractID, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	if err != nil || contractData.ContractId == nativeAssetContractID {
		return nil
	}

	if !contractData.Key.Equals(assetMetadataKey) {
		return nil
	}
	if contractData.Val.Type != xdr.ScValTypeScvObject {
		return nil
	}
	obj := contractData.Val.MustObj()
	if obj == nil {
		return nil
	}
	if obj.Type != xdr.ScObjectTypeScoVec {
		return nil
	}
	vec := obj.MustVec()
	if len(vec) <= 0 {
		return nil
	}

	if vec[0].Type != xdr.ScValTypeScvSymbol {
		return nil
	}
	switch vec[0].MustSym() {
	case "AlphaNum4":
	case "AlphaNum12":
	default:
		return nil
	}
	if len(vec) != 2 {
		return nil
	}

	var assetCode, assetIssuer string
	if vec[1].Type != xdr.ScValTypeScvObject {
		return nil
	}
	obj = vec[1].MustObj()
	if obj == nil {
		return nil
	}
	if obj.Type != xdr.ScObjectTypeScoMap {
		return nil
	}
	assetMap := obj.MustMap()
	if len(assetMap) != 2 {
		return nil
	}
	assetCodeEntry, assetIssuerEntry := assetMap[0], assetMap[1]
	if assetCodeEntry.Key.Type != xdr.ScValTypeScvSymbol {
		return nil
	}
	if assetCodeEntry.Key.MustSym() != "asset_code" {
		return nil
	}
	if assetCodeEntry.Val.Type != xdr.ScValTypeScvObject {
		return nil
	}
	obj = assetCodeEntry.Val.MustObj()
	if obj == nil {
		return nil
	}
	if obj.Type != xdr.ScObjectTypeScoBytes {
		return nil
	}
	assetCode = string(obj.MustBin())

	if assetIssuerEntry.Key.Type != xdr.ScValTypeScvSymbol {
		return nil
	}
	if assetIssuerEntry.Key.MustSym() != "issuer" {
		return nil
	}
	if assetIssuerEntry.Val.Type != xdr.ScValTypeScvObject {
		return nil
	}
	obj = assetIssuerEntry.Val.MustObj()
	if obj == nil {
		return nil
	}
	if obj.Type != xdr.ScObjectTypeScoBytes {
		return nil
	}
	assetIssuer, err = strkey.Encode(strkey.VersionByteAccountID, obj.MustBin())
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

func ContractBalanceFromContractData(ledgerEntry xdr.LedgerEntry, passphrase string) ([32]byte, *big.Int, bool) {
	if ledgerEntry.Data.Type != xdr.LedgerEntryTypeContractData {
		return [32]byte{}, nil, false
	}

	contractData := ledgerEntry.Data.MustContractData()
	// we don't support asset stats for lumens
	nativeAssetContractID, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	if err != nil || contractData.ContractId == nativeAssetContractID {
		return [32]byte{}, nil, false
	}

	if contractData.Key.Type != xdr.ScValTypeScvObject {
		return [32]byte{}, nil, false
	}
	keyObj := contractData.Key.MustObj()
	if keyObj == nil {
		return [32]byte{}, nil, false
	}
	if keyObj.Type != xdr.ScObjectTypeScoVec {
		return [32]byte{}, nil, false
	}
	keyEnumVec := keyObj.MustVec()
	if len(keyEnumVec) != 2 || !keyEnumVec[0].Equals(xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &balanceMetadataSym}) {
		return [32]byte{}, nil, false
	}
	if keyEnumVec[1].Type != xdr.ScValTypeScvObject {
		return [32]byte{}, nil, false
	}
	addressObj := keyEnumVec[1].MustObj()
	if addressObj == nil {
		return [32]byte{}, nil, false
	}
	if addressObj.Type != xdr.ScObjectTypeScoAddress {
		return [32]byte{}, nil, false
	}
	scAddress := addressObj.MustAddress()
	if scAddress.Type != xdr.ScAddressTypeScAddressTypeContract {
		return [32]byte{}, nil, false
	}
	holder := scAddress.MustContractId()

	if contractData.Val.Type != xdr.ScValTypeScvObject {
		return [32]byte{}, nil, false
	}
	obj := contractData.Val.MustObj()
	if obj == nil {
		return [32]byte{}, nil, false
	}
	if obj.Type != xdr.ScObjectTypeScoMap {
		return [32]byte{}, nil, false
	}
	balanceMap := obj.MustMap()
	if len(balanceMap) != 3 {
		return [32]byte{}, nil, false
	}

	if balanceMap[0].Key.Type != xdr.ScValTypeScvSymbol ||
		balanceMap[0].Key.MustSym() != "amount" ||
		balanceMap[0].Val.Type != xdr.ScValTypeScvObject {
		return [32]byte{}, nil, false
	}
	if balanceMap[1].Key.Type != xdr.ScValTypeScvSymbol ||
		balanceMap[1].Key.MustSym() != "authorized" ||
		!isBool(balanceMap[1].Val) {
		return [32]byte{}, nil, false
	}
	if balanceMap[2].Key.Type != xdr.ScValTypeScvSymbol ||
		balanceMap[2].Key.MustSym() != "clawback" ||
		!isBool(balanceMap[1].Val) {
		return [32]byte{}, nil, false
	}
	amountObj := balanceMap[0].Val.MustObj()
	if amountObj == nil {
		return [32]byte{}, nil, false
	}
	if amountObj.Type != xdr.ScObjectTypeScoI128 {
		return [32]byte{}, nil, false
	}
	amount := amountObj.MustI128()
	// check if negative
	if ((amount.Hi >> 56) & 0x80) > 0 {
		return [32]byte{}, nil, false
	}
	amt := new(big.Int).Lsh(big.NewInt(int64(amount.Hi)), 56)
	amt = new(big.Int).Add(amt, big.NewInt(int64(amount.Lo)))
	return holder, amt, true
}

func isBool(v xdr.ScVal) bool {
	if v.Type != xdr.ScValTypeScvStatic {
		return false
	}
	ic := v.MustIc()
	return ic == xdr.ScStaticScsTrue || ic == xdr.ScStaticScsFalse
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
		I128: &xdr.Int128Parts{
			Lo: xdr.Uint64(amt),
			Hi: 0,
		},
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
