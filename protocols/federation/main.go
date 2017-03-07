package federation

import (
	"encoding/json"
)

// NameResponse represents the result of a federation request
// for `name` and `forward` requests.
type NameResponse struct {
	AccountID string `json:"account_id"`
	MemoType  string `json:"memo_type,omitempty"`
	// We are using json.Number to support unmarshalling integer
	// values into `memo` field. This will contain string value of
	// numeric `memo` (123/"123") but also normal string values.
	Memo json.Number `json:"memo,omitempty"`
}

// IDResponse represents the result of a federation request
// for `id` request.
type IDResponse struct {
	Address string `json:"stellar_address"`
}
