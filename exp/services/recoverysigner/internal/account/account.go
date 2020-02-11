package account

type Account struct {
	Address         string
	Type            string
	OwnerIdentities Identities
	OtherIdentities Identities
}

type Identities struct {
	Address     string
	PhoneNumber string
	Email       string
}
