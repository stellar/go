package address

func NewContractAddress(contract string) *Address {
	return &Address{AddressType: &Address_Contract{Contract: contract}}
}

func NewAccountAddress(account string) *Address {
	return &Address{AddressType: &Address_Account{Account: account}}
}

func NewLiquidityPoolAddress(lpHash string) *Address {
	return &Address{AddressType: &Address_LiquidityPool{LiquidityPool: lpHash}}
}

func NewClaimableBalanceAddress(claimableBalanceID string) *Address {
	return &Address{AddressType: &Address_ClaimableBalance{ClaimableBalance: claimableBalanceID}}
}
