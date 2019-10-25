package hubble

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

func serializeEntry(entry xdr.LedgerEntryChange) string {
	prefix := "\t"
	entryString := ""
	entryString += fmt.Sprintf("Type: %s\n", entry.Type)
	entryString += fmt.Sprintf("Created: %s\n", serializeLedgerEntry(entry.Created, prefix))
	entryString += fmt.Sprintf("Updated: %s\n", serializeLedgerEntry(entry.Updated, prefix))
	entryString += fmt.Sprintf("Removed: %v\n", entry.Removed)
	entryString += fmt.Sprintf("State: %s\n", serializeLedgerEntry(entry.State, prefix))
	return entryString
}

func serializeLedgerEntry(entry *xdr.LedgerEntry, prefix string) string {
	if entry == nil {
		return "<nil>"
	}
	entryString := "\n"
	newPrefix := prefix + "\t"
	entryString += fmt.Sprintf("%sLastModifiedLedgerSeq: %d\n", prefix, entry.LastModifiedLedgerSeq)
	entryString += fmt.Sprintf("%sData: %s", prefix, serializeEntryData(entry.Data, newPrefix))
	entryString += fmt.Sprintf("%sExt: %v\n", prefix, entry.Ext)
	return entryString
}

func serializeEntryData(data xdr.LedgerEntryData, prefix string) string {
	entryString := "\n"
	newPrefix := prefix + "\t"
	entryString += fmt.Sprintf("%sType: %s\n", prefix, data.Type)
	entryString += fmt.Sprintf("%sOffer: %s\n", prefix, serializeOfferEntry(data.Offer, newPrefix))
	entryString += fmt.Sprintf("%sData: %s\n", prefix, serializeDataEntry(data.Data, newPrefix))
	entryString += fmt.Sprintf("%sTrustLine: %s\n", prefix, serializeTrustlineEntry(data.TrustLine, newPrefix))
	entryString += fmt.Sprintf("%sAccount: %s\n", prefix, serializeAccountEntry(data.Account, newPrefix))

	return entryString
}

func serializeOfferEntry(offerEntry *xdr.OfferEntry, prefix string) string {
	if offerEntry == nil {
		return "<nil>"
	}
	entryString := "\n"
	entryString += fmt.Sprintf("%sSellerId: %s\n", prefix, offerEntry.SellerId.Address())
	entryString += fmt.Sprintf("%sOfferId: %d\n", prefix, offerEntry.OfferId)
	entryString += fmt.Sprintf("%sSelling: %s\n", prefix, offerEntry.Selling.String())
	entryString += fmt.Sprintf("%sBuying: %s\n", prefix, offerEntry.Buying.String())
	entryString += fmt.Sprintf("%sAmount: %d\n", prefix, offerEntry.Amount)
	// TODO: Price may need custom rule to clarify numerator and denominator.
	entryString += fmt.Sprintf("%sPrice: %v\n", prefix, offerEntry.Price)
	entryString += fmt.Sprintf("%sFlags: %d\n", prefix, offerEntry.Flags)
	entryString += fmt.Sprintf("%sExt: %v", prefix, offerEntry.Ext)
	return entryString
}

func serializeDataEntry(dataEntry *xdr.DataEntry, prefix string) string {
	if dataEntry == nil {
		return "<nil>"
	}
	entryString := "\n"
	entryString += fmt.Sprintf("%sAccountId: %s\n", prefix, dataEntry.AccountId.Address())
	entryString += fmt.Sprintf("%sDataName: %s\n", prefix, dataEntry.DataName)
	entryString += fmt.Sprintf("%sDataValue: %x\n", prefix, dataEntry.DataValue)
	entryString += fmt.Sprintf("%sExt: %v", prefix, dataEntry.Ext)
	return entryString
}

func serializeTrustlineEntry(trustlineEntry *xdr.TrustLineEntry, prefix string) string {
	if trustlineEntry == nil {
		return "<nil>"
	}
	entryString := "\n"
	entryString += fmt.Sprintf("%sAccountId: %s\n", prefix, trustlineEntry.AccountId.Address())
	entryString += fmt.Sprintf("%sAsset: %s\n", prefix, trustlineEntry.Asset.String())
	entryString += fmt.Sprintf("%sBalance: %d\n", prefix, trustlineEntry.Balance)
	entryString += fmt.Sprintf("%sLimit: %d\n", prefix, trustlineEntry.Limit)
	entryString += fmt.Sprintf("%sFlags: %d\n", prefix, trustlineEntry.Flags)
	entryString += fmt.Sprintf("%sExt: %v", prefix, trustlineEntry.Ext)
	return entryString
}

func serializeAccountEntry(accountEntry *xdr.AccountEntry, prefix string) string {
	if accountEntry == nil {
		return "<nil>"
	}
	entryString := "\n"
	entryString += fmt.Sprintf("%sAccountId: %s\n", prefix, accountEntry.AccountId.Address())
	entryString += fmt.Sprintf("%sBalance: %d\n", prefix, accountEntry.Balance)
	entryString += fmt.Sprintf("%sSeqNum: %d\n", prefix, accountEntry.SeqNum)
	entryString += fmt.Sprintf("%sNumSubEntries: %d\n", prefix, accountEntry.NumSubEntries)
	entryString += fmt.Sprintf("%sInflationDest: %s\n", prefix, accountEntry.InflationDest.Address())
	entryString += fmt.Sprintf("%sFlags: %d\n", prefix, accountEntry.Flags)
	entryString += fmt.Sprintf("%sHomeDomain: %s\n", prefix, accountEntry.HomeDomain)
	entryString += fmt.Sprintf("%sThresholds: %v\n", prefix, accountEntry.Thresholds)
	entryString += fmt.Sprintf("%sSigners: %v\n", prefix, accountEntry.Signers)
	entryString += fmt.Sprintf("%sExt: %v", prefix, accountEntry.Ext)
	return entryString
}
