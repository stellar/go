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
func (handler GetAccountsHandler) GetResourcePage(r *http.Request) ([]hal.Pageable, error) {
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
