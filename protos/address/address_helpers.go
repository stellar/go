package address

import (
	"fmt"
)

// String returns a human-readable representation of the Address.
func (a *Address) Strings() string {
	switch addr := a.AddressType.(type) {
	case *Address_SmartContractAddress:
		return fmt.Sprintf("Smart Contract Address: %s", addr.SmartContractAddress)
	case *Address_AccountAddress:
		return fmt.Sprintf("Account Address: %s", addr.AccountAddress)
	case *Address_LiquidityPoolHash:
		return fmt.Sprintf("Liquidity Pool Hash: %s", addr.LiquidityPoolHash)
	case *Address_ClaimableBalanceId:
		return fmt.Sprintf("Claimable Balance ID: %s", addr.ClaimableBalanceId)
	default:
		return "Unknown Address Type"
	}
}
