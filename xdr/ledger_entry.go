package xdr

import "fmt"

// LedgerKey implements the `Keyer` interface.
func (entry *LedgerEntry) LedgerKey() LedgerKey {
	key, err := entry.GetLedgerKey()
	if err != nil {
		panic(err)
	}
	return key
}

// GetLedgerKey implements the `Keyer` interface, extracts the
// information from a `LedgerEntry` into a `LedgerKey`, and returns
// an error if it is unable to do so.
func (entry *LedgerEntry) GetLedgerKey() (LedgerKey, error) {
	var body interface{}

	switch entry.Data.Type {
	case LedgerEntryTypeAccount:
		account, ok := entry.Data.GetAccount()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get account")
		}
		body = LedgerKeyAccount{
			AccountId: account.AccountId,
		}
	case LedgerEntryTypeData:
		data, ok := entry.Data.GetData()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get data")
		}
		body = LedgerKeyData{
			AccountId: data.AccountId,
			DataName:  data.DataName,
		}
	case LedgerEntryTypeOffer:
		offer, ok := entry.Data.GetOffer()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get offer")
		}
		body = LedgerKeyOffer{
			SellerId: offer.SellerId,
			OfferId:  offer.OfferId,
		}
	case LedgerEntryTypeTrustline:
		tline, ok := entry.Data.GetTrustLine()
		if !ok {
			return LedgerKey{}, fmt.Errorf("could not get trustline")
		}
		body = LedgerKeyTrustLine{
			AccountId: tline.AccountId,
			Asset:     tline.Asset,
		}
	default:
		return LedgerKey{}, fmt.Errorf("unknown entry type: %v", entry.Data.Type)
	}

	ret, err := NewLedgerKey(entry.Data.Type, body)
	if err != nil {
		return LedgerKey{}, err
	}

	return ret, nil
}
