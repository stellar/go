package resourceadapter

import (
	"context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	. "github.com/stellar/go/protocols/resource/operations"
	"github.com/stellar/go/xdr"
	"github.com/stellar/go/support/render/hal"
	"fmt"
	"github.com/stellar/go/services/horizon/internal/httpx"
)

// TypeNames maps from operation type to the string used to represent that type
// in horizon's JSON responses
var OperationTypeNames = map[xdr.OperationType]string{
	xdr.OperationTypeCreateAccount:      "create_account",
	xdr.OperationTypePayment:            "payment",
	xdr.OperationTypePathPayment:        "path_payment",
	xdr.OperationTypeManageOffer:        "manage_offer",
	xdr.OperationTypeCreatePassiveOffer: "create_passive_offer",
	xdr.OperationTypeSetOptions:         "set_options",
	xdr.OperationTypeChangeTrust:        "change_trust",
	xdr.OperationTypeAllowTrust:         "allow_trust",
	xdr.OperationTypeAccountMerge:       "account_merge",
	xdr.OperationTypeInflation:          "inflation",
	xdr.OperationTypeManageData:         "manage_data",
}

// NewOperation creates a new operation resource, finding the appropriate type to use
// based upon the row's type.
func NewOperation(
	ctx context.Context,
	row history.Operation,
	ledger history.Ledger,
) (result hal.Pageable, err error) {

	base := BaseOperation{}
	PopulateBaseOperation(ctx, &base, row, ledger)

	switch row.Type {
	case xdr.OperationTypeCreateAccount:
		e := CreateAccount{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypePayment:
		e := Payment{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypePathPayment:
		e := PathPayment{}
		e.Payment.BaseOperation = base
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeManageOffer:
		e := ManageOffer{}
		e.CreatePassiveOffer.BaseOperation = base
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeCreatePassiveOffer:
		e := CreatePassiveOffer{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeSetOptions:
		e := SetOptions{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeChangeTrust:
		e := ChangeTrust{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeAllowTrust:
		e := AllowTrust{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeAccountMerge:
		e := AccountMerge{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeInflation:
		e := Inflation{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	case xdr.OperationTypeManageData:
		e := ManageData{BaseOperation: base}
		err = row.UnmarshalDetails(&e)
		result = e
	default:
		result = base
	}

	return
}

// Populate fills out this resource using `row` as the source.
func PopulateBaseOperation(
	ctx context.Context,
	dest *BaseOperation,
	row history.Operation,
	ledger history.Ledger,
) {
	dest.ID = fmt.Sprintf("%d", row.ID)
	dest.PT = row.PagingToken()
	dest.SourceAccount = row.SourceAccount
	populateOperationType(dest, row)
	dest.LedgerCloseTime = ledger.ClosedAt
	dest.TransactionHash = row.TransactionHash

	lb := hal.LinkBuilder{Base: httpx.BaseURL(ctx)}
	self := fmt.Sprintf("/operations/%d", row.ID)
	dest.Links.Self = lb.Link(self)
	dest.Links.Succeeds = lb.Linkf("/effects?order=desc&cursor=%s", dest.PT)
	dest.Links.Precedes = lb.Linkf("/effects?order=asc&cursor=%s", dest.PT)
	dest.Links.Transaction = lb.Linkf("/transactions/%s", row.TransactionHash)
	dest.Links.Effects = lb.Link(self, "effects")
}

func populateOperationType(dest *BaseOperation, row history.Operation) {
	var ok bool
	dest.TypeI = int32(row.Type)
	dest.Type, ok = OperationTypeNames[row.Type]

	if !ok {
		dest.Type = "unknown"
	}
}
