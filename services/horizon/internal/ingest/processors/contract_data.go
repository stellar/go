package processors

import (
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

var (
	assetMetadataSym = xdr.ScSymbol("Metadata")
	assetMetadataObj = &xdr.ScObject{
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
	if !contractData.Key.Equals(assetMetadataKey) {
		return nil
	}
	if contractData.Val.Type != xdr.ScValTypeScvObject {
		return nil
	}
	obj := contractData.Val.MustObj()
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
	case "Native":
		asset := xdr.MustNewNativeAsset()
		nativeAssetContractID, err := asset.ContractID(passphrase)
		if err != nil {
			return nil
		}
		if contractData.ContractId == nativeAssetContractID {
			return &asset
		}
		return nil
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
	if obj.Type != xdr.ScObjectTypeScoBytes {
		return nil
	}
	var err error
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
