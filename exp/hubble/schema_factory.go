// +build go1.13

package hubble

import (
	"fmt"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func makeNewAccountState(state *accountState, change *xdr.LedgerEntryChange) (*accountState, error) {
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved && change.EntryType() == xdr.LedgerEntryTypeAccount {
		return nil, nil
	}

	var newAccountState accountState
	address, err := makeAccountIDFromStateOrChange(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get address")
	}
	newAccountState.address = address

	seqnum, err := makeSeqnum(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get seqnum")
	}
	newAccountState.seqnum = seqnum

	balance, err := makeBalance(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get balance")
	}
	newAccountState.balance = balance

	signers, err := makeSigners(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not set signers")
	}
	newAccountState.signers = signers

	trustlines, err := makeTrustlines(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not update trustlines")
	}
	newAccountState.trustlines = trustlines

	offers, err := makeOffers(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not update offers")
	}
	newAccountState.offers = offers

	data, err := makeData(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not update data")
	}
	newAccountState.data = data

	return &newAccountState, nil
}

func makeAccountIDFromStateOrChange(state *accountState, change *xdr.LedgerEntryChange) (string, error) {
	if state != nil {
		return state.address, nil
	}
	return makeAccountIDFromChange(change)
}

func makeAccountIDFromChange(change *xdr.LedgerEntryChange) (string, error) {
	key := change.LedgerKey()
	var accountID xdr.AccountId
	switch keyType := key.Type; keyType {
	case xdr.LedgerEntryTypeAccount:
		account, ok := key.GetAccount()
		if !ok {
			return "", fmt.Errorf("could not get account")
		}
		accountID = account.AccountId
	case xdr.LedgerEntryTypeTrustline:
		trustline, ok := key.GetTrustLine()
		if !ok {
			return "", fmt.Errorf("could not get trustline")
		}
		accountID = trustline.AccountId
	case xdr.LedgerEntryTypeOffer:
		offer, ok := key.GetOffer()
		if !ok {
			return "", fmt.Errorf("could not get offer")
		}
		accountID = offer.SellerId
	case xdr.LedgerEntryTypeData:
		data, ok := key.GetData()
		if !ok {
			return "", fmt.Errorf("could not get data")
		}
		accountID = data.AccountId
	default:
		return "", fmt.Errorf("Unknown entry type: %v", keyType)
	}
	return accountID.Address(), nil
}

func makeSeqnum(state *accountState, change *xdr.LedgerEntryChange) (uint32, error) {
	// Removed entries do not change the last modified ledger seqnum.
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		return state.seqnum, nil
	}

	// TODO: Use state to check if the change of seqnum is valid.

	entry, err := getLedgerEntry(change)
	if err != nil {
		return 0, errors.Wrap(err, "could not get ledger entry")
	}
	return uint32(entry.LastModifiedLedgerSeq), nil
}

func makeBalance(state *accountState, change *xdr.LedgerEntryChange) (uint32, error) {
	account, err := getAccountEntry(change)
	if err != nil {
		return 0, err
	}

	// The change does not update the account state, so we return
	// the current balance.
	if account == nil {
		return state.balance, nil
	}
	return uint32(account.Balance), nil
}

func makeSigners(state *accountState, change *xdr.LedgerEntryChange) ([]signer, error) {
	account, err := getAccountEntry(change)
	if err != nil {
		return nil, err
	}

	// The change does not update the account state, so we return
	// the current signers.
	if account == nil {
		return state.signers, nil
	}

	var signers []signer
	for _, accountSigner := range account.Signers {
		signers = append(signers, signer{
			address: accountSigner.Key.Address(),
			weight:  uint32(accountSigner.Weight),
		})
	}
	return signers, nil
}

func makeTrustlines(state *accountState, change *xdr.LedgerEntryChange) (map[string]trustline, error) {
	// Get current trustlines.
	var trustlines map[string]trustline
	if state != nil {
		if state.trustlines != nil {
			trustlines = make(map[string]trustline)
			for k, v := range state.trustlines {
				trustlines[k] = v
			}
		}
	}

	// Return existing trustlines if the change is not of type trustline.
	if change.EntryType() != xdr.LedgerEntryTypeTrustline {
		return trustlines, nil
	}

	// If the change is removed, remove the corresponding trustline.
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		removeEntry, ok := change.GetRemoved()
		if !ok {
			return nil, fmt.Errorf("Could not get removed ledger key")
		}
		asset := removeEntry.TrustLine.Asset.String()
		delete(trustlines, asset)
		return trustlines, nil
	}

	entry, err := getLedgerEntry(change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ledger entry")
	}
	trustlineEntry, ok := entry.Data.GetTrustLine()
	if !ok {
		return nil, fmt.Errorf("Could not get trustline")
	}
	assetKey := trustlineEntry.Asset.String()
	newTrustline := trustline{
		asset:      assetKey,
		balance:    uint32(trustlineEntry.Balance),
		limit:      uint32(trustlineEntry.Limit),
		authorized: (trustlineEntry.Flags != 0),
	}

	trustlines[assetKey] = newTrustline
	return trustlines, nil
}

func makeOffers(state *accountState, change *xdr.LedgerEntryChange) (map[uint32]offer, error) {
	// Return existing offers if the change is not of type offer.
	if change.EntryType() != xdr.LedgerEntryTypeOffer {
		if state != nil {
			return state.offers, nil
		}
		return nil, nil
	}

	// Get current offers.
	offers := make(map[uint32]offer)
	if state != nil {
		for k, v := range state.offers {
			offers[k] = v
		}
	}

	// If the change is removed, remove the corresponding offer.
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		removeEntry, ok := change.GetRemoved()
		if !ok {
			return nil, fmt.Errorf("Could not get removed ledger key")
		}
		id := uint32(removeEntry.Offer.OfferId)
		delete(offers, id)
		return offers, nil
	}

	// Get and store offer.
	entry, err := getLedgerEntry(change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ledger entry")
	}
	offerEntry, ok := entry.Data.GetOffer()
	if !ok {
		return nil, fmt.Errorf("Could not get offer")
	}

	offerID := uint32(offerEntry.OfferId)
	newOffer := offer{
		id:         offerID,
		seller:     offerEntry.SellerId.Address(),
		selling:    offerEntry.Selling.String(),
		buying:     offerEntry.Buying.String(),
		amount:     uint32(offerEntry.Amount),
		priceNum:   uint16(offerEntry.Price.N),
		priceDenom: uint16(offerEntry.Price.D),
	}

	offers[offerID] = newOffer
	return offers, nil
}

func makeData(state *accountState, change *xdr.LedgerEntryChange) (map[string][]byte, error) {
	// Return existing data if the change is not of type data.
	if change.EntryType() != xdr.LedgerEntryTypeData {
		if state != nil {
			return state.data, nil
		}
		return nil, nil
	}

	// Copy current data.
	data := make(map[string][]byte)
	if state != nil {
		for k, v := range state.data {
			data[k] = v
		}
	}

	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		removeEntry, ok := change.GetRemoved()
		if !ok {
			return nil, fmt.Errorf("Could not get removed ledger key")
		}
		name := string(removeEntry.Data.DataName)
		delete(data, name)
		return data, nil
	}

	// Get and store the data key-value pair.
	entry, err := getLedgerEntry(change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ledger entry")
	}
	dataEntry, ok := entry.Data.GetData()
	if !ok {
		return nil, fmt.Errorf("Could not get data")
	}
	data[string(dataEntry.DataName)] = dataEntry.DataValue
	return data, nil
}

func getAccountEntry(change *xdr.LedgerEntryChange) (*xdr.AccountEntry, error) {
	if change.EntryType() != xdr.LedgerEntryTypeAccount {
		return nil, nil
	}

	entry, err := getLedgerEntry(change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ledger entry")
	}

	account, ok := entry.Data.GetAccount()
	if !ok {
		return nil, fmt.Errorf("Could not get account")
	}

	return &account, nil
}

func getLedgerEntry(change *xdr.LedgerEntryChange) (*xdr.LedgerEntry, error) {
	var (
		account xdr.LedgerEntry
		ok      bool
	)
	entryType := change.Type
	switch entryType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		account, ok = change.GetCreated()
	case xdr.LedgerEntryChangeTypeLedgerEntryState:
		account, ok = change.GetState()
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		account, ok = change.GetUpdated()
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return nil, fmt.Errorf("Entry type %v does not have LedgerEntry", entryType)
	default:
		return nil, fmt.Errorf("Unknown entry type: %v", entryType)
	}
	if !ok {
		return nil, fmt.Errorf("Could not get account from entry type %v", entryType)
	}
	return &account, nil
}
