package serve

type accountResponse struct {
	Address    string                    `json:"address"`
	Identities []accountResponseIdentity `json:"identities"`
	Signer     string                    `json:"signer"`
}

type accountResponseIdentity struct {
	Role          string `json:"role"`
	Authenticated bool   `json:"authenticated,omitempty"`
}
