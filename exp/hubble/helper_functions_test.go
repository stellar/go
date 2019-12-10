package hubble

import "github.com/stellar/go/xdr"

func makeLedgerEntryChangeAccount(entry *xdr.AccountEntry) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
		State: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: entry,
			},
		},
	}
}

func makeLedgerEntryChangeTrustline(issuer, code string, balance, limit int) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
		State: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeTrustline,
				TrustLine: &xdr.TrustLineEntry{
					AccountId: xdr.MustAddress(issuer),
					Asset:     xdr.MustNewCreditAsset(code, issuer),
					Balance:   xdr.Int64(balance),
					Limit:     xdr.Int64(limit),
				},
			},
		},
	}
}

func makeLedgerEntryChangeOffer(offerID int, sellerID string) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
		State: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(offerID),
					SellerId: xdr.MustAddress(sellerID),
				},
			},
		},
	}
}

func makeLedgerEntryChangeData(address, name, value string) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
		State: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &xdr.DataEntry{
					AccountId: xdr.MustAddress(address),
					DataName:  xdr.String64(name),
					DataValue: xdr.DataValue(value),
				},
			},
		},
	}
}

func makeLedgerEntryChangeSeqnumState(seqnum uint32) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
		State: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(seqnum),
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{},
			},
		},
	}
}

func makeLedgerEntryChangeAccountRemoved(address string) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
		Removed: &xdr.LedgerKey{
			Account: &xdr.LedgerKeyAccount{
				AccountId: xdr.MustAddress(address),
			},
		},
	}
}

func makeLedgerEntryChangeTrustlineRemoved(issuer, code string) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
		Removed: &xdr.LedgerKey{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.LedgerKeyTrustLine{
				AccountId: xdr.MustAddress(issuer),
				Asset:     xdr.MustNewCreditAsset(code, issuer),
			},
		},
	}
}

func makeLedgerEntryChangeDataRemoved(key string) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
		Removed: &xdr.LedgerKey{
			Type: xdr.LedgerEntryTypeData,
			Data: &xdr.LedgerKeyData{
				DataName: xdr.String64(key),
			},
		},
	}
}

func makeLedgerEntryChangeOfferRemoved(id int) *xdr.LedgerEntryChange {
	return &xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
		Removed: &xdr.LedgerKey{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.LedgerKeyOffer{
				OfferId: xdr.Int64(id),
			},
		},
	}
}
