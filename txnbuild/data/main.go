package data

import "github.com/stellar/go/txnbuild"

// NewManageDataMemoRequired returns a valid txnbuild.ManageData operation setting memo required to 1 or 0.
func NewManageDataMemoRequired(b bool) *txnbuild.ManageData {
	value := "0"
	if b {
		value = "1"
	}
	return &txnbuild.ManageData{
		Name:  "config.memo_required",
		Value: []byte(value),
	}
}
