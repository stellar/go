// +build go1.13

package hubble

type accountState struct {
	address    string
	seqnum     uint32
	balance    uint32
	signers    []signer
	trustlines map[string]trustline
	offers     map[uint32]offer
	data       map[string][]byte
	// TODO: May want to track other fields in AccountEntry.
}

type signer struct {
	address string
	weight  uint32
}

type trustline struct {
	asset      string
	balance    uint32
	limit      uint32
	authorized bool
	// TODO: Add liabilities.
}

// TODO: Save amount as a decimal instead of integer.
// TODO: Save decimal in addition to N and D.
type offer struct {
	id         uint32
	seller     string // seller address
	selling    string // selling asset
	buying     string // buying asset
	amount     uint32
	priceNum   uint16
	priceDenom uint16
	// TODO: Add flags.
}
