package serve

type accountResponse struct {
	Address  string `json:"address"`
	Type     string `json:"type"`
	Identity string `json:"identity"`
	Signer   string `json:"signer"`
}
