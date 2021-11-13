package xdr

import (
	"encoding/base64"
	"fmt"
)

// LedgerKey implements the `Keyer` interface
func (key *LedgerKey) LedgerKey() LedgerKey {
	return *key
}

// Equals returns true if `other` is equivalent to `key`
func (key *LedgerKey) Equals(other LedgerKey) bool {
	if key.Type != other.Type {
		return false
	}

	switch key.Type {
	case LedgerEntryTypeAccount:
		l := key.MustAccount()
		r := other.MustAccount()
		return l.AccountId.Equals(r.AccountId)
	case LedgerEntryTypeData:
		l := key.MustData()
		r := other.MustData()
		return l.AccountId.Equals(r.AccountId) && l.DataName == r.DataName
	case LedgerEntryTypeOffer:
		l := key.MustOffer()
		r := other.MustOffer()
		return l.SellerId.Equals(r.SellerId) && l.OfferId == r.OfferId
	case LedgerEntryTypeTrustline:
		l := key.MustTrustLine()
		r := other.MustTrustLine()
		return l.AccountId.Equals(r.AccountId) && l.Asset.Equals(r.Asset)
	case LedgerEntryTypeLiquidityPool:
		l := key.MustLiquidityPool()
		r := other.MustLiquidityPool()
		return l.LiquidityPoolId == r.LiquidityPoolId
	default:
		panic(fmt.Errorf("Unknown ledger key type: %v", key.Type))
	}
}

// SetAccount mutates `key` such that it represents the identity of `account`
func (key *LedgerKey) SetAccount(account AccountId) error {
	data := LedgerKeyAccount{account}
	nkey, err := NewLedgerKey(LedgerEntryTypeAccount, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// SetData mutates `key` such that it represents the identity of the
// data entry owned by `account` and for `name`.
func (key *LedgerKey) SetData(account AccountId, name string) error {
	data := LedgerKeyData{account, String64(name)}
	nkey, err := NewLedgerKey(LedgerEntryTypeData, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// SetOffer mutates `key` such that it represents the identity of the
// data entry owned by `account` and for offer `id`.
func (key *LedgerKey) SetOffer(account AccountId, id uint64) error {
	data := LedgerKeyOffer{account, Int64(id)}
	nkey, err := NewLedgerKey(LedgerEntryTypeOffer, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// SetTrustline mutates `key` such that it represents the identity of the
// trustline owned by `account` and for `asset`.
func (key *LedgerKey) SetTrustline(account AccountId, line TrustLineAsset) error {
	data := LedgerKeyTrustLine{account, line}
	nkey, err := NewLedgerKey(LedgerEntryTypeTrustline, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// SetClaimableBalance mutates `key` such that it represents the identity of a
// claimable balance.
func (key *LedgerKey) SetClaimableBalance(balanceID ClaimableBalanceId) error {
	data := LedgerKeyClaimableBalance{balanceID}
	nkey, err := NewLedgerKey(LedgerEntryTypeClaimableBalance, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// SetLiquidityPool mutates `key` such that it represents the identity of a
// liquidity pool.
func (key *LedgerKey) SetLiquidityPool(poolID PoolId) error {
	data := LedgerKeyLiquidityPool{poolID}
	nkey, err := NewLedgerKey(LedgerEntryTypeLiquidityPool, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// MarshalBinaryCompress marshals LedgerKey to []byte but unlike
// MarshalBinary() it removes all unnecessary bytes, exploting the fact
// that XDR is padding data to 4 bytes in union discriminants etc.
// It's primary use is in ingest/io.StateReader that keep LedgerKeys in
// memory so this function decrease memory requirements.
//
// Warning, do not use UnmarshalBinary() on data encoded using this method!
//
// Optimizations:
// - Writes a single byte for union discriminants vs 4 bytes.
// - Removes type and code padding for Asset.
func (key LedgerKey) MarshalBinaryCompress() ([]byte, error) {
	e := NewEncodingBuffer()
	if err := e.ledgerKeyCompressEncodeTo(key); err != nil {
		return nil, err
	}
	return e.xdrEncoderBuf.Bytes(), nil
}

func (e *EncodingBuffer) ledgerKeyCompressEncodeTo(key LedgerKey) error {
	if err := e.xdrEncoderBuf.WriteByte(byte(key.Type)); err != nil {
		return err
	}

	switch key.Type {
	case LedgerEntryTypeAccount:
		return e.accountIdCompressEncodeTo(key.Account.AccountId)
	case LedgerEntryTypeTrustline:
		if err := e.accountIdCompressEncodeTo(key.TrustLine.AccountId); err != nil {
			return err
		}
		return e.assetTrustlineCompressEncodeTo(key.TrustLine.Asset)
	case LedgerEntryTypeOffer:
		if err := e.accountIdCompressEncodeTo(key.Offer.SellerId); err != nil {
			return err
		}
		return key.Offer.OfferId.EncodeTo(e.encoder)
	case LedgerEntryTypeData:
		if err := e.accountIdCompressEncodeTo(key.Data.AccountId); err != nil {
			return err
		}
		dataName := trimRightZeros(e.scratchBuf)
		_, err := e.xdrEncoderBuf.Write(dataName)
		return err
	case LedgerEntryTypeClaimableBalance:
		return key.ClaimableBalance.BalanceId.EncodeTo(e.encoder)
	case LedgerEntryTypeLiquidityPool:
		// TODO: why are we encoding the full pool id (with padding)
		//       here, in TrustLineAsset we just write the bytes directly?
		return key.LiquidityPool.LiquidityPoolId.EncodeTo(e.encoder)
	default:
		panic("Unknown type")
	}

}

// MarshalBinaryBase64 marshals XDR into a binary form and then encodes it
// using base64.
func (key LedgerKey) MarshalBinaryBase64() (string, error) {
	b, err := key.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
