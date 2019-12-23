package processors

import (
	"fmt"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

// EffectProcessor process effects
type EffectProcessor struct {
	EffectsQ history.QEffects
}

// Effects returns the operation effects
func (operation *transactionOperationWrapper) Effects() (effects []map[string]interface{}, err error) {
	op := operation.operation

	switch operation.OperationType() {
	case xdr.OperationTypeCreateAccount:
		effects = operation.accountCreatedEffects()
	case xdr.OperationTypePayment:
		// TBD
	case xdr.OperationTypePathPaymentStrictReceive:
		// TBD
	case xdr.OperationTypePathPaymentStrictSend:
		// TBD
	case xdr.OperationTypeManageBuyOffer:
		// TBD
	case xdr.OperationTypeManageSellOffer:
		// TBD
	case xdr.OperationTypeCreatePassiveSellOffer:
		// TBD
	case xdr.OperationTypeSetOptions:
		// TBD
	case xdr.OperationTypeChangeTrust:
		// TBD
	case xdr.OperationTypeAllowTrust:
		// TBD
	case xdr.OperationTypeAccountMerge:
		// TBD
	case xdr.OperationTypeInflation:
		// TBD
	case xdr.OperationTypeManageData:
		// TBD
	case xdr.OperationTypeBumpSequence:
		// TBD
	default:
		return effects, fmt.Errorf("Unknown operation type: %s", op.Body.Type)
	}

	return effects, err
}

func (operation *transactionOperationWrapper) accountCreatedEffects() []map[string]interface{} {
	op := operation.operation.Body.MustCreateAccountOp()
	return []map[string]interface{}{
		buildEffectRow(
			op.Destination.Address(),
			operation.ID(),
			history.EffectAccountCreated,
			1,
			map[string]interface{}{
				"starting_balance": amount.String(op.StartingBalance),
			},
		),
		buildEffectRow(
			operation.SourceAccount().Address(),
			operation.ID(),
			history.EffectAccountDebited,
			2,
			map[string]interface{}{
				"asset_type": "native",
				"amount":     amount.String(op.StartingBalance),
			},
		),
		buildEffectRow(
			op.Destination.Address(),
			operation.ID(),
			history.EffectSignerCreated,
			3,
			map[string]interface{}{
				"public_key": op.Destination.Address(),
				"weight":     keypair.DefaultSignerWeight,
			},
		),
	}
}

func buildEffectRow(address string, operationID int64, effectType history.EffectType, order uint32, details map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"address":     address,
		"operationID": operationID,
		"effectType":  effectType,
		"order":       order,
		"details":     details,
	}
}
