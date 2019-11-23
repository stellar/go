// +build go1.13

package hubble

import (
	"fmt"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type accountState struct {
	address    string
	seqnum     uint32
	balance    uint32
	signers    []signer
	trustlines map[string]trustline
	offers     map[uint32]offer
	data       map[string][]byte
	// TODO: May want to track other fields in AccountEntry.
}

type signer struct {
	address string
	weight  uint32
}

type trustline struct {
	asset      string
	balance    uint32
	limit      uint32
	authorized bool
	// TODO: Add liabilities.
}

type offer struct {
	id         uint32
	seller     string // seller address
	selling    string // selling asset
	buying     string // buying asset
	amount     uint32
	priceNum   uint16
	priceDenom uint16
	// TODO: Add flags.
}

func makeNewAccountState(state *accountState, change *xdr.LedgerEntryChange) (*accountState, error) {
	// TODO: Handle account removal.
	var newAccountState accountState
	address, err := getAccountID(change, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not get address")
	}
	state.address = address

	seqnum, err := getSeqnum(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get seqnum")
	}
	newAccountState.seqnum = seqnum

	balance, err := getBalance(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not get balance")
	}
	newAccountState.balance = balance

	signers, err := getSigners(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not set signers")
	}
	newAccountState.signers = signers

	trustlines, err := getTrustlines(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not update trustlines")
	}
	newAccountState.trustlines = trustlines

	offers, err := getOffers(state, change)
	if err != nil {
		return nil, errors.Wrap(err, "could not update offers")
	}
	newAccountState.offers = offers

	return &newAccountState, nil
}

func getAccountID(change *xdr.LedgerEntryChange, states ...*accountState) (string, error) {
	// If the address has already been set on the account state, we return it.
	// We pass this as a optional parameter, so we can use this function more easily in the processor.
	if len(states) == 1 {
		return states[0].address, nil
	}

	key := change.LedgerKey()
	var accountID xdr.AccountId
	switch keyType := key.Type; keyType {
	case xdr.LedgerEntryTypeAccount:
		accountID = key.MustAccount().AccountId
	case xdr.LedgerEntryTypeTrustline:
		accountID = key.MustTrustLine().AccountId
	case xdr.LedgerEntryTypeOffer:
		accountID = key.MustOffer().SellerId
	case xdr.LedgerEntryTypeData:
		accountID = key.MustData().AccountId
	default:
		return "", fmt.Errorf("Unknown entry type: %v", keyType)
	}
	return accountID.Address(), nil
}

func getSeqnum(state *accountState, change *xdr.LedgerEntryChange) (uint32, error) {
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

func getBalance(state *accountState, change *xdr.LedgerEntryChange) (uint32, error) {
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

func getSigners(state *accountState, change *xdr.LedgerEntryChange) ([]signer, error) {
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

func getTrustlines(state *accountState, change *xdr.LedgerEntryChange) (map[string]trustline, error) {
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
		if _, ok := trustlines[asset]; ok {
			delete(trustlines, asset)
		}
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

func getOffers(state *accountState, change *xdr.LedgerEntryChange) (map[uint32]offer, error) {
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
		if _, ok := offers[id]; ok {
			delete(offers, id)
		}
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

func getData(state *accountState, change *xdr.LedgerEntryChange) (map[string][]byte, error) {
	if change.EntryType() != xdr.LedgerEntryTypeData {
		return state.data, nil
	}

	var data map[string][]byte
	for k, v := range state.data {
		data[k] = v
	}

	if change.Type == xdr.LedgerEntryChangeTypeLedgerEntryRemoved {
		name := string(change.MustRemoved().Data.DataName)
		if _, ok := data[name]; ok {
			delete(data, name)
		}
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
