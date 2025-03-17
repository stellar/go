package address

import (
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

func (a *Address) Equals(other *Address) bool {
	if a == nil || other == nil {
		return a == other
	}
	return a.Strkey == other.Strkey
}

func NewAddressFromAccount(account xdr.MuxedAccount) *Address {
	return &Address{
		// Always convert the address to a G address, since, in CAP-67, - addresses will not appear in output event
		Strkey: account.ToAccountId().Address(),
	}
}

func NewAddressFromContract(contractId xdr.Hash) *Address {
	return &Address{
		Strkey: strkey.MustEncode(strkey.VersionByteContract, contractId[:]),
	}
}

func NewAddressFromClaimableBalance(cbId xdr.ClaimableBalanceId) *Address {
	return &Address{
		Strkey: cbId.MustV0().HexString(),
	}
}

// Helper function to create an Address for a LiquidityPool
func NewLiquidityPoolAddress(lpId xdr.PoolId) *Address {
	return &Address{
		Strkey: xdr.Hash(lpId).HexString(),
	}
}
