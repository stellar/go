package stellarcore

const (
	LedgerEntryStateLive              = "live"
	LedgerEntryStateNewProofless      = "new_entry_no_proof"
	LedgerEntryStateNewProof          = "new_entry_proof"
	LedgerEntryStateArchivedProofless = "archived_no_proof"
	LedgerEntryStateArchivedProof     = "archived_proof"
)

// GetLedgerEntriesResponse is the structure of Stellar Core's /getledgerentry
type GetLedgerEntriesResponse struct {
	Ledger  uint32                     `json:"ledger"`
	Entries []RawLedgerEntriesResponse `json:"entries"`
}

type RawLedgerEntriesResponse struct {
	Entry string `json:"le"`    // base64-encoded xdr.LedgerEntry
	State string `json:"state"` // one of the above states
}
