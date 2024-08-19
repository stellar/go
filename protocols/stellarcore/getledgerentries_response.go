package stellarcore

const (
	// LiveState represents the state value returned by stellar-core when a
	// ledger entry is live
	LiveState = "live"

	// DeadState represents the state value returned by stellar-core when a
	// ledger entry is dead
	DeadState = "dead"

	// NewStateNoProof indicates the entry is new and does NOT require a proof
	// of non-existence.
	NewStateNoProof = "new_entry_no_proof"

	// NewStateNeedsProof indicates the entry is new and DOES require a proof of
	// non-existence.
	NewStateNeedsProof = "new_entry_proof"

	// ArchivedStateNoProof indicates the entry is archived and does NOT require
	// a proof of existence.
	ArchivedStateNoProof = "archived_no_proof"

	// ArchivedStateNeedsProof indicates the entry is archived and DOES require
	// a proof of non-existence.
	ArchivedStateNeedsProof = "archived_proof"
)

// GetLedgerEntriesResponse is the response from Stellar Core for the getledgerentries endpoint
type GetLedgerEntriesResponse struct {
	Ledger  uint32                `json:"ledger"`
	Entries []LedgerEntryResponse `json:"entries"`
}

type LedgerEntryResponse struct {
	Entry string `json:"e"`     // base64-encoded xdr.LedgerEntry
	State string `json:"state"` // one of the above State constants
}
