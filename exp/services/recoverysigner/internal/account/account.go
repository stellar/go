package account

import "github.com/stellar/go/support/errors"

type Account struct {
	Address    string
	Identities []Identity
}

type Identity struct {
	Role        string
	AuthMethods []AuthMethod
}

type AuthMethodType string

const (
	AuthMethodTypeAddress     AuthMethodType = "stellar_address"
	AuthMethodTypePhoneNumber AuthMethodType = "phone_number"
	AuthMethodTypeEmail       AuthMethodType = "email"
)

func AuthMethodTypeFromString(s string) (AuthMethodType, error) {
	if AuthMethodTypes[AuthMethodType(s)] {
		return AuthMethodType(s), nil
	}
	return AuthMethodType(""), errors.Errorf("auth method type %q unrecognized", s)
}

var AuthMethodTypes = map[AuthMethodType]bool{
	AuthMethodTypeAddress:     true,
	AuthMethodTypePhoneNumber: true,
	AuthMethodTypeEmail:       true,
}

type AuthMethod struct {
	Type  AuthMethodType
	Value string
}
