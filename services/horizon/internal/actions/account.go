package actions

import (
	"context"

	"github.com/stellar/go/clients/horizon"
	pHorizon "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/hchi"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
)

func AccountInfo(ctx context.Context, cq *core.Q, hq *history.Q, protocolVersion int32) (*pHorizon.Account, error) {
	addr := hchi.AccountID(ctx)
	if addr == "" {
		// This should be impossible because of accountIdMiddleware
		return nil, errors.New("account address is empty")
	}

	var (
		coreRecord     core.Account
		coreData       []core.AccountData
		coreSigners    []core.Signer
		coreTrustlines []core.Trustline
		historyRecord  history.Account
		resource       horizon.Account
	)

	err := cq.AccountByAddress(&coreRecord, addr, protocolVersion)
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

	err = cq.TrustlinesByAddress(&coreTrustlines, addr, protocolVersion)
	if err != nil {
		return nil, errors.Wrap(err, "getting core trustline")
	}

	err = hq.AccountByAddress(&historyRecord, addr)
	// Do not fail when we cannot find the history record... it probably just
	// means that the account was created outside of our known history range.
	if err != nil && !hq.NoRows(err) {
		return nil, errors.Wrap(err, "getting history record")
	}

	err = resourceadapter.PopulateAccount(
		ctx,
		&resource,
		coreRecord,
		coreData,
		coreSigners,
		coreTrustlines,
		historyRecord,
	)

	return &resource, errors.Wrap(err, "populating account")
}
