// +build go1.13

package hubble

import (
	"fmt"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func makeNewAccountState(state *accountState, change *xdr.LedgerEntryChange) (*accountState, error) {
	accountRemoved, err := isAccountRemoved(change)
	if err != nil {
		return nil, errors.Wrap(err, "could not check removed account")
	}
	// If the account has been removed, then we return a nil account state.
	if accountRemoved {
		return nil, nil
	}

	// We should never be given a nil pointer for account state, rather one
	// to an empty accountState struct. If we are given a nil pointer to state
	// somehow, we replace it with the desired input. This prevents repeated,
	// per-function checks for nil state.
	if state == nil {
		state = &accountState{}
		accountID, aerr := makeAccountIDFromChange(change)
		if err != nil {
			return nil, errors.Wrap(aerr, "could not get ledger account address")
		}
		state.address = accountID
	}

	var newAccountState accountState
	newAccountState.address = state.address

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

func isAccountRemoved(change *xdr.LedgerEntryChange) (bool, error) {
	// If the change is not of Removed type, it cannot represent the
	// removal of an Account.
	if change.Type != xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		return false, nil
	}

	ledgerKey, ok := change.GetRemoved()
	if !ok {
		return false, fmt.Errorf("Could not get ledger key from Removed struct")
	}

	return (ledgerKey.Type == xdr.LedgerEntryTypeAccount), nil
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
	// Removed entries would not be of Account type, and thus
	// do not change the last modified ledger seqnum.
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		return state.seqnum, nil
	}

	entry, ok := change.GetLedgerEntry()
	if !ok {
		return 0, fmt.Errorf("Could not get ledger entry")
	}
	return uint32(entry.LastModifiedLedgerSeq), nil
}

func makeBalance(state *accountState, change *xdr.LedgerEntryChange) (uint32, error) {
	// If the change is a non-account removal or a change of non-account type, then
	// we simply return the current balance (or 0, if we were passed a nil state pointer).
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		return state.balance, nil
	}

	entry, ok := change.GetLedgerEntry()
	if !ok {
		return state.balance, nil
	}

	account, ok := entry.Data.GetAccount()
	if !ok {
		return state.balance, nil
	}

	return uint32(account.Balance), nil
}

func makeSigners(state *accountState, change *xdr.LedgerEntryChange) ([]signer, error) {
	// If the change is a non-account removal or a change of non-account type, then
	// we simply return the current signers (or empty list, if we were passed a nil state pointer).
	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		return state.signers, nil
	}

	entry, ok := change.GetLedgerEntry()
	if !ok {
		return state.signers, nil
	}

	account, ok := entry.Data.GetAccount()
	if !ok {
		return state.signers, nil
	}

	var signers []signer
	for _, accountSigner := range account.Signers {
		// TODO: Replace `SignerKey.Address()` with a panic-free version.
		signers = append(signers, signer{
			address: accountSigner.Key.Address(),
			weight:  uint32(accountSigner.Weight),
		})
	}
	return signers, nil
}

func makeTrustlines(state *accountState, change *xdr.LedgerEntryChange) (map[string]trustline, error) {
	// TODO: Replace reference to `EntryType` with a panic-free version.
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
		removeEntry, ok := change.GetRemoved()
		if !ok {
			return nil, fmt.Errorf("Could not get removed ledger key")
		}
		asset := removeEntry.TrustLine.Asset.String()
		delete(trustlines, asset)
		return trustlines, nil
	}

	// Get the trustline entry from the change and an error if we cannot.
	entry, ok := change.GetLedgerEntry()
	if !ok {
		return nil, fmt.Errorf("Could not get ledger entry")
	}
	trustlineEntry, ok := entry.Data.GetTrustLine()
	if !ok {
		return nil, fmt.Errorf("Could not get trustline entry")
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
	// TODO: Replace reference to `EntryType` with a panic-free version.
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
		removeEntry, ok := change.GetRemoved()
		if !ok {
			return nil, fmt.Errorf("Could not get removed ledger key")
		}
		id := uint32(removeEntry.Offer.OfferId)
		delete(offers, id)
		return offers, nil
	}

	// Get and store the offer.
	entry, ok := change.GetLedgerEntry()
	if !ok {
		return nil, fmt.Errorf("Could not get ledger entry")
	}
	offerEntry, ok := entry.Data.GetOffer()
	if !ok {
		return nil, fmt.Errorf("Could not get offer entry")
	}

	offerSellerAddress, err := offerEntry.SellerId.GetAddress()
	if err != nil {
		return nil, errors.Wrap(err, "could not get offer seller address")
	}

	offerID := uint32(offerEntry.OfferId)
	newOffer := offer{
		id:         offerID,
		seller:     offerSellerAddress,
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
	// TODO: Replace reference to `EntryType` with a panic-free version.
	if change.EntryType() != xdr.LedgerEntryTypeData {
		return state.data, nil
	}

	data := make(map[string][]byte)
	for k, v := range state.data {
		data[k] = v
	}

	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		key, ok := change.GetRemoved()
		if !ok {
			return nil, fmt.Errorf("Could not get removed ledger key")
		}
		name := string(key.Data.DataName)
		delete(data, name)
		return data, nil
	}

	// Get and store the data key-value pair.
	entry, ok := change.GetLedgerEntry()
	if !ok {
		return nil, fmt.Errorf("Could not get ledger entry")
	}

	dataEntry, ok := entry.Data.GetData()
	if !ok {
		return nil, fmt.Errorf("Could not get data entry")
	}

	data[string(dataEntry.DataName)] = dataEntry.DataValue
	return data, nil
}
