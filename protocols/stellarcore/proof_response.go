package stellarcore

// ProofResponse is the structure of Stellar Core's /getrestorationproof and
// /getinvocationproof
type ProofResponse struct {
	Ledger uint32 `json:"ledger"`
	Proof  string `json:"proof,omitempty"`
}
