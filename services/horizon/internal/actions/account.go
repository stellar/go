package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
)

// AccountInfo returns the information about an account identified by addr.
func AccountInfo(ctx context.Context, cq *core.Q, hq *history.Q, addr string) (*protocol.Account, error) {
	var (
		coreRecord     core.Account
		coreData       []core.AccountData
		coreSigners    []core.Signer
		coreTrustlines []core.Trustline
		resource       protocol.Account
	)

	err := cq.AccountByAddress(&coreRecord, addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting core account record")
	}

	err = cq.AllDataByAddress(&coreData, addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting core account data")
	}

	err = cq.SignersByAddress(&coreSigners, addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting core signer")
	}

	err = cq.TrustlinesByAddress(&coreTrustlines, addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting core trustline")
	}

	err = resourceadapter.PopulateAccount(
		ctx,
		&resource,
		coreRecord,
		coreData,
		coreSigners,
		coreTrustlines,
	)
	if err != nil {
		return nil, errors.Wrap(err, "populating account")
	}

	c, err := json.Marshal(resource)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling resource")
	}

	// We send JSON bytes to compareAccountResults to prevent modifying
	// `resource` by any way.
	err = compareAccountResults(ctx, hq, c, addr)
	if err != nil {
		log.Ctx(ctx).WithFields(log.F{
			"err":            err,
			"accounts_check": true, // So it's easy to find all diffs
		}).Warn("error comparing core and horizon accounts")
	}

	return &resource, nil
}

func compareAccountResults(
	ctx context.Context,
	hq *history.Q,
	expectedResourceBytes []byte,
	addr string,
) error {
	var (
		horizonRecord     history.AccountEntry
		horizonData       []history.Data
		horizonSigners    []history.AccountSigner
		horizonTrustLines []history.TrustLine
		newResource       protocol.Account
	)

	horizonRecord, err := hq.GetAccountByID(addr)
	if err != nil {
		return err
	}

	horizonData, err = hq.GetAccountDataByAccountID(addr)
	if err != nil {
		return err
	}

	horizonSigners, err = hq.GetAccountSignersByAccountID(addr)
	if err != nil {
		return err
	}

	horizonTrustLines, err = hq.GetTrustLinesByAccountID(addr)
	if err != nil {
		return err
	}

	err = resourceadapter.PopulateAccountEntry(
		ctx,
		&newResource,
		horizonRecord,
		horizonData,
		horizonSigners,
		horizonTrustLines,
	)
	if err != nil {
		return err
	}

	var expectedResource protocol.Account
	err = json.Unmarshal(expectedResourceBytes, &expectedResource)
	if err != nil {
		return errors.Wrap(err, "Error unmarshaling expectedResourceBytes")
	}

	if err = accountResourcesEqual(newResource, expectedResource); err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf(
				"Core and Horizon accounts responses do not match: %+v %+v",
				expectedResource, newResource,
			))

	}

	return nil
}

// accountResourcesEqual compares two protocol.Account objects and returns an
// error if they are different but only if `LastModifiedLedger` fields are the
// same.
func accountResourcesEqual(actual, expected protocol.Account) error {
	if actual.Links != expected.Links {
		return errors.New("Links are different")
	}

	if actual.LastModifiedLedger != expected.LastModifiedLedger {
		// Modified at different ledgers so values will be different
		return nil
	}

	if actual.ID != expected.ID ||
		actual.AccountID != expected.AccountID ||
		actual.Sequence != expected.Sequence ||
		actual.SubentryCount != expected.SubentryCount ||
		actual.InflationDestination != expected.InflationDestination ||
		actual.HomeDomain != expected.HomeDomain ||
		actual.Thresholds != expected.Thresholds ||
		actual.Flags != expected.Flags {
		return errors.New("Main fields are different")
	}

	// Ignore PT

	// Balances
	balances := map[string]protocol.Balance{}
	for _, balance := range expected.Balances {
		id := balance.Asset.Type + balance.Asset.Code + balance.Asset.Issuer
		balances[id] = balance
	}

	for _, actualBalance := range actual.Balances {
		id := actualBalance.Asset.Type + actualBalance.Asset.Code + actualBalance.Asset.Issuer
		expectedBalance := balances[id]
		delete(balances, id)

		if expectedBalance.LastModifiedLedger != actualBalance.LastModifiedLedger {
			// Modified at different ledgers so values will be different
			continue
		}

		if expectedBalance.Balance != actualBalance.Balance ||
			expectedBalance.Limit != actualBalance.Limit ||
			expectedBalance.BuyingLiabilities != actualBalance.BuyingLiabilities ||
			expectedBalance.SellingLiabilities != actualBalance.SellingLiabilities {
			return errors.New("Balance " + id + " is different")
		}

		if expectedBalance.IsAuthorized == nil && actualBalance.IsAuthorized == nil {
			continue
		}

		if expectedBalance.IsAuthorized != nil && actualBalance.IsAuthorized != nil &&
			*expectedBalance.IsAuthorized == *actualBalance.IsAuthorized {
			continue
		}

		return errors.New("IsAuthorized is different for " + id)
	}

	if len(balances) > 0 {
		return errors.New("Some extra balances")
	}

	// Signers
	signers := map[string]protocol.Signer{}
	for _, signer := range expected.Signers {
		signers[signer.Key] = signer
	}

	for _, actualSigner := range actual.Signers {
		expectedSigner := signers[actualSigner.Key]
		delete(signers, actualSigner.Key)

		if expectedSigner != actualSigner {
			return errors.New("Signer is different")
		}
	}

	if len(signers) > 0 {
		return errors.New("Extra signers")
	}

	// Data
	data := map[string]string{}
	for key, value := range expected.Data {
		data[key] = value
	}

	for actualKey, actualValue := range actual.Data {
		expectedValue := data[actualKey]
		delete(data, actualKey)

		if expectedValue != actualValue {
			return errors.New("Data is different")
		}
	}

	if len(data) > 0 {
		return errors.New("Extra data")
	}

	return nil
}

// GetAccountsHandler is the action handler for the /accounts endpoint
type GetAccountsHandler struct {
	HistoryQ *history.Q
}

// GetResourcePage returns a page containing the account records that have
// `signer` as a signer. This doesn't return full account details resource
// because of the limitations of existing ingestion architecture. In a future,
// when the new ingestion system is fully integrated, this endpoint can be used
// to find accounts for signer but also accounts for assets, home domain,
// inflation_dest etc.
func (handler GetAccountsHandler) GetResourcePage(
	w HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	ctx := r.Context()

	signer, err := GetAccountID(r, "signer")
	if err != nil {
		return nil, err
	}

	pq, err := GetPageQuery(r, DisableCursorValidation)
	if err != nil {
		return nil, err
	}

	records, err := handler.HistoryQ.AccountsForSigner(signer.Address(), pq)
	if err != nil {
		return nil, errors.Wrap(err, "loading account records")
	}

	var accounts []hal.Pageable
	for _, record := range records {
		var res protocol.AccountSigner
		resourceadapter.PopulateAccountSigner(ctx, &res, record)
		accounts = append(accounts, res)
	}

	return accounts, nil
}
