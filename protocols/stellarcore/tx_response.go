package stellarcore

// TxResponse represents the response returned from a submission request sent to stellar-core's /tx
// endpoint
type TxResponse struct {
	Exception string `json:"exception"`
	Error     string `json:"error"`
	Status    string `json:"status"`
}
