package resourceadapter

import (
	"github.com/stellar/go/services/horizon/internal/db2/core"
	. "github.com/stellar/go/protocols/resource"
)

func PopulateAccountFlags(dest *AccountFlags, row core.Account) {
	dest.AuthRequired = row.IsAuthRequired()
	dest.AuthRevocable = row.IsAuthRevocable()
}
