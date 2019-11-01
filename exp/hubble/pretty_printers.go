package hubble

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

// TODO: Rather than hand serializing every struct, use Go reflection and other type tricks
// to set up custom printing rules for certain types (i.e., AccountId and Sequence).
// Then, we can replace this hand serialization with the JSON package methods.
func prettyPrintEntry(entry xdr.LedgerEntryChange, prefix string) string {
	entryString := ""
	entryString += fmt.Sprintf("Type: %s\n", entry.Type)
	entryString += fmt.Sprintf("Created: %s\n", prettyPrintLedgerEntry(entry.Created, prefix))
	entryString += fmt.Sprintf("Updated: %s\n", prettyPrintLedgerEntry(entry.Updated, prefix))
	entryString += fmt.Sprintf("Removed: %v\n", prettyPrintLedgerKey(entry.Removed, prefix))
	entryString += fmt.Sprintf("State: %s\n", prettyPrintLedgerEntry(entry.State, prefix))
	entryString += "\n" // For extra spacing when printing multiple entries.
	return entryString
}

func prettyPrintLedgerEntry(entry *xdr.LedgerEntry, prefix string) string {
	if entry == nil {
		return "<nil>"
	}
	entryString := "\n"
	newPrefix := prefix + prefix
	entryString += fmt.Sprintf("%sLastModifiedLedgerSeq: %d\n", prefix, entry.LastModifiedLedgerSeq)
	entryString += fmt.Sprintf("%sData: %s\n", prefix, prettyPrintLedgerEntryData(entry.Data, newPrefix))
	entryString += fmt.Sprintf("%sExt: %v", prefix, entry.Ext)
	return entryString
}

func prettyPrintLedgerKey(key *xdr.LedgerKey, prefix string) string {
	if key == nil {
		return "<nil>"
	}
	keyString := "\n"
	newPrefix := prefix + prefix
	keyString += fmt.Sprintf("%sType: %s\n", prefix, key.Type)
	keyString += fmt.Sprintf("%sAccount: %s\n", prefix, prettyPrintLedgerKeyAccount(key.Account, newPrefix))
	keyString += fmt.Sprintf("%sTrustLine: %s\n", prefix, prettyPrintLedgerKeyTrustLine(key.TrustLine, newPrefix))
	keyString += fmt.Sprintf("%sOffer: %s\n", prefix, prettyPrintLedgerKeyOffer(key.Offer, newPrefix))
	keyString += fmt.Sprintf("%sData: %s", prefix, prettyPrintLedgerKeyData(key.Data, newPrefix))
	return keyString
}

func prettyPrintLedgerKeyAccount(account *xdr.LedgerKeyAccount, prefix string) string {
	if account == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sAccountId: %s", prefix, account.AccountId.Address())
	return keyString
}

func prettyPrintLedgerKeyTrustLine(trustline *xdr.LedgerKeyTrustLine, prefix string) string {
	if trustline == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sAccountId: %s\n", prefix, trustline.AccountId.Address())
	keyString += fmt.Sprintf("%sAsset: %s", prefix, trustline.Asset)
	return keyString
}

func prettyPrintLedgerKeyOffer(offer *xdr.LedgerKeyOffer, prefix string) string {
	if offer == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sSellerId: %s\n", prefix, offer.SellerId.Address())
	keyString += fmt.Sprintf("%sOfferId: %d", prefix, offer.OfferId)
	return keyString
}

func prettyPrintLedgerKeyData(data *xdr.LedgerKeyData, prefix string) string {
	if data == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sAccountId: %s\n", prefix, data.AccountId.Address())
	keyString += fmt.Sprintf("%sDataName: %s", prefix, data.DataName)
	return keyString
}

func prettyPrintLedgerEntryData(data xdr.LedgerEntryData, prefix string) string {
	entryString := "\n"
	newPrefix := prefix + prefix
	entryString += fmt.Sprintf("%sType: %s\n", prefix, data.Type)
	entryString += fmt.Sprintf("%sOffer: %s\n", prefix, prettyPrintOfferEntry(data.Offer, newPrefix))
	entryString += fmt.Sprintf("%sData: %s\n", prefix, prettyPrintDataEntry(data.Data, newPrefix))
	entryString += fmt.Sprintf("%sTrustLine: %s\n", prefix, prettyPrintTrustlineEntry(data.TrustLine, newPrefix))
	entryString += fmt.Sprintf("%sAccount: %s", prefix, prettyPrintAccountEntry(data.Account, newPrefix))
	return entryString
}

func prettyPrintOfferEntry(offerEntry *xdr.OfferEntry, prefix string) string {
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

func prettyPrintDataEntry(dataEntry *xdr.DataEntry, prefix string) string {
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

func prettyPrintTrustlineEntry(trustlineEntry *xdr.TrustLineEntry, prefix string) string {
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

func prettyPrintAccountEntry(accountEntry *xdr.AccountEntry, prefix string) string {
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
