package account

type Account struct {
	Address         string
	OwnerIdentities Identities
	OtherIdentities Identities
}

type Identities struct {
	Address     string
	PhoneNumber string
	Email       string
}

// Present indicates if the Identities contains at least one identity. Returns
// false if it is an empty/zero value.
func (i Identities) Present() bool {
	return i != Identities{}
}
