package crypto

import "strings"

func ContextInfo(accountAddress, signingAddress string) []byte {
	parts := []string{
		accountAddress,
		signingAddress,
	}
	return []byte(strings.Join(parts, ","))
}
