package address

import "github.com/stellar/go/xdr"

func AddressFromAccount(account xdr.MuxedAccount) *Address {
	address := &Address{}
	switch account.Type {
	case xdr.CryptoKeyTypeKeyTypeEd25519:
		address.AddressType = AddressType_ADDRESS_TYPE_ACCOUNT
	case xdr.CryptoKeyTypeKeyTypeMuxedEd25519:
		address.AddressType = AddressType_ADDRESS_TYPE_MUXED_ACCOUNT
	}
	address.StrKey = account.Address()
	return address
}

func AddressFromAccountId(account xdr.AccountId) *Address {
	return &Address{
		AddressType: AddressType_ADDRESS_TYPE_ACCOUNT,
		StrKey:      account.Address(),
	}
}
