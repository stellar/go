package processors

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdio "io"
	"reflect"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// EffectProcessor process effects
type EffectProcessor struct {
	EffectsQ history.QEffects
}

func (p *EffectProcessor) loadAccountIDs(accountSet map[string]int64) error {
	addresses := make([]string, 0, len(accountSet))
	for address := range accountSet {
		addresses = append(addresses, address)
	}

	addressToID, err := p.EffectsQ.CreateExpAccounts(addresses)
	if err != nil {
		return errors.Wrap(err, "Could not create account ids")
	}

	for _, address := range addresses {
		id, ok := addressToID[address]
		if !ok {
			return errors.Errorf("no id found for account address %s", address)
		}

		accountSet[address] = id
	}

	return nil
}

func operationsEffects(transaction io.LedgerTransaction, sequence uint32) ([]effect, error) {
	effects := []effect{}

	for opi, op := range transaction.Envelope.Tx.Operations {
		operation := transactionOperationWrapper{
			index:          uint32(opi),
			transaction:    transaction,
			operation:      op,
			ledgerSequence: sequence,
		}

		p, err := operation.effects()
		if err != nil {
			return effects, errors.Wrapf(err, "reading operation %v effects", operation.ID())
		}
		effects = append(effects, p...)
	}

	return effects, nil
}

func (p *EffectProcessor) insertDBOperationsEffects(effects []effect, accountSet map[string]int64) error {
	batch := p.EffectsQ.NewEffectBatchInsertBuilder(maxBatchSize)

	for _, effect := range effects {
		accountID, found := accountSet[effect.address]

		if !found {
			return errors.Errorf("Error finding exp_history_account_id for address %v", effect.address)
		}

		var detailsJSON []byte
		detailsJSON, err := json.Marshal(effect.details)

		if err != nil {
			return errors.Wrapf(err, "Error marshaling details for operation effect %v", effect.operationID)
		}

		if err := batch.Add(
			accountID,
			effect.operationID,
			effect.order,
			effect.effectType,
			detailsJSON,
		); err != nil {
			return errors.Wrap(err, "could not insert operation effect in db")
		}
	}

	if err := batch.Exec(); err != nil {
		return errors.Wrap(err, "could not flush operation effects to db")
	}
	return nil
}

func (p *EffectProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer func() {
		// io.LedgerReader.Close() returns error if upgrade changes have not
		// been processed so it's worth checking the error.
		closeErr := r.Close()
		// Do not overwrite the previous error
		if err == nil {
			err = closeErr
		}
	}()
	defer w.Close()
	r.IgnoreUpgradeChanges()

	// Exit early if not ingesting into a DB
	if v := ctx.Value(IngestUpdateDatabase); v == nil {
		return nil
	}

	effects := []effect{}
	sequence := r.GetSequence()

	for {
		var transaction io.LedgerTransaction
		transaction, err = r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if transaction.Successful() {
			e, err2 := operationsEffects(transaction, sequence)

			if err2 != nil {
				return err2
			}

			effects = append(effects, e...)
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	if len(effects) > 0 {
		accountSet := map[string]int64{}

		for _, effect := range effects {
			accountSet[effect.address] = 0
		}

		if err = p.loadAccountIDs(accountSet); err != nil {
			return err
		}

		if err = p.insertDBOperationsEffects(effects, accountSet); err != nil {
			return err
		}
	}

	// use an older lookup sequence because the experimental ingestion system and the
	// legacy ingestion system might not be in sync
	if sequence > 10 {
		checkSequence := int32(sequence - 10)
		var valid bool
		valid, err = p.EffectsQ.CheckExpOperationEffects(checkSequence)
		if err != nil {
			log.WithField("sequence", checkSequence).WithError(err).
				Error("Could not compare effects for ledger")
			return nil
		}

		if !valid {
			log.WithField("sequence", checkSequence).
				Error("effects do not match")
		}
	}

	return nil
}

func (p *EffectProcessor) Name() string {
	return "EffectProcessor"
}

func (p *EffectProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &EffectProcessor{}

type effect struct {
	address     string
	operationID int64
	details     map[string]interface{}
	effectType  history.EffectType
	order       uint32
}

type effectsWrapper struct {
	effects   []effect
	operation *transactionOperationWrapper
}

func (e *effectsWrapper) add(address string, effectType history.EffectType, details map[string]interface{}) {
	e.effects = append(e.effects, effect{
		address:     address,
		operationID: e.operation.ID(),
		effectType:  effectType,
		order:       uint32(len(e.effects) + 1),
		details:     details,
	})
}

// Effects returns the operation effects
func (operation *transactionOperationWrapper) effects() (effects []effect, err error) {
	if !operation.transaction.Successful() {
		return []effect{}, err
	}

	op := operation.operation

	switch operation.OperationType() {
	case xdr.OperationTypeCreateAccount:
		effects = operation.accountCreatedEffects()
	case xdr.OperationTypePayment:
		effects = operation.paymentEffects()
	case xdr.OperationTypePathPaymentStrictReceive:
		effects = operation.pathPaymentStrictReceiveEffects()
	case xdr.OperationTypePathPaymentStrictSend:
		effects = operation.pathPaymentStrictSendEffects()
	case xdr.OperationTypeManageSellOffer:
		effects = operation.manageSellOfferEffects()
	case xdr.OperationTypeManageBuyOffer:
		effects = operation.manageBuyOfferEffects()
	case xdr.OperationTypeCreatePassiveSellOffer:
		effects = operation.createPassiveSellOfferEffect()
	case xdr.OperationTypeSetOptions:
		effects = operation.setOptionsEffects()
	case xdr.OperationTypeChangeTrust:
		effects = operation.changeTrustEffects()
	case xdr.OperationTypeAllowTrust:
		effects = operation.allowTrustEffects()
	case xdr.OperationTypeAccountMerge:
		effects = operation.accountMergeEffects()
	case xdr.OperationTypeInflation:
		effects = operation.inflationEffects()
	case xdr.OperationTypeManageData:
		effects = operation.manageDataEffects()
	case xdr.OperationTypeBumpSequence:
		effects = operation.bumpSequenceEffects()
	default:
		return effects, fmt.Errorf("Unknown operation type: %s", op.Body.Type)
	}

	return effects, err
}

func (operation *transactionOperationWrapper) accountCreatedEffects() []effect {
	op := operation.operation.Body.MustCreateAccountOp()
	effects := effectsWrapper{
		effects:   []effect{},
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

func (operation *transactionOperationWrapper) paymentEffects() []effect {
	op := operation.operation.Body.MustPaymentOp()
	effects := effectsWrapper{
		effects:   []effect{},
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

func (operation *transactionOperationWrapper) pathPaymentStrictReceiveEffects() []effect {
	op := operation.operation.Body.MustPathPaymentStrictReceiveOp()
	resultSuccess := operation.OperationResult().MustPathPaymentStrictReceiveResult().MustSuccess()
	source := operation.SourceAccount()

	details := map[string]interface{}{"amount": amount.String(op.DestAmount)}
	assetDetails(details, op.DestAsset, "")

	effects := effectsWrapper{
		effects:   []effect{},
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

func (operation *transactionOperationWrapper) pathPaymentStrictSendEffects() []effect {
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}
	source := operation.SourceAccount()
	op := operation.operation.Body.MustPathPaymentStrictSendOp()
	resultSuccess := operation.OperationResult().MustPathPaymentStrictSendResult().MustSuccess()
	result := operation.OperationResult().MustPathPaymentStrictSendResult()

	details := map[string]interface{}{"amount": amount.String(result.DestAmount())}
	assetDetails(details, op.DestAsset, "")
	effects.add(op.Destination.Address(), history.EffectAccountCredited, details)

	details = map[string]interface{}{"amount": amount.String(op.SendAmount)}
	assetDetails(details, op.SendAsset, "")
	effects.add(source.Address(), history.EffectAccountDebited, details)

	ingestTradeEffects(&effects, *source, resultSuccess.Offers)

	return effects.effects
}

func (operation *transactionOperationWrapper) manageSellOfferEffects() []effect {
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}
	result := operation.OperationResult().MustManageSellOfferResult().MustSuccess()
	ingestTradeEffects(&effects, *source, result.OffersClaimed)

	return effects.effects
}

func (operation *transactionOperationWrapper) manageBuyOfferEffects() []effect {
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}
	result := operation.OperationResult().MustManageBuyOfferResult().MustSuccess()
	ingestTradeEffects(&effects, *source, result.OffersClaimed)

	return effects.effects
}

func (operation *transactionOperationWrapper) createPassiveSellOfferEffect() []effect {
	result := operation.OperationResult()
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []effect{},
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

func (operation *transactionOperationWrapper) setOptionsEffects() []effect {
	source := operation.SourceAccount()
	op := operation.operation.Body.MustSetOptionsOp()

	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}

	if op.HomeDomain != nil {
		effects.add(source.Address(), history.EffectAccountHomeDomainUpdated,
			map[string]interface{}{
				"home_domain": string(*op.HomeDomain),
			},
		)
	}

	thresholdDetails := map[string]interface{}{}

	if op.LowThreshold != nil {
		thresholdDetails["low_threshold"] = *op.LowThreshold
	}

	if op.MedThreshold != nil {
		thresholdDetails["med_threshold"] = *op.MedThreshold
	}

	if op.HighThreshold != nil {
		thresholdDetails["high_threshold"] = *op.HighThreshold
	}

	if len(thresholdDetails) > 0 {
		effects.add(source.Address(), history.EffectAccountThresholdsUpdated, thresholdDetails)
	}

	flagDetails := map[string]interface{}{}
	effectFlagDetails(flagDetails, op.SetFlags, true)
	effectFlagDetails(flagDetails, op.ClearFlags, false)

	if len(flagDetails) > 0 {
		effects.add(source.Address(), history.EffectAccountFlagsUpdated, flagDetails)
	}

	if op.InflationDest != nil {
		effects.add(source.Address(), history.EffectAccountInflationDestinationUpdated,
			map[string]interface{}{
				"inflation_destination": op.InflationDest.Address(),
			},
		)
	}
	changes := operation.transaction.GetOperationChanges(operation.index)

	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeAccount {
			continue
		}

		beforeAccount := change.Pre.Data.MustAccount()
		afterAccount := change.Post.Data.MustAccount()

		before := beforeAccount.SignerSummary()
		after := afterAccount.SignerSummary()

		// if before and after are the same, the signers have not changed
		if reflect.DeepEqual(before, after) {
			continue
		}

		for addy := range before {
			weight, ok := after[addy]
			if !ok {
				effects.add(source.Address(), history.EffectSignerRemoved, map[string]interface{}{
					"public_key": addy,
				})
				continue
			}
			effects.add(source.Address(), history.EffectSignerUpdated, map[string]interface{}{
				"public_key": addy,
				"weight":     weight,
			})
		}

		// Add the "created" effects
		for addy, weight := range after {
			// if `addy` is in before, the previous for loop should have recorded
			// the update, so skip this key
			if _, ok := before[addy]; ok {
				continue
			}

			effects.add(source.Address(), history.EffectSignerCreated, map[string]interface{}{
				"public_key": addy,
				"weight":     weight,
			})
		}
	}

	return effects.effects
}

func (operation *transactionOperationWrapper) changeTrustEffects() []effect {
	source := operation.SourceAccount()

	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}

	op := operation.operation.Body.MustChangeTrustOp()
	details := map[string]interface{}{"limit": amount.String(op.Limit)}

	effect := history.EffectType(0)
	assetDetails(details, op.Line, "")

	changes := operation.transaction.GetOperationChanges(operation.index)

	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeTrustline {
			continue
		}

		switch {
		case change.Pre == nil && change.Post != nil:
			effect = history.EffectTrustlineCreated
		case change.Pre != nil && change.Post == nil:
			effect = history.EffectTrustlineRemoved
		case change.Pre != nil && change.Post != nil:
			effect = history.EffectTrustlineUpdated
		default:
			panic("Invalid change")
		}

		break
	}

	effects.add(source.Address(), effect, details)

	return effects.effects
}

func (operation *transactionOperationWrapper) allowTrustEffects() []effect {
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}
	op := operation.operation.Body.MustAllowTrustOp()
	asset := op.Asset.ToAsset(*source)
	details := map[string]interface{}{
		"trustor": op.Trustor.Address(),
	}
	assetDetails(details, asset, "")

	if op.Authorize {
		effects.add(source.Address(), history.EffectTrustlineAuthorized, details)
	} else {
		effects.add(source.Address(), history.EffectTrustlineDeauthorized, details)
	}

	return effects.effects
}

func (operation *transactionOperationWrapper) accountMergeEffects() []effect {
	source := operation.SourceAccount()
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}

	dest := operation.operation.Body.MustDestination()
	result := operation.OperationResult().MustAccountMergeResult()
	details := map[string]interface{}{
		"amount":     amount.String(result.MustSourceAccountBalance()),
		"asset_type": "native",
	}

	effects.add(source.Address(), history.EffectAccountDebited, details)
	effects.add(dest.Address(), history.EffectAccountCredited, details)
	effects.add(source.Address(), history.EffectAccountRemoved, map[string]interface{}{})

	return effects.effects
}

func (operation *transactionOperationWrapper) inflationEffects() []effect {
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}
	payouts := operation.OperationResult().MustInflationResult().MustPayouts()
	for _, payout := range payouts {
		effects.add(payout.Destination.Address(), history.EffectAccountCredited,
			map[string]interface{}{
				"amount":     amount.String(payout.Amount),
				"asset_type": "native",
			},
		)
	}

	return effects.effects
}

func (operation *transactionOperationWrapper) manageDataEffects() []effect {
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}
	source := operation.SourceAccount()
	op := operation.operation.Body.MustManageDataOp()
	details := map[string]interface{}{"name": op.DataName}
	effect := history.EffectType(0)
	changes := operation.transaction.GetOperationChanges(operation.index)

	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeData {
			continue
		}

		before := change.Pre
		after := change.Post

		if after != nil {
			raw := after.Data.MustData().DataValue
			details["value"] = base64.StdEncoding.EncodeToString(raw)
		}

		switch {
		case before == nil && after != nil:
			effect = history.EffectDataCreated
		case before != nil && after == nil:
			effect = history.EffectDataRemoved
		case before != nil && after != nil:
			effect = history.EffectDataUpdated
		default:
			panic("Invalid before-and-after state")
		}

		break
	}

	effects.add(source.Address(), effect, details)

	return effects.effects
}

func (operation *transactionOperationWrapper) bumpSequenceEffects() []effect {
	effects := effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}
	source := operation.SourceAccount()
	changes := operation.transaction.GetOperationChanges(operation.index)

	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeAccount {
			continue
		}

		before := change.Pre
		after := change.Post

		beforeAccount := before.Data.MustAccount()
		afterAccount := after.Data.MustAccount()

		if beforeAccount.SeqNum != afterAccount.SeqNum {
			details := map[string]interface{}{"new_seq": afterAccount.SeqNum}
			effects.add(source.Address(), history.EffectSequenceBumped, details)
		}

		break
	}

	return effects.effects
}

func effectFlagDetails(flagDetails map[string]interface{}, flagPtr *xdr.Uint32, setValue bool) {
	if flagPtr != nil {
		flags := xdr.AccountFlags(*flagPtr)

		if flags&xdr.AccountFlagsAuthRequiredFlag != 0 {
			flagDetails["auth_required_flag"] = setValue
		}
		if flags&xdr.AccountFlagsAuthRevocableFlag != 0 {
			flagDetails["auth_revocable_flag"] = setValue
		}
		if flags&xdr.AccountFlagsAuthImmutableFlag != 0 {
			flagDetails["auth_immutable_flag"] = setValue
		}
	}
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
