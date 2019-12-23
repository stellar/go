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

type effectsWrapper struct {
	effects []map[string]interface{}
	order   uint32
}

func (e *effectsWrapper) add(address string, operationID int64, effectType history.EffectType, details map[string]interface{}) {
	e.order++
	e.effects = append(e.effects, map[string]interface{}{
		"address":     address,
		"operationID": operationID,
		"effectType":  effectType,
		"order":       e.order,
		"details":     details,
	})
}

// Effects returns the operation effects
func (operation *transactionOperationWrapper) Effects() (effects []map[string]interface{}, err error) {
	op := operation.operation

	switch operation.OperationType() {
	case xdr.OperationTypeCreateAccount:
		effects = operation.accountCreatedEffects()
	case xdr.OperationTypePayment:
		effects = operation.paymentEffects()
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
	effects := effectsWrapper{
		effects: []map[string]interface{}{},
	}

	effects.add(
		op.Destination.Address(),
		operation.ID(),
		history.EffectAccountCreated,
		map[string]interface{}{
			"starting_balance": amount.String(op.StartingBalance),
		},
	)
	effects.add(
		operation.SourceAccount().Address(),
		operation.ID(),
		history.EffectAccountDebited,
		map[string]interface{}{
			"asset_type": "native",
			"amount":     amount.String(op.StartingBalance),
		},
	)
	effects.add(
		op.Destination.Address(),
		operation.ID(),
		history.EffectSignerCreated,
		map[string]interface{}{
			"public_key": op.Destination.Address(),
			"weight":     keypair.DefaultSignerWeight,
		},
	)

	return effects.effects
}

func (operation *transactionOperationWrapper) paymentEffects() []map[string]interface{} {
	op := operation.operation.Body.MustPaymentOp()
	effects := effectsWrapper{
		effects: []map[string]interface{}{},
	}

	details := map[string]interface{}{"amount": amount.String(op.Amount)}
	assetDetails(details, op.Asset, "")

	effects.add(
		op.Destination.Address(),
		operation.ID(),
		history.EffectAccountCredited,
		details,
	)
	effects.add(
		operation.SourceAccount().Address(),
		operation.ID(),
		history.EffectAccountDebited,
		details,
	)

	return effects.effects
}
