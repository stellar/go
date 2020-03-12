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
	AuthMethodTypeAccount     AuthMethodType = "stellar_account"
	AuthMethodTypePhoneNumber AuthMethodType = "phone_number"
	AuthMethodTypeEmail       AuthMethodType = "email"
)

type AuthMethod struct {
	Type  AuthMethodType
	Value string
}
