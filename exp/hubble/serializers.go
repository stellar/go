package hubble

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

func serializeEntry(entry xdr.LedgerEntryChange, prefix string) string {
	entryString := ""
	entryString += fmt.Sprintf("Type: %s\n", entry.Type)
	entryString += fmt.Sprintf("Created: %s\n", serializeLedgerEntry(entry.Created, prefix))
	entryString += fmt.Sprintf("Updated: %s\n", serializeLedgerEntry(entry.Updated, prefix))
	entryString += fmt.Sprintf("Removed: %v\n", serializeLedgerKey(entry.Removed, prefix))
	entryString += fmt.Sprintf("State: %s\n", serializeLedgerEntry(entry.State, prefix))
	entryString += "\n" // For extra spacing when printing multiple entries.
	return entryString
}

func serializeLedgerEntry(entry *xdr.LedgerEntry, prefix string) string {
	if entry == nil {
		return "<nil>"
	}
	entryString := "\n"
	newPrefix := prefix + prefix
	entryString += fmt.Sprintf("%sLastModifiedLedgerSeq: %d\n", prefix, entry.LastModifiedLedgerSeq)
	entryString += fmt.Sprintf("%sData: %s\n", prefix, serializeLedgerEntryData(entry.Data, newPrefix))
	entryString += fmt.Sprintf("%sExt: %v\n", prefix, entry.Ext)
	return entryString
}

func serializeLedgerKey(key *xdr.LedgerKey, prefix string) string {
	if key == nil {
		return "<nil>"
	}
	keyString := "\n"
	newPrefix := prefix + prefix
	keyString += fmt.Sprintf("%sType: %s\n", prefix, key.Type)
	keyString += fmt.Sprintf("%sAccount: %s\n", prefix, serializeLedgerKeyAccount(key.Account, newPrefix))
	keyString += fmt.Sprintf("%sTrustLine: %s\n", prefix, serializeLedgerKeyTrustLine(key.TrustLine, newPrefix))
	keyString += fmt.Sprintf("%sOffer: %s\n", prefix, serializeLedgerKeyOffer(key.Offer, newPrefix))
	keyString += fmt.Sprintf("%sData: %s", prefix, serializeLedgerKeyData(key.Data, newPrefix))
	return keyString
}

func serializeLedgerKeyAccount(account *xdr.LedgerKeyAccount, prefix string) string {
	if account == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sAccountId: %s", prefix, account.AccountId.Address())
	return keyString
}

func serializeLedgerKeyTrustLine(trustline *xdr.LedgerKeyTrustLine, prefix string) string {
	if trustline == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sAccountId: %s\n", prefix, trustline.AccountId.Address())
	keyString += fmt.Sprintf("%sAsset: %s", prefix, trustline.Asset)
	return keyString
}

func serializeLedgerKeyOffer(offer *xdr.LedgerKeyOffer, prefix string) string {
	if offer == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sSellerId: %s\n", prefix, offer.SellerId.Address())
	keyString += fmt.Sprintf("%sOfferId: %d", prefix, offer.OfferId)
	return keyString
}

func serializeLedgerKeyData(data *xdr.LedgerKeyData, prefix string) string {
	if data == nil {
		return "<nil>"
	}
	keyString := "\n"
	keyString += fmt.Sprintf("%sAccountId: %s\n", prefix, data.AccountId.Address())
	keyString += fmt.Sprintf("%sDataName: %s", prefix, data.DataName)
	return keyString
}

func serializeLedgerEntryData(data xdr.LedgerEntryData, prefix string) string {
	entryString := "\n"
	newPrefix := prefix + prefix
	entryString += fmt.Sprintf("%sType: %s\n", prefix, data.Type)
	entryString += fmt.Sprintf("%sOffer: %s\n", prefix, serializeOfferEntry(data.Offer, newPrefix))
	entryString += fmt.Sprintf("%sData: %s\n", prefix, serializeDataEntry(data.Data, newPrefix))
	entryString += fmt.Sprintf("%sTrustLine: %s\n", prefix, serializeTrustlineEntry(data.TrustLine, newPrefix))
	entryString += fmt.Sprintf("%sAccount: %s", prefix, serializeAccountEntry(data.Account, newPrefix))
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
