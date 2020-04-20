package actions

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

func accountFromCoreDB(ctx context.Context, cq *core.Q, addr string) (*protocol.Account, error) {
	var (
		coreRecord     core.Account
		coreData       []core.AccountData
		coreSigners    []core.Signer
		coreTrustlines []core.Trustline
		resource       protocol.Account
	)

	err := cq.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "starting repeatable read transaction")
	}
	defer cq.Rollback()

	err = cq.AccountByAddress(&coreRecord, addr)
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

	return &resource, nil
}

func accountFromExperimentalIngestion(ctx context.Context, hq *history.Q, addr string) (*protocol.Account, error) {
	var (
		horizonRecord     history.AccountEntry
		horizonData       []history.Data
		horizonSigners    []history.AccountSigner
		horizonTrustLines []history.TrustLine
		resouce           protocol.Account
	)

	horizonRecord, err := hq.GetAccountByID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history account record")
	}

	horizonData, err = hq.GetAccountDataByAccountID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history account data")
	}

	horizonSigners, err = hq.GetAccountSignersByAccountID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history signers")
	}

	horizonTrustLines, err = hq.GetTrustLinesByAccountID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history trustlines")
	}

	err = resourceadapter.PopulateAccountEntry(
		ctx,
		&resouce,
		horizonRecord,
		horizonData,
		horizonSigners,
		horizonTrustLines,
	)
	if err != nil {
		return nil, errors.Wrap(err, "populating account entry")
	}

	return &resouce, nil
}

// AccountInfo returns the information about an account identified by addr.
func AccountInfo(ctx context.Context, cq *core.Q, hq *history.Q, addr string) (*protocol.Account, error) {
	var resource, otherResource *protocol.Account
	var err error

	resource, err = accountFromExperimentalIngestion(ctx, hq, addr)
	if err != nil {
		return nil, err
	}

	otherResource, err = accountFromCoreDB(ctx, cq, addr)
	if err != nil {
		err = errors.Wrap(err, "Could not fetch account from core DB")
	}

	if err == nil {
		if err = accountResourcesEqual(*resource, *otherResource); err != nil {
			log.Ctx(ctx).WithFields(log.F{
				"err":            err,
				"accounts_check": true, // So it's easy to find all diffs
			}).Warn("error comparing core and horizon accounts")
		}
	}

	return resource, err
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

// AccountsQuery query struct for accounts end-point
type AccountsQuery struct {
	Signer      string `schema:"signer" valid:"accountID,optional"`
	AssetFilter string `schema:"asset" valid:"asset,optional"`
}

// URITemplate returns a rfc6570 URI template the query struct
func (q AccountsQuery) URITemplate() string {
	return "/accounts{?" + strings.Join(GetURIParams(&q, true), ",") + "}"
}

var invalidAccountsParams = problem.P{
	Type:   "invalid_accounts_params",
	Title:  "Invalid Accounts Parameters",
	Status: http.StatusBadRequest,
	Detail: "A filter is required. Please ensure that you are including a signer or an asset.",
}

// Validate runs custom validations.
func (q AccountsQuery) Validate() error {
	if q.AssetFilter == "native" {
		return problem.MakeInvalidFieldProblem(
			"asset",
			errors.New("you can't filter by asset: native"),
		)
	}

	if len(q.Signer) == 0 && q.Asset() == nil {
		return invalidAccountsParams
	}

	if len(q.Signer) > 0 && q.Asset() != nil {
		return problem.MakeInvalidFieldProblem(
			"signer",
			errors.New("you can't filter by signer and asset at the same time"),
		)
	}

	return nil
}

// Asset returns an xdr.Asset representing the Asset we want to find the trustees by.
func (q AccountsQuery) Asset() *xdr.Asset {
	if len(q.AssetFilter) == 0 {
		return nil
	}

	parts := strings.Split(q.AssetFilter, ":")
	asset := xdr.MustNewCreditAsset(parts[0], parts[1])

	return &asset
}

// GetAccountsHandler is the action handler for the /accounts endpoint
type GetAccountsHandler struct {
}

// GetResourcePage returns a page containing the account records that have
// `signer` as a signer or have a trustline to the given asset.
func (handler GetAccountsHandler) GetResourcePage(
	w HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	ctx := r.Context()
	pq, err := GetPageQuery(r, DisableCursorValidation)
	if err != nil {
		return nil, err
	}

	historyQ, err := HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	qp := AccountsQuery{}
	err = GetParams(&qp, r)
	if err != nil {
		return nil, err
	}

	var records []history.AccountEntry

	if len(qp.Signer) > 0 {
		records, err = historyQ.AccountEntriesForSigner(qp.Signer, pq)
		if err != nil {
			return nil, errors.Wrap(err, "loading account records")
		}
	} else {
		records, err = historyQ.AccountsForAsset(*qp.Asset(), pq)
		if err != nil {
			return nil, errors.Wrap(err, "loading account records")
		}
	}

	accounts := make([]hal.Pageable, 0, len(records))

	if len(records) == 0 {
		// early return
		return accounts, nil
	}

	accountIDs := make([]string, 0, len(records))
	for _, record := range records {
		accountIDs = append(accountIDs, record.AccountID)
	}

	signers, err := handler.loadSigners(historyQ, accountIDs)
	if err != nil {
		return nil, err
	}

	trustlines, err := handler.loadTrustlines(historyQ, accountIDs)
	if err != nil {
		return nil, err
	}

	data, err := handler.loadData(historyQ, accountIDs)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		var res protocol.Account
		s := signers[record.AccountID]
		t := trustlines[record.AccountID]
		d := data[record.AccountID]

		resourceadapter.PopulateAccountEntry(ctx, &res, record, d, s, t)

		accounts = append(accounts, res)
	}

	return accounts, nil
}

func (handler GetAccountsHandler) loadData(historyQ *history.Q, accounts []string) (map[string][]history.Data, error) {
	data := make(map[string][]history.Data)

	records, err := historyQ.GetAccountDataByAccountsID(accounts)
	if err != nil {
		return data, errors.Wrap(err, "loading account data records by accounts id")
	}

	for _, record := range records {
		data[record.AccountID] = append(data[record.AccountID], record)
	}

	return data, nil
}

func (handler GetAccountsHandler) loadTrustlines(historyQ *history.Q, accounts []string) (map[string][]history.TrustLine, error) {
	trustLines := make(map[string][]history.TrustLine)

	records, err := historyQ.GetTrustLinesByAccountsID(accounts)
	if err != nil {
		return trustLines, errors.Wrap(err, "loading trustline records by accounts")
	}

	for _, record := range records {
		trustLines[record.AccountID] = append(trustLines[record.AccountID], record)
	}

	return trustLines, nil
}

func (handler GetAccountsHandler) loadSigners(historyQ *history.Q, accounts []string) (map[string][]history.AccountSigner, error) {
	signers := make(map[string][]history.AccountSigner)

	records, err := historyQ.SignersForAccounts(accounts)
	if err != nil {
		return signers, errors.Wrap(err, "loading account signers by account")
	}

	for _, record := range records {
		signers[record.Account] = append(signers[record.Account], record)
	}

	return signers, nil
}
