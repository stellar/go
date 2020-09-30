package xdr

import "fmt"

// LedgerKey implements the `Keyer` interface
func (entry *LedgerEntry) LedgerKey() LedgerKey {
	var body interface{}

	switch entry.Data.Type {
	case LedgerEntryTypeAccount:
		account := entry.Data.MustAccount()
		body = LedgerKeyAccount{
			AccountId: account.AccountId,
		}
	case LedgerEntryTypeData:
		data := entry.Data.MustData()
		body = LedgerKeyData{
			AccountId: data.AccountId,
			DataName:  data.DataName,
		}
	case LedgerEntryTypeOffer:
		offer := entry.Data.MustOffer()
		body = LedgerKeyOffer{
			SellerId: offer.SellerId,
			OfferId:  offer.OfferId,
		}
	case LedgerEntryTypeTrustline:
		tline := entry.Data.MustTrustLine()
		body = LedgerKeyTrustLine{
			AccountId: tline.AccountId,
			Asset:     tline.Asset,
		}
	case LedgerEntryTypeClaimableBalance:
		cBalance := entry.Data.MustClaimableBalance()
		body = LedgerKeyClaimableBalance{
			BalanceId: cBalance.BalanceId,
		}
	default:
		panic(fmt.Errorf("Unknown entry type: %v", entry.Data.Type))
	}

	ret, err := NewLedgerKey(entry.Data.Type, body)
	if err != nil {
		panic(err)
	}

	return ret
}

// SponsoringID return SponsorshipDescriptor for a given ledger entry
func (entry *LedgerEntry) SponsoringID() SponsorshipDescriptor {
	var sponsor SponsorshipDescriptor
	if entry.Ext.V == 1 && entry.Ext.V1 != nil {
		sponsor = entry.Ext.V1.SponsoringId
	}
	return sponsor
}
