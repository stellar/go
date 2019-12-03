// +build go1.13

package hubble

import (
	"fmt"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func makeNewAccountState(state *accountState, change *xdr.LedgerEntryChange) (*accountState, error) {
	// TODO: Handle account removal.
	var newAccountState accountState
	address, err := makeAccountIDFromStateOrChange(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get address")
	}
	state.address = address

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
	entry, ok := change.GetLedgerEntry()
	if !ok {
		return "", fmt.Errorf("Could not get ledger entry from change")
	}
	var accountID xdr.AccountId
	entryData := entry.Data
	switch entryType := entryData.Type; entryType {
	case xdr.LedgerEntryTypeAccount:
		account, ok := entryData.GetAccount()
		if !ok {
			return "", fmt.Errorf("could not get account")
		}
		accountID = account.AccountId
	case xdr.LedgerEntryTypeTrustline:
		trustline, ok := entryData.GetTrustLine()
		if !ok {
			return "", fmt.Errorf("could not get trustline")
		}
		accountID = trustline.AccountId
	case xdr.LedgerEntryTypeOffer:
		offer, ok := entryData.GetOffer()
		if !ok {
			return "", fmt.Errorf("could not get offer")
		}
		accountID = offer.SellerId
	case xdr.LedgerEntryTypeData:
		data, ok := entryData.GetData()
		if !ok {
			return "", fmt.Errorf("could not get data")
		}
		accountID = data.AccountId
	default:
		return "", fmt.Errorf("Unknown entry type: %v", entryType)
	}
	address, err := accountID.GetAddress()
	if err != nil {
		return "", errors.Wrap(err, "could not get address")
	}
	return address, nil
}

func makeSeqnum(state *accountState, change *xdr.LedgerEntryChange) (uint32, error) {
	var seqnum xdr.Uint32
	switch entryType := change.Type; entryType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		seqnum = change.MustCreated().LastModifiedLedgerSeq
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		seqnum = change.MustUpdated().LastModifiedLedgerSeq
	case xdr.LedgerEntryChangeTypeLedgerEntryState:
		seqnum = change.MustState().LastModifiedLedgerSeq

	// We do not need to update the seqnum for removed changes, because
	// we just remove the accompanying account's state.
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return 0, nil
	default:
		return 0, fmt.Errorf("Unknown entry type: %v", entryType)
	}
	return uint32(seqnum), nil
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
	if change.EntryType() != xdr.LedgerEntryTypeTrustline {
		return state.trustlines, nil
	}

	// Copy the current state trustlines.
	trustlines := make(map[string]trustline)
	for k, v := range state.trustlines {
		trustlines[k] = v
	}

	// If the change is removed, remove the corresponding trustline.
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		asset := change.MustRemoved().TrustLine.Asset.String()
		delete(trustlines, asset)
		return trustlines, nil
	}

	// Get and store the new trustline.
	var trustlineEntry xdr.TrustLineEntry
	switch entryType := change.Type; entryType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		trustlineEntry = change.MustCreated().Data.MustTrustLine()
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		trustlineEntry = change.MustUpdated().Data.MustTrustLine()
	case xdr.LedgerEntryChangeTypeLedgerEntryState:
		trustlineEntry = change.MustState().Data.MustTrustLine()
	default:
		return nil, fmt.Errorf("Unknown entry type: %v", entryType)
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
	if change.EntryType() != xdr.LedgerEntryTypeOffer {
		return state.offers, nil
	}

	// Copy the current state offers.
	offers := make(map[uint32]offer)
	for k, v := range state.offers {
		offers[k] = v
	}

	// If the change is removed, remove the corresponding offer.
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		id := uint32(change.MustRemoved().Offer.OfferId)
		delete(offers, id)
		return offers, nil
	}

	// Get and store the offer.
	var offerEntry xdr.OfferEntry
	switch entryType := change.Type; entryType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		offerEntry = change.MustCreated().Data.MustOffer()
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		offerEntry = change.MustUpdated().Data.MustOffer()
	case xdr.LedgerEntryChangeTypeLedgerEntryState:
		offerEntry = change.MustState().Data.MustOffer()
	default:
		return nil, fmt.Errorf("Unknown entry type: %v", entryType)
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
	if change.EntryType() != xdr.LedgerEntryTypeData {
		return state.data, nil
	}

	data := make(map[string][]byte)
	for k, v := range state.data {
		data[k] = v
	}

	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		name := string(change.MustRemoved().Data.DataName)
		delete(data, name)
		return data, nil
	}

	// Get and store the data key-value pair.
	var dataEntry xdr.DataEntry
	switch entryType := change.Type; entryType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		dataEntry = change.MustCreated().Data.MustData()
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		dataEntry = change.MustUpdated().Data.MustData()
	case xdr.LedgerEntryChangeTypeLedgerEntryState:
		dataEntry = change.MustState().Data.MustData()
	default:
		return nil, fmt.Errorf("Unknown entry type: %v", entryType)
	}
	data[string(dataEntry.DataName)] = dataEntry.DataValue
	return data, nil
}

func getAccountEntry(change *xdr.LedgerEntryChange) (*xdr.AccountEntry, error) {
	if change.EntryType() != xdr.LedgerEntryTypeAccount {
		return nil, nil
	}
	var account xdr.AccountEntry
	switch entryType := change.Type; entryType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
		account = change.MustCreated().Data.MustAccount()
	case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		account = change.MustUpdated().Data.MustAccount()
	case xdr.LedgerEntryChangeTypeLedgerEntryState:
		account = change.MustState().Data.MustAccount()
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return nil, nil
	default:
		return nil, fmt.Errorf("Unknown entry type: %v", entryType)
	}
	return &account, nil
}
