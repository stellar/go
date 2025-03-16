package address

import (
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (a *Address) ToStrkey() string {
	if a == nil {
		panic(errors.New("nil address"))
	}

	switch a.AddressType.(type) {
	case *Address_Account:
		return a.GetAccount().StrKey

	case *Address_Contract:
		return a.GetContract().StrKey

	case *Address_ClaimableBalance:
		return a.GetClaimableBalance().StrKey

	case *Address_LiquidityPool:
		return a.GetLiquidityPool().StrKey

	case *Address_MuxedAccount:
		return a.GetMuxedAccount().StrKey
	}
	panic(errors.Errorf("invalid address type: %v", a.AddressType))
}

func (a *Address) Equals(other *Address) bool {
	if a == nil || other == nil {
		return a == other
	}

	// Compare the types of the oneof fields.
	switch a.AddressType.(type) {
	case *Address_Account:
		_, ok := other.AddressType.(*Address_Account)
		if !ok {
			return false
		}
		return a.GetAccount().StrKey == other.GetAccount().StrKey

	case *Address_Contract:
		_, ok := other.AddressType.(*Address_Contract)
		if !ok {
			return false
		}
		return a.GetContract().StrKey == other.GetContract().StrKey

	case *Address_ClaimableBalance:
		_, ok := other.AddressType.(*Address_ClaimableBalance)
		if !ok {
			return false
		}
		return a.GetClaimableBalance().StrKey == other.GetClaimableBalance().StrKey

	case *Address_LiquidityPool:
		_, ok := other.AddressType.(*Address_LiquidityPool)
		if !ok {
			return false
		}
		return a.GetLiquidityPool().StrKey == other.GetLiquidityPool().StrKey

	case *Address_MuxedAccount:
		_, ok := other.AddressType.(*Address_MuxedAccount)
		if !ok {
			return false
		}
		return a.GetMuxedAccount().MemoId == other.GetMuxedAccount().MemoId &&
			a.GetMuxedAccount().StrKey == other.GetMuxedAccount().StrKey &&
			a.GetMuxedAccount().BaseAccount.StrKey == other.GetMuxedAccount().BaseAccount.StrKey
	}

	// If we fall through, it means the types didn't match.
	return false

}

func NewAddressFromAccount(account xdr.MuxedAccount) *Address {
	var addr *Address
	switch account.Type {
	case xdr.CryptoKeyTypeKeyTypeEd25519:
		addr = &Address{
			AddressType: &Address_Account{
				Account: &Account{
					StrKey: account.Address(),
				},
			},
		}
	case xdr.CryptoKeyTypeKeyTypeMuxedEd25519:
		memoId, err := account.GetId()
		if err != nil {
			panic(errors.Wrapf(err, "Unable to generate address from muxed account"))
		}
		addr = &Address{
			AddressType: &Address_MuxedAccount{
				MuxedAccount: &Muxed_Account{
					MemoId: memoId,
					StrKey: account.Address(),
					BaseAccount: &Account{
						StrKey: account.ToAccountId().Address(),
					},
				},
			},
		}
	}
	return addr
}

func NewAddressFromContract(contractId xdr.Hash) *Address {
	return &Address{
		AddressType: &Address_Contract{
			Contract: &Contract{
				StrKey: strkey.MustEncode(strkey.VersionByteContract, contractId[:]),
			},
		},
	}
}

func NewAddressFromClaimableBalance(cbId xdr.ClaimableBalanceId) *Address {
	return &Address{
		AddressType: &Address_ClaimableBalance{
			ClaimableBalance: &ClaimableBalance{
				//TODO - convert to strkey
				StrKey: cbId.V0.HexString(),
			},
		},
	}
}

// Helper function to create an Address for a LiquidityPool
func NewLiquidityPoolAddress(lpId xdr.PoolId) *Address {
	return &Address{
		AddressType: &Address_LiquidityPool{
			//TODO - convert to strkey
			LiquidityPool: &LiquidityPool{
				StrKey: xdr.Hash(lpId).HexString(),
			},
		},
	}
}
