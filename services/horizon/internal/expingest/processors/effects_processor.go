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
	effects   []map[string]interface{}
	operation *transactionOperationWrapper
	order     uint32
}

func (e *effectsWrapper) add(address string, effectType history.EffectType, details map[string]interface{}) {
	e.order++
	e.effects = append(e.effects, map[string]interface{}{
		"address":     address,
		"operationID": e.operation.ID(),
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
		effects = operation.pathPaymentStrictReceiveEffects()
	case xdr.OperationTypePathPaymentStrictSend:
		// TBD
	case xdr.OperationTypeManageSellOffer:
		effects = operation.manageSellOfferEffects()
	case xdr.OperationTypeManageBuyOffer:
		effects = operation.manageBuyOfferEffects()
	case xdr.OperationTypeCreatePassiveSellOffer:
		effects = operation.createPassiveSellOfferEffect()
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
		effects:   []map[string]interface{}{},
		operation: operation,
	}

	effects.add(
		op.Destination.Address(),
		history.EffectAccountCreated,
		map[string]interface{}{
			"starting_balance": amount.String(op.StartingBalance),
		},
	)
	effects.add(
		operation.SourceAccount().Address(),
		history.EffectAccountDebited,
		map[string]interface{}{
			"asset_type": "native",
			"amount":     amount.String(op.StartingBalance),
		},
	)
	effects.add(
		op.Destination.Address(),
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
		effects:   []map[string]interface{}{},
		operation: operation,
	}

	details := map[string]interface{}{"amount": amount.String(op.Amount)}
	assetDetails(details, op.Asset, "")

	effects.add(
		op.Destination.Address(),
		history.EffectAccountCredited,
		details,
	)
	effects.add(
		operation.SourceAccount().Address(),
		history.EffectAccountDebited,
		details,
	)

	return effects.effects
}

func (operation *transactionOperationWrapper) pathPaymentStrictReceiveEffects() []map[string]interface{} {
	op := operation.operation.Body.MustPathPaymentStrictReceiveOp()
	resultSuccess := operation.OperationResult().MustPathPaymentStrictReceiveResult().MustSuccess()
	source := operation.SourceAccount()

	details := map[string]interface{}{"amount": amount.String(op.DestAmount)}
	assetDetails(details, op.DestAsset, "")

	effects := effectsWrapper{
		effects:   []map[string]interface{}{},
		operation: operation,
	}

	effects.add(
		op.Destination.Address(),
		history.EffectAccountCredited,
		details,
	)

	result := operation.OperationResult().MustPathPaymentStrictReceiveResult()
	details = map[string]interface{}{"amount": amount.String(result.SendAmount())}
	assetDetails(details, op.SendAsset, "")

	effects.add(
		source.Address(),
		history.EffectAccountDebited,
		details,
	)

	ingestTradeEffects(&effects, *source, resultSuccess.Offers)

	return effects.effects
}

func (operation *transactionOperationWrapper) manageSellOfferEffects() []map[string]interface{} {
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []map[string]interface{}{},
		operation: operation,
	}
	result := operation.OperationResult().MustManageSellOfferResult().MustSuccess()
	ingestTradeEffects(&effects, *source, result.OffersClaimed)

	return effects.effects
}

func (operation *transactionOperationWrapper) manageBuyOfferEffects() []map[string]interface{} {
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []map[string]interface{}{},
		operation: operation,
	}
	result := operation.OperationResult().MustManageBuyOfferResult().MustSuccess()
	ingestTradeEffects(&effects, *source, result.OffersClaimed)

	return effects.effects
}

func (operation *transactionOperationWrapper) createPassiveSellOfferEffect() []map[string]interface{} {
	result := operation.OperationResult()
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []map[string]interface{}{},
		operation: operation,
	}

	var claims []xdr.ClaimOfferAtom

	// KNOWN ISSUE:  stellar-core creates results for CreatePassiveOffer operations
	// with the wrong result arm set.
	if result.Type == xdr.OperationTypeManageSellOffer {
		claims = result.MustManageSellOfferResult().MustSuccess().OffersClaimed
	} else {
		claims = result.MustCreatePassiveSellOfferResult().MustSuccess().OffersClaimed
	}

	ingestTradeEffects(&effects, *source, claims)

	return effects.effects
}

func ingestTradeEffects(effects *effectsWrapper, buyer xdr.AccountId, claims []xdr.ClaimOfferAtom) {
	for _, claim := range claims {
		if claim.AmountSold == 0 && claim.AmountBought == 0 {
			continue
		}

		seller := claim.SellerId
		bd, sd := tradeDetails(buyer, seller, claim)

		effects.add(
			buyer.Address(),
			history.EffectTrade,
			bd,
		)

		effects.add(
			seller.Address(),
			history.EffectTrade,
			sd,
		)
	}
}

func tradeDetails(buyer, seller xdr.AccountId, claim xdr.ClaimOfferAtom) (bd map[string]interface{}, sd map[string]interface{}) {
	bd = map[string]interface{}{
		"offer_id":      claim.OfferId,
		"seller":        seller.Address(),
		"bought_amount": amount.String(claim.AmountSold),
		"sold_amount":   amount.String(claim.AmountBought),
	}
	assetDetails(bd, claim.AssetSold, "bought_")
	assetDetails(bd, claim.AssetBought, "sold_")

	sd = map[string]interface{}{
		"offer_id":      claim.OfferId,
		"seller":        buyer.Address(),
		"bought_amount": amount.String(claim.AmountBought),
		"sold_amount":   amount.String(claim.AmountSold),
	}
	assetDetails(sd, claim.AssetBought, "bought_")
	assetDetails(sd, claim.AssetSold, "sold_")

	return
}
