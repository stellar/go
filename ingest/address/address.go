package address

func NewAccountAddress(addr string) *Address {
	return &Address{AddressType: AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: addr}
}
func NewContractAddress(addr string) *Address {
	return &Address{AddressType: AddressType_ADDRESS_TYPE_CONTRACT, StrKey: addr}
}

func NewLiquidityPoolAddress(addr string) *Address {
	return &Address{AddressType: AddressType_ADDRESS_TYPE_LIQUIDITY_POOL, StrKey: addr}
}

func NewClaimableBalanceAddress(addr string) *Address {
	return &Address{AddressType: AddressType_ADDRESS_TYPE_CLAIMABLE_BALANCE, StrKey: addr}
}

func NewMuxedAccountAddress(addr string) *Address {
	return &Address{AddressType: AddressType_ADDRESS_TYPE_MUXED_ACCOUNT, StrKey: addr}
}
