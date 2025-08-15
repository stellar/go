package stellarcore

const (
	// Indicates that the entry is live in the current state
	LedgerEntryStateLive = "live"
	// Indicates that the entry wasn't found (thus proven to be brand new) and
	// will live in the current state if created.
	LedgerEntryStateNotFound = "not-found"
	// Indicates that the entry has been archived to the hot archive due to its
	// TTL expiring
	LedgerEntryStateArchived = "archived"
)

// GetLedgerEntryResponse is the structure of Stellar Core's /getledgerentry
type GetLedgerEntryResponse struct {
	Ledger  uint32                `json:"ledgerSeq"`
	Entries []LedgerEntryResponse `json:"entries"`
}

type LedgerEntryResponse struct {
	State              string `json:"state"`                        // one of the above states
	Entry              string `json:"entry,omitempty"`              // base64-encoded xdr.LedgerEntry, or missing if state == new
	LiveUntilLedgerSeq uint32 `json:"liveUntilLedgerSeq,omitempty"` // optional, for live contract data/code
}
