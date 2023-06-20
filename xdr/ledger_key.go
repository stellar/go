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
	case LedgerEntryTypeConfigSetting:
		l := key.MustConfigSetting()
		r := other.MustConfigSetting()
		return l.ConfigSettingId == r.ConfigSettingId
	case LedgerEntryTypeContractData:
		l := key.MustContractData()
		r := other.MustContractData()
		return l.Contract.Equals(r.Contract) && l.Key.Equals(r.Key) && l.LeType == r.LeType && l.Type == r.Type
	case LedgerEntryTypeContractCode:
		l := key.MustContractCode()
		r := other.MustContractCode()
		return l.Hash == r.Hash
	case LedgerEntryTypeClaimableBalance:
		l := key.MustClaimableBalance()
		r := other.MustClaimableBalance()
		return l.BalanceId.MustV0() == r.BalanceId.MustV0()
	default:
		panic(fmt.Errorf("unknown ledger key type: %v", key.Type))
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

// SetContractData mutates `key` such that it represents the identity of a
// contract data entry.
func (key *LedgerKey) SetContractData(contract ScAddress, keyVal ScVal, keyType ContractDataType, keyLeType ContractLedgerEntryType) error {
	data := LedgerKeyContractData{
		Contract: contract,
		Key:      keyVal,
		Type:     keyType,
		LeType:   keyLeType,
	}
	nkey, err := NewLedgerKey(LedgerEntryTypeContractData, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// SetContractCode mutates `key` such that it represents the identity of a
// contract code entry.
func (key *LedgerKey) SetContractCode(contractID Hash) error {
	data := LedgerKeyContractCode{
		Hash: contractID,
	}
	nkey, err := NewLedgerKey(LedgerEntryTypeContractCode, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// SetConfigSetting mutates `key` such that it represents the identity of a
// config setting entry.
func (key *LedgerKey) SetConfigSetting(configSettingID ConfigSettingId) error {
	data := LedgerKeyConfigSetting{
		ConfigSettingId: configSettingID,
	}
	nkey, err := NewLedgerKey(LedgerEntryTypeConfigSetting, data)
	if err != nil {
		return err
	}

	*key = nkey
	return nil
}

// GetLedgerKeyFromData obtains a ledger key from LedgerEntryData
//
//nolint:gocyclo
func GetLedgerKeyFromData(data LedgerEntryData) (LedgerKey, error) {
	var key LedgerKey
	switch data.Type {
	case LedgerEntryTypeAccount:
		if err := key.SetAccount(data.Account.AccountId); err != nil {
			return key, err
		}
	case LedgerEntryTypeTrustline:
		if err := key.SetTrustline(data.TrustLine.AccountId, data.TrustLine.Asset); err != nil {
			return key, err
		}
	case LedgerEntryTypeContractData:
		if err := key.SetContractData(
			data.ContractData.Contract,
			data.ContractData.Key,
			data.ContractData.Type,
			data.ContractData.Body.LeType); err != nil {
			return key, err
		}
	case LedgerEntryTypeContractCode:
		if err := key.SetContractCode(data.ContractCode.Hash); err != nil {
			return key, err
		}
	case LedgerEntryTypeData:
		if err := key.SetData(data.Data.AccountId, string(data.Data.DataName)); err != nil {
			return key, err
		}
	case LedgerEntryTypeOffer:
		if err := key.SetOffer(data.Offer.SellerId, uint64(data.Offer.OfferId)); err != nil {
			return key, err
		}
	case LedgerEntryTypeLiquidityPool:
		if err := key.SetLiquidityPool(data.LiquidityPool.LiquidityPoolId); err != nil {
			return key, err
		}
	case LedgerEntryTypeClaimableBalance:
		if err := key.SetClaimableBalance(data.ClaimableBalance.BalanceId); err != nil {
			return key, err
		}
	case LedgerEntryTypeConfigSetting:
		if err := key.SetConfigSetting(data.ConfigSetting.ConfigSettingId); err != nil {
			return key, err
		}
	default:
		return key, fmt.Errorf("unknown ledger entry type %d", data.Type)
	}

	return key, nil
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
		// We intentionally don't encode the SellerID since the OfferID is enough
		// (it's unique to the network)
		return key.Offer.OfferId.EncodeTo(e.encoder)
	case LedgerEntryTypeData:
		if err := e.accountIdCompressEncodeTo(key.Data.AccountId); err != nil {
			return err
		}
		dataName := trimRightZeros([]byte(key.Data.DataName))
		_, err := e.xdrEncoderBuf.Write(dataName)
		return err
	case LedgerEntryTypeClaimableBalance:
		return e.claimableBalanceCompressEncodeTo(key.ClaimableBalance.BalanceId)
	case LedgerEntryTypeLiquidityPool:
		_, err := e.xdrEncoderBuf.Write(key.LiquidityPool.LiquidityPoolId[:])
		return err
	case LedgerEntryTypeContractData:
		// contract
		if contractBytes, err := key.ContractData.Contract.MarshalBinary(); err != nil {
			return err
		} else {
			if _, err := e.xdrEncoderBuf.Write(contractBytes[:]); err != nil {
				return err
			}
		}
		// key
		if err := key.ContractData.Key.EncodeTo(e.encoder); err != nil {
			return err
		}
		// type
		if err := e.xdrEncoderBuf.WriteByte(byte(key.ContractData.Type)); err != nil {
			return err
		}
		// letype
		return e.xdrEncoderBuf.WriteByte(byte(key.ContractData.LeType))
	case LedgerEntryTypeContractCode:
		_, err := e.xdrEncoderBuf.Write(key.ContractCode.Hash[:])
		return err
	case LedgerEntryTypeConfigSetting:
		return key.ConfigSetting.ConfigSettingId.EncodeTo(e.encoder)
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
