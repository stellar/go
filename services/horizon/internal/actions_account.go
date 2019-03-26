package horizon

import (
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/render/hal"
)

// This file contains the actions:
//
// AccountShowAction: details for single account (including stellar-core state)

// Interface verifications
var _ actions.JSONer = (*AccountShowAction)(nil)
var _ actions.SingleObjectStreamer = (*AccountShowAction)(nil)

// AccountShowAction renders a account summary found by its address.
type AccountShowAction struct {
	Action
	Address        string
	CoreData       []core.AccountData
	CoreRecord     core.Account
	CoreSigners    []core.Signer
	CoreTrustlines []core.Trustline
	Resource       horizon.Account
}

// JSON is a method for actions.JSON
func (action *AccountShowAction) JSON() error {
	action.Do(
		action.loadParams,
		action.loadRecord,
		action.loadResource,
		func() { hal.Render(action.W, action.Resource) },
	)
	return action.Err
}

func (action *AccountShowAction) LoadEvent() (sse.Event, error) {
	action.Do(action.loadParams, action.loadRecord, action.loadResource)
	return sse.Event{Data: action.Resource}, action.Err
}

func (action *AccountShowAction) loadParams() {
	action.Address = action.GetAddress("account_id", actions.RequiredParam)
}

func (action *AccountShowAction) loadRecord() {
	action.Err = action.CoreQ().AccountByAddress(&action.CoreRecord, action.Address)
	if action.Err != nil {
		return
	}

	action.Err = action.CoreQ().AllDataByAddress(&action.CoreData, action.Address)
	if action.Err != nil {
		return
	}

	action.Err = action.CoreQ().SignersByAddress(&action.CoreSigners, action.Address)
	if action.Err != nil {
		return
	}

	action.Err = action.CoreQ().TrustlinesByAddress(&action.CoreTrustlines, action.Address)
}

func (action *AccountShowAction) loadResource() {
	action.Err = resourceadapter.PopulateAccount(
		action.R.Context(),
		&action.Resource,
		action.CoreRecord,
		action.CoreData,
		action.CoreSigners,
		action.CoreTrustlines,
	)
}
