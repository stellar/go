package processors

import (
	"crypto/sha256"
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

func contractIDForAsset(isNative bool, code, issuer, passphrase string) ([32]byte, xdr.Asset, error) {
	var asset xdr.Asset
	if isNative {
		asset = xdr.MustNewNativeAsset()
	} else {
		var err error
		asset, err = xdr.NewCreditAsset(code, issuer)
		if err != nil {
			return [32]byte{}, asset, err
		}
	}
	networkId := xdr.Hash(sha256.Sum256([]byte(passphrase)))
	preImage := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypeContractIdFromAsset,
		FromAsset: &xdr.HashIdPreimageFromAsset{
			NetworkId: networkId,
			Asset:     asset,
		},
	}
	xdrPreImageBytes, err := preImage.MarshalBinary()
	if err != nil {
		return [32]byte{}, asset, err
	}
	return sha256.Sum256(xdrPreImageBytes), asset, nil
}

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
		nativeAssetContractID, asset, err := contractIDForAsset(true, "", "", passphrase)
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

	expectedID, asset, err := contractIDForAsset(false, assetCode, assetIssuer, passphrase)
	if err != nil {
		return nil
	}
	if expectedID != contractData.ContractId {
		return nil
	}

	return &asset
}

func padTrailingZeros(buf []byte, length int) []byte {
	for i := len(buf); i < length; i++ {
		buf = append(buf, 0)
	}
	return buf
}

func metadataObjFromAsset(isNative bool, code, issuer string) (*xdr.ScObject, error) {
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
	if isNative {
		return metadataObj, nil
	}

	var assetCodeLength int
	if len(code) <= 4 {
		symbol = "AlphaNum4"
		assetCodeLength = 4
	} else {
		symbol = "AlphaNum12"
		assetCodeLength = 12
	}
	assetCodeBytes := padTrailingZeros([]byte(code), assetCodeLength)
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
	metadataObj = &xdr.ScObject{
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
