package history

import (
	"github.com/guregu/null"
	"github.com/stellar/go/xdr"
)

func ledgerEntrySponsorToNullString(entry xdr.LedgerEntry) null.String {
	sponsorshipDescriptor := entry.SponsorshipDescriptor()

	var sponsor null.String
	if sponsorshipDescriptor != nil {
		accountID := xdr.AccountId(*sponsorshipDescriptor)
		sponsor.SetValid(accountID.Address())
	}

	return sponsor
}
