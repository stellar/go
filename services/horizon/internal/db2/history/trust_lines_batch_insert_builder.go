package history

import (
	"github.com/stellar/go/xdr"
)

// Add adds a new trust line entry to the batch. `lastModifiedLedger` is another
// parameter because `xdr.TrustLineEntry` does not have a field to hold this value.
func (i *trustLinesBatchInsertBuilder) Add(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) error {
	m := trustLineToMap(trustLine, lastModifiedLedger)
	return i.builder.Row(m)
}

func (i *trustLinesBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
