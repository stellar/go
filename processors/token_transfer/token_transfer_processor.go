package token_transfer

import (
	"github.com/stellar/go/protos/processors/token_transfer"
	"github.com/stellar/go/xdr"
)

func ProcessTokenTransferEvents(ledger xdr.LedgerCloseMeta) ([]token_transfer.TokenTransferEvent, error) {
	return nil, nil
}
