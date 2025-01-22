package processors

import (
	"fmt"
	"sort"

	"github.com/guregu/null"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

// TransformAccountSigners converts account signers from the history archive ingestion system into a form suitable for BigQuery
func TransformAccountSigners(ledgerChange ingest.Change, header xdr.LedgerHeaderHistoryEntry) ([]AccountSignerOutput, error) {
	var signers []AccountSignerOutput

	ledgerEntry, changeType, outputDeleted, err := ExtractEntryFromChange(ledgerChange)
	if err != nil {
		return signers, err
	}
	outputLastModifiedLedger := uint32(ledgerEntry.LastModifiedLedgerSeq)
	accountEntry, accountFound := ledgerEntry.Data.GetAccount()
	if !accountFound {
		return signers, fmt.Errorf("could not extract signer data from ledger entry of type: %+v", ledgerEntry.Data.Type)
	}

	closedAt, err := TimePointToUTCTimeStamp(header.Header.ScpValue.CloseTime)
	if err != nil {
		return signers, err
	}

	ledgerSequence := header.Header.LedgerSeq

	sponsors := accountEntry.SponsorPerSigner()
	for signer, weight := range accountEntry.SignerSummary() {
		var sponsor null.String
		if sponsorDesc, isSponsored := sponsors[signer]; isSponsored {
			sponsor = null.StringFrom(sponsorDesc.Address())
		}

		signers = append(signers, AccountSignerOutput{
			AccountID:          accountEntry.AccountId.Address(),
			Signer:             signer,
			Weight:             weight,
			Sponsor:            sponsor,
			LastModifiedLedger: outputLastModifiedLedger,
			LedgerEntryChange:  uint32(changeType),
			Deleted:            outputDeleted,
			ClosedAt:           closedAt,
			LedgerSequence:     uint32(ledgerSequence),
		})
	}
	sort.Slice(signers, func(a, b int) bool { return signers[a].Weight < signers[b].Weight })
	return signers, nil
}
