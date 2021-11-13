package xdr

import (
	"fmt"
)

// ToAsset converts TrustLineAsset to Asset. Panics on type other than
// AssetTypeAssetTypeNative, AssetTypeAssetTypeCreditAlphanum4 or
// AssetTypeAssetTypeCreditAlphanum12.
func (tla TrustLineAsset) ToAsset() Asset {
	var a Asset

	a.Type = tla.Type

	switch a.Type {
	case AssetTypeAssetTypeNative:
		// Empty branch
	case AssetTypeAssetTypeCreditAlphanum4:
		assetCode4 := *tla.AlphaNum4
		a.AlphaNum4 = &assetCode4
	case AssetTypeAssetTypeCreditAlphanum12:
		assetCode12 := *tla.AlphaNum12
		a.AlphaNum12 = &assetCode12
	default:
		panic(fmt.Errorf("Cannot transform type %v to Asset", a.Type))
	}

	return a
}

// MustExtract behaves as Extract, but panics if an error occurs.
func (a TrustLineAsset) Extract(typ interface{}, code interface{}, issuer interface{}) error {
	return a.ToAsset().Extract(typ, code, issuer)
}

// MustExtract behaves as Extract, but panics if an error occurs.
func (a TrustLineAsset) MustExtract(typ interface{}, code interface{}, issuer interface{}) {
	err := a.ToAsset().Extract(typ, code, issuer)

	if err != nil {
		panic(err)
	}
}

// MarshalBinaryCompress marshals TrustLineAsset to []byte but unlike
// MarshalBinary() it removes all unnecessary bytes, exploting the fact
// that XDR is padding data to 4 bytes in union discriminants etc.
// It's primary use is in ingest/io.StateReader that keep LedgerKeys in
// memory so this function decrease memory requirements.
//
// Warning, do not use UnmarshalBinary() on data encoded using this method!
func (a TrustLineAsset) MarshalBinaryCompress() ([]byte, error) {
	e := NewEncodingBuffer()
	if err := e.assetTrustlineCompressEncodeTo(a); err != nil {
		return nil, err
	}
	return e.xdrEncoderBuf.Bytes(), nil
}

func (e *EncodingBuffer) assetTrustlineCompressEncodeTo(a TrustLineAsset) error {
	switch a.Type {
	case AssetTypeAssetTypeNative,
		AssetTypeAssetTypeCreditAlphanum4,
		AssetTypeAssetTypeCreditAlphanum12:
		return e.assetCompressEncodeTo(a.ToAsset())
	case AssetTypeAssetTypePoolShare:
		if err := e.xdrEncoderBuf.WriteByte(byte(a.Type)); err != nil {
			return err
		}
		_, err := e.xdrEncoderBuf.Write((*a.LiquidityPoolId)[:])
		return err
	default:
		panic(fmt.Errorf("Unknown asset type: %v", a.Type))
	}
}

func (a TrustLineAsset) Equals(other TrustLineAsset) bool {
	if a.Type != other.Type {
		return false
	}
	switch a.Type {
	case AssetTypeAssetTypeNative,
		AssetTypeAssetTypeCreditAlphanum4,
		AssetTypeAssetTypeCreditAlphanum12:
		// Safe because a.Type == other.Type
		return a.ToAsset().Equals(other.ToAsset())
	case AssetTypeAssetTypePoolShare:
		return *a.LiquidityPoolId == *other.LiquidityPoolId
	default:
		panic(fmt.Errorf("Unknown asset type: %v", a.Type))
	}
}
