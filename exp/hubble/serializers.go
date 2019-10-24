package hubble

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

func serializeEntry(entry xdr.LedgerEntryChange) error {
	switch entryType := entry.EntryType().String(); entryType {
	case "LedgerEntryTypeOffer":
		return serializeOfferEntry(entry.State.Data.Offer)
	case "LedgerEntryTypeData":
		return serializeDataEntry(entry.State.Data.Data)
	case "LedgerEntryTypeTrustline":
		return serializeTrustlineEntry(entry.State.Data.TrustLine)
	case "LedgerEntryTypeAccount":
		return serializeAccountEntry(entry.State.Data.Account)
	default:
		fmt.Println("Found none of the above!")
	}
	return nil
}

func serializeOfferEntry(offerEntry *xdr.OfferEntry) error {
	fmt.Println("Found offer!")
	fmt.Printf("SellerId: %s\n", offerEntry.SellerId.Address())
	fmt.Printf("OfferId: %d\n", offerEntry.OfferId)
	fmt.Printf("Selling: %s\n", offerEntry.Selling.String())
	fmt.Printf("Buying: %s\n", offerEntry.Buying.String())
	fmt.Printf("Amount: %d\n", offerEntry.Amount)
	// TODO: Price may need custom rule to clarify numerator and denominator.
	fmt.Printf("Price: %v\n", offerEntry.Price)
	fmt.Printf("Flags: %d\n", offerEntry.Flags)
	fmt.Printf("Ext: %v\n", offerEntry.Ext)
	return nil
}

func serializeDataEntry(dataEntry *xdr.DataEntry) error {
	fmt.Println("Found data!")
	fmt.Printf("AccountId: %s\n", dataEntry.AccountId.Address())
	fmt.Printf("DataName: %s\n", dataEntry.DataName)
	fmt.Printf("DataValue: %x\n", dataEntry.DataValue)
	fmt.Printf("Ext: %v\n", dataEntry.Ext)
	return nil
}

func serializeTrustlineEntry(trustlineEntry *xdr.TrustLineEntry) error {
	fmt.Println("Found trust!")
	fmt.Printf("AccountId: %s\n", trustlineEntry.AccountId.Address())
	fmt.Printf("Asset: %s\n", trustlineEntry.Asset.String())
	fmt.Printf("Balance: %d\n", trustlineEntry.Balance)
	fmt.Printf("Limit: %d\n", trustlineEntry.Limit)
	fmt.Printf("Flags: %d\n", trustlineEntry.Flags)
	fmt.Printf("Ext: %v\n", trustlineEntry.Ext)
	return nil
}

func serializeAccountEntry(accountEntry *xdr.AccountEntry) error {
	fmt.Println("Found account!")
	fmt.Printf("AccountId: %s\n", accountEntry.AccountId.Address())
	fmt.Printf("Balance: %d\n", accountEntry.Balance)
	fmt.Printf("SeqNum: %d\n", accountEntry.SeqNum)
	fmt.Printf("NumSubEntries: %d\n", accountEntry.NumSubEntries)
	fmt.Printf("InflationDest: %s\n", accountEntry.InflationDest.Address())
	fmt.Printf("Flags: %d\n", accountEntry.Flags)
	fmt.Printf("HomeDomain: %s\n", accountEntry.HomeDomain)
	fmt.Printf("Thresholds: %v\n", accountEntry.Thresholds)
	fmt.Printf("Signers: %v\n", accountEntry.Signers)
	fmt.Printf("Ext: %v\n", accountEntry.Ext)
	return nil
}
