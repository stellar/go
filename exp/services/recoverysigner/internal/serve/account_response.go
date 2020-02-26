package serve

type accountResponse struct {
	Address    string                    `json:"address"`
	Identities accountResponseIdentities `json:"identities"`
	Identity   string                    `json:"identity"`
	Signer     string                    `json:"signer"`
}

type accountResponseIdentities struct {
	Owner accountResponseIdentity `json:"owner"`
	Other accountResponseIdentity `json:"other"`
}

type accountResponseIdentity struct {
	Present bool `json:"present"`
}
