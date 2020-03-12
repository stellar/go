package account

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

type AuthMethod struct {
	Type  AuthMethodType
	Value string
}
