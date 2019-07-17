package processors

import (
	"github.com/stellar/go/services/horizon/internal/db2/history"
)

type DatabaseProcessorActionType string

const (
	AccountsForSigner DatabaseProcessorActionType = "AccountsForSigner"
)

// DatabaseProcessor is a processor (both state and ledger) that's responsible
// for persisting ledger data used in horizon-demo in a database. It's possible
// to create multiple procesors of this type but they all should share the same
// Database object to share a common transaction. `Action` defines what each
// processor is responsible for.
type DatabaseProcessor struct {
	HistoryQ *history.Q
	Action   DatabaseProcessorActionType
}
