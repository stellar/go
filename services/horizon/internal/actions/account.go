package actions

import (
	"context"
	"net/http"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// AccountInfo returns the information about an account identified by addr.
func AccountInfo(ctx context.Context, cq *core.Q, addr string) (*protocol.Account, error) {
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

	return &resource, errors.Wrap(err, "populating account")
}

// AccountsQuery query struct for accounts end-point
type AccountsQuery struct {
	Signer      string `schema:"signer" valid:"accountID,optional"`
	AssetType   string `schema:"asset_type" valid:"assetType,optional"`
	AssetIssuer string `schema:"asset_issuer" valid:"accountID,optional"`
	AssetCode   string `schema:"asset_code" valid:"-"`
}

var invalidAccountsParams = problem.P{
	Type:   "invalid_accounts_params",
	Title:  "Invalid Accounts Parameters",
	Status: http.StatusBadRequest,
	Detail: "A filter is required. Please ensure that you are including a signer or an asset (asset_type, asset_issuer and asset_code).",
}

// Validate runs custom validations.
func (q AccountsQuery) Validate() error {
	if len(q.Signer) == 0 && q.Asset() == nil {
		return invalidAccountsParams
	}

	err := validateAssetParams(q.AssetType, q.AssetCode, q.AssetIssuer, "")
	if err != nil {
		return err
	}

	if len(q.Signer) > 0 && q.Asset() != nil {
		return problem.MakeInvalidFieldProblem(
			"signer",
			errors.New("you can't filter by signer and asset at the same time"),
		)
	}

	if q.Asset() != nil && q.AssetType == "native" {
		return problem.MakeInvalidFieldProblem(
			"asset_type",
			errors.New("you can't filter by asset type: native"),
		)
	}

	return nil
}

// Asset returns an xdr.Asset representing the Asset we want to find the trustees by.
func (q AccountsQuery) Asset() *xdr.Asset {
	if len(q.AssetType) == 0 {
		return nil
	}

	asset, err := xdr.BuildAsset(
		q.AssetType,
		q.AssetIssuer,
		q.AssetCode,
	)

	if err != nil {
		panic(err)
	}

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

	var accounts []hal.Pageable

	historyQ, err := historyQFromRequest(r)
	if err != nil {
		return nil, err
	}

	qp := AccountsQuery{}
	err = GetParams(&qp, r)
	if err != nil {
		return nil, err
	}

	if len(qp.Signer) > 0 {
		records, err := historyQ.AccountsForSigner(qp.Signer, pq)
		if err != nil {
			return nil, errors.Wrap(err, "loading account records")
		}

		for _, record := range records {
			var res protocol.AccountSigner
			resourceadapter.PopulateAccountSigner(ctx, &res, record)
			accounts = append(accounts, res)
		}
	} else {
		records, err := historyQ.AccountsForAsset(*qp.Asset(), pq)
		if err != nil {
			return nil, errors.Wrap(err, "loading account records")
		}

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
