package processors

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// EffectProcessor process effects
type EffectProcessor struct {
	effects  []effect
	effectsQ history.QEffects
	sequence uint32
}

func NewEffectProcessor(effectsQ history.QEffects, sequence uint32) *EffectProcessor {
	return &EffectProcessor{
		effectsQ: effectsQ,
		sequence: sequence,
	}
}

func (p *EffectProcessor) loadAccountIDs(accountSet map[string]int64) error {
	addresses := make([]string, 0, len(accountSet))
	for address := range accountSet {
		addresses = append(addresses, address)
	}

	addressToID, err := p.effectsQ.CreateAccounts(addresses, maxBatchSize)
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

func operationsEffects(transaction ingest.LedgerTransaction, sequence uint32) ([]effect, error) {
	effects := []effect{}

	for opi, op := range transaction.Envelope.Operations() {
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
	batch := p.effectsQ.NewEffectBatchInsertBuilder(maxBatchSize)

	for _, effect := range effects {
		accountID, found := accountSet[effect.address]

		if !found {
			return errors.Errorf("Error finding history_account_id for address %v", effect.address)
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

func (p *EffectProcessor) ProcessTransaction(transaction ingest.LedgerTransaction) (err error) {
	// Failed transactions don't have operation effects
	if !transaction.Result.Successful() {
		return nil
	}

	var effectsForTx []effect
	effectsForTx, err = operationsEffects(transaction, p.sequence)
	if err != nil {
		return err
	}
	p.effects = append(p.effects, effectsForTx...)

	return nil
}

func (p *EffectProcessor) Commit() (err error) {
	if len(p.effects) > 0 {
		accountSet := map[string]int64{}

		for _, effect := range p.effects {
			accountSet[effect.address] = 0
		}

		if err = p.loadAccountIDs(accountSet); err != nil {
			return err
		}

		if err = p.insertDBOperationsEffects(p.effects, accountSet); err != nil {
			return err
		}
	}

	return err
}

type effect struct {
	address     string
	operationID int64
	details     map[string]interface{}
	effectType  history.EffectType
	order       uint32
}

// Effects returns the operation effects
func (operation *transactionOperationWrapper) effects() ([]effect, error) {
	if !operation.transaction.Result.Successful() {
		return []effect{}, nil
	}
	var (
		op  = operation.operation
		err error
	)

	changes, err := operation.transaction.GetOperationChanges(operation.index)
	if err != nil {
		return nil, err
	}

	wrapper := &effectsWrapper{
		effects:   []effect{},
		operation: operation,
	}

	switch operation.OperationType() {
	case xdr.OperationTypeCreateAccount:
		wrapper.addAccountCreatedEffects()
	case xdr.OperationTypePayment:
		wrapper.addPaymentEffects()
	case xdr.OperationTypePathPaymentStrictReceive:
		wrapper.pathPaymentStrictReceiveEffects()
	case xdr.OperationTypePathPaymentStrictSend:
		wrapper.addPathPaymentStrictSendEffects()
	case xdr.OperationTypeManageSellOffer:
		wrapper.addManageSellOfferEffects()
	case xdr.OperationTypeManageBuyOffer:
		wrapper.addManageBuyOfferEffects()
	case xdr.OperationTypeCreatePassiveSellOffer:
		wrapper.addCreatePassiveSellOfferEffect()
	case xdr.OperationTypeSetOptions:
		wrapper.addSetOptionsEffects()
	case xdr.OperationTypeChangeTrust:
		err = wrapper.addChangeTrustEffects()
	case xdr.OperationTypeAllowTrust:
		wrapper.addAllowTrustEffects()
	case xdr.OperationTypeAccountMerge:
		wrapper.addAccountMergeEffects()
	case xdr.OperationTypeInflation:
		wrapper.addInflationEffects()
	case xdr.OperationTypeManageData:
		err = wrapper.addManageDataEffects()
	case xdr.OperationTypeBumpSequence:
		err = wrapper.addBumpSequenceEffects()
	case xdr.OperationTypeCreateClaimableBalance:
		err = wrapper.addCreateClaimableBalanceEffects(changes)
	case xdr.OperationTypeClaimClaimableBalance:
		err = wrapper.addClaimClaimableBalanceEffects(changes)
	case xdr.OperationTypeBeginSponsoringFutureReserves, xdr.OperationTypeEndSponsoringFutureReserves, xdr.OperationTypeRevokeSponsorship:
	// The effects of these operations are obtained  indirectly from the ledger entries
	case xdr.OperationTypeClawback:
		err = wrapper.addClawbackEffects()
	case xdr.OperationTypeClawbackClaimableBalance:
		err = wrapper.addClawbackClaimableBalanceEffects(changes)
	case xdr.OperationTypeSetTrustLineFlags:
		err = wrapper.addSetTrustLineFlagsEffects()
	default:
		return nil, fmt.Errorf("Unknown operation type: %s", op.Body.Type)
	}
	if err != nil {
		return nil, err
	}
	for _, change := range changes {
		if err = wrapper.addLedgerEntrySponsorshipEffects(change); err != nil {
			return nil, err
		}
		wrapper.addSignerSponsorshipEffects(change)
	}

	return wrapper.effects, nil
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

var sponsoringEffectsTable = map[xdr.LedgerEntryType]struct {
	created, updated, removed history.EffectType
}{
	xdr.LedgerEntryTypeAccount: {
		created: history.EffectAccountSponsorshipCreated,
		updated: history.EffectAccountSponsorshipUpdated,
		removed: history.EffectAccountSponsorshipRemoved,
	},
	xdr.LedgerEntryTypeTrustline: {
		created: history.EffectTrustlineSponsorshipCreated,
		updated: history.EffectTrustlineSponsorshipUpdated,
		removed: history.EffectTrustlineSponsorshipRemoved,
	},
	xdr.LedgerEntryTypeData: {
		created: history.EffectDataSponsorshipCreated,
		updated: history.EffectDataSponsorshipUpdated,
		removed: history.EffectDataSponsorshipRemoved,
	},
	xdr.LedgerEntryTypeClaimableBalance: {
		created: history.EffectClaimableBalanceSponsorshipCreated,
		updated: history.EffectClaimableBalanceSponsorshipUpdated,
		removed: history.EffectClaimableBalanceSponsorshipRemoved,
	},

	// We intentionally don't have Sponsoring effects for Offer
	// entries because we don't generate creation effects for them.
}

func (e *effectsWrapper) addSignerSponsorshipEffects(change ingest.Change) {
	if change.Type != xdr.LedgerEntryTypeAccount {
		return
	}

	preSigners := map[string]xdr.AccountId{}
	postSigners := map[string]xdr.AccountId{}
	if change.Pre != nil {
		account := change.Pre.Data.MustAccount()
		preSigners = account.SponsorPerSigner()
	}
	if change.Post != nil {
		account := change.Post.Data.MustAccount()
		postSigners = account.SponsorPerSigner()
	}

	var all []string
	for signer := range preSigners {
		all = append(all, signer)
	}
	for signer := range postSigners {
		if _, ok := preSigners[signer]; ok {
			continue
		}
		all = append(all, signer)
	}
	sort.Strings(all)

	for _, signer := range all {
		pre, foundPre := preSigners[signer]
		post, foundPost := postSigners[signer]
		details := map[string]interface{}{}

		switch {
		case !foundPre && !foundPost:
			continue
		case !foundPre && foundPost:
			details["sponsor"] = post.Address()
			details["signer"] = signer
			srcAccount := change.Post.Data.MustAccount().AccountId
			e.add(srcAccount.Address(), history.EffectSignerSponsorshipCreated, details)
		case !foundPost && foundPre:
			details["former_sponsor"] = pre.Address()
			details["signer"] = signer
			srcAccount := change.Pre.Data.MustAccount().AccountId
			e.add(srcAccount.Address(), history.EffectSignerSponsorshipRemoved, details)
		case foundPre && foundPost:
			formerSponsor := pre.Address()
			newSponsor := post.Address()
			if formerSponsor == newSponsor {
				continue
			}

			details["former_sponsor"] = formerSponsor
			details["new_sponsor"] = newSponsor
			details["signer"] = signer
			srcAccount := change.Post.Data.MustAccount().AccountId
			e.add(srcAccount.Address(), history.EffectSignerSponsorshipUpdated, details)
		}
	}
}

func (e *effectsWrapper) addLedgerEntrySponsorshipEffects(change ingest.Change) error {
	effectsForEntryType, found := sponsoringEffectsTable[change.Type]
	if !found {
		return nil
	}

	details := map[string]interface{}{}
	var effectType history.EffectType

	switch {
	case (change.Pre == nil || change.Pre.SponsoringID() == nil) &&
		(change.Post != nil && change.Post.SponsoringID() != nil):
		effectType = effectsForEntryType.created
		details["sponsor"] = (*change.Post.SponsoringID()).Address()
	case (change.Pre != nil && change.Pre.SponsoringID() != nil) &&
		(change.Post == nil || change.Post.SponsoringID() == nil):
		effectType = effectsForEntryType.removed
		details["former_sponsor"] = (*change.Pre.SponsoringID()).Address()
	case (change.Pre != nil && change.Pre.SponsoringID() != nil) &&
		(change.Post != nil && change.Post.SponsoringID() != nil):
		preSponsor := (*change.Pre.SponsoringID()).Address()
		postSponsor := (*change.Post.SponsoringID()).Address()
		if preSponsor == postSponsor {
			return nil
		}
		effectType = effectsForEntryType.updated
		details["new_sponsor"] = postSponsor
		details["former_sponsor"] = preSponsor
	default:
		return nil
	}

	var accountAddress string
	var data xdr.LedgerEntryData
	if change.Post != nil {
		data = change.Post.Data
	} else {
		data = change.Pre.Data
	}

	switch change.Type {
	case xdr.LedgerEntryTypeAccount:
		aid := data.MustAccount().AccountId
		accountAddress = aid.Address()
	case xdr.LedgerEntryTypeTrustline:
		aid := data.MustTrustLine().AccountId
		accountAddress = aid.Address()
		details["asset"] = data.MustTrustLine().Asset.StringCanonical()
	case xdr.LedgerEntryTypeData:
		accountAddress = e.operation.SourceAccount().Address()
		details["data_name"] = data.MustData().DataName
	case xdr.LedgerEntryTypeClaimableBalance:
		accountAddress = e.operation.SourceAccount().Address()
		var err error
		details["balance_id"], err = xdr.MarshalHex(data.MustClaimableBalance().BalanceId)
		if err != nil {
			return errors.Wrapf(err, "Invalid balanceId in change from op %d", e.operation.index)
		}
	default:
		return errors.Errorf("invalid sponsorship ledger entry type %v", change.Type.String())
	}

	e.add(accountAddress, effectType, details)
	return nil
}

func (e *effectsWrapper) addAccountCreatedEffects() {
	op := e.operation.operation.Body.MustCreateAccountOp()

	e.add(
		op.Destination.Address(),
		history.EffectAccountCreated,
		map[string]interface{}{
			"starting_balance": amount.String(op.StartingBalance),
		},
	)
	e.add(
		e.operation.SourceAccount().Address(),
		history.EffectAccountDebited,
		map[string]interface{}{
			"asset_type": "native",
			"amount":     amount.String(op.StartingBalance),
		},
	)
	e.add(
		op.Destination.Address(),
		history.EffectSignerCreated,
		map[string]interface{}{
			"public_key": op.Destination.Address(),
			"weight":     keypair.DefaultSignerWeight,
		},
	)
}

func (e *effectsWrapper) addPaymentEffects() {
	op := e.operation.operation.Body.MustPaymentOp()

	details := map[string]interface{}{"amount": amount.String(op.Amount)}
	addAssetDetails(details, op.Asset, "")

	aid := op.Destination.ToAccountId()
	e.add(
		aid.Address(),
		history.EffectAccountCredited,
		details,
	)
	e.add(
		e.operation.SourceAccount().Address(),
		history.EffectAccountDebited,
		details,
	)
}

func (e *effectsWrapper) pathPaymentStrictReceiveEffects() {
	op := e.operation.operation.Body.MustPathPaymentStrictReceiveOp()
	resultSuccess := e.operation.OperationResult().MustPathPaymentStrictReceiveResult().MustSuccess()
	source := e.operation.SourceAccount()

	details := map[string]interface{}{"amount": amount.String(op.DestAmount)}
	addAssetDetails(details, op.DestAsset, "")

	aid := op.Destination.ToAccountId()
	e.add(
		aid.Address(),
		history.EffectAccountCredited,
		details,
	)

	result := e.operation.OperationResult().MustPathPaymentStrictReceiveResult()
	details = map[string]interface{}{"amount": amount.String(result.SendAmount())}
	addAssetDetails(details, op.SendAsset, "")

	e.add(
		source.Address(),
		history.EffectAccountDebited,
		details,
	)

	e.addIngestTradeEffects(*source, resultSuccess.Offers)
}

func (e *effectsWrapper) addPathPaymentStrictSendEffects() {
	source := e.operation.SourceAccount()
	op := e.operation.operation.Body.MustPathPaymentStrictSendOp()
	resultSuccess := e.operation.OperationResult().MustPathPaymentStrictSendResult().MustSuccess()
	result := e.operation.OperationResult().MustPathPaymentStrictSendResult()

	details := map[string]interface{}{"amount": amount.String(result.DestAmount())}
	addAssetDetails(details, op.DestAsset, "")
	aid := op.Destination.ToAccountId()
	e.add(aid.Address(), history.EffectAccountCredited, details)

	details = map[string]interface{}{"amount": amount.String(op.SendAmount)}
	addAssetDetails(details, op.SendAsset, "")
	e.add(source.Address(), history.EffectAccountDebited, details)

	e.addIngestTradeEffects(*source, resultSuccess.Offers)
}

func (e *effectsWrapper) addManageSellOfferEffects() {
	source := e.operation.SourceAccount()
	result := e.operation.OperationResult().MustManageSellOfferResult().MustSuccess()
	e.addIngestTradeEffects(*source, result.OffersClaimed)
}

func (e *effectsWrapper) addManageBuyOfferEffects() {
	source := e.operation.SourceAccount()
	result := e.operation.OperationResult().MustManageBuyOfferResult().MustSuccess()
	e.addIngestTradeEffects(*source, result.OffersClaimed)
}

func (e *effectsWrapper) addCreatePassiveSellOfferEffect() {
	result := e.operation.OperationResult()
	source := e.operation.SourceAccount()

	var claims []xdr.ClaimOfferAtom

	// KNOWN ISSUE:  stellar-core creates results for CreatePassiveOffer operations
	// with the wrong result arm set.
	if result.Type == xdr.OperationTypeManageSellOffer {
		claims = result.MustManageSellOfferResult().MustSuccess().OffersClaimed
	} else {
		claims = result.MustCreatePassiveSellOfferResult().MustSuccess().OffersClaimed
	}

	e.addIngestTradeEffects(*source, claims)
}

func (e *effectsWrapper) addSetOptionsEffects() error {
	source := e.operation.SourceAccount()
	op := e.operation.operation.Body.MustSetOptionsOp()

	if op.HomeDomain != nil {
		e.add(source.Address(), history.EffectAccountHomeDomainUpdated,
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
		e.add(source.Address(), history.EffectAccountThresholdsUpdated, thresholdDetails)
	}

	flagDetails := map[string]interface{}{}
	if op.SetFlags != nil {
		setAuthFlagDetails(flagDetails, xdr.AccountFlags(*op.SetFlags), true)
	}
	if op.ClearFlags != nil {
		setAuthFlagDetails(flagDetails, xdr.AccountFlags(*op.ClearFlags), false)
	}

	if len(flagDetails) > 0 {
		e.add(source.Address(), history.EffectAccountFlagsUpdated, flagDetails)
	}

	if op.InflationDest != nil {
		e.add(source.Address(), history.EffectAccountInflationDestinationUpdated,
			map[string]interface{}{
				"inflation_destination": op.InflationDest.Address(),
			},
		)
	}
	changes, err := e.operation.transaction.GetOperationChanges(e.operation.index)
	if err != nil {
		return err
	}

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

		beforeSortedSigners := []string{}
		for signer := range before {
			beforeSortedSigners = append(beforeSortedSigners, signer)
		}
		sort.Strings(beforeSortedSigners)

		for _, addy := range beforeSortedSigners {
			weight, ok := after[addy]
			if !ok {
				e.add(source.Address(), history.EffectSignerRemoved, map[string]interface{}{
					"public_key": addy,
				})
				continue
			}

			if weight != before[addy] {
				e.add(source.Address(), history.EffectSignerUpdated, map[string]interface{}{
					"public_key": addy,
					"weight":     weight,
				})
			}
		}

		afterSortedSigners := []string{}
		for signer := range after {
			afterSortedSigners = append(afterSortedSigners, signer)
		}
		sort.Strings(afterSortedSigners)

		// Add the "created" effects
		for _, addy := range afterSortedSigners {
			weight := after[addy]
			// if `addy` is in before, the previous for loop should have recorded
			// the update, so skip this key
			if _, ok := before[addy]; ok {
				continue
			}

			e.add(source.Address(), history.EffectSignerCreated, map[string]interface{}{
				"public_key": addy,
				"weight":     weight,
			})
		}
	}
	return nil
}

func (e *effectsWrapper) addChangeTrustEffects() error {
	source := e.operation.SourceAccount()

	op := e.operation.operation.Body.MustChangeTrustOp()
	changes, err := e.operation.transaction.GetOperationChanges(e.operation.index)
	if err != nil {
		return err
	}

	// NOTE:  when an account trusts itself, the transaction is successful but
	// no ledger entries are actually modified.
	if len(changes) > 0 {
		details := map[string]interface{}{"limit": amount.String(op.Limit)}
		effect := history.EffectType(0)
		addAssetDetails(details, op.Line, "")

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

		e.add(source.Address(), effect, details)
	}
	return nil
}

func (e *effectsWrapper) addAllowTrustEffects() {
	source := e.operation.SourceAccount()
	op := e.operation.operation.Body.MustAllowTrustOp()
	asset := op.Asset.ToAsset(*source)
	details := map[string]interface{}{
		"trustor": op.Trustor.Address(),
	}
	addAssetDetails(details, asset, "")

	switch {
	case xdr.TrustLineFlags(op.Authorize).IsAuthorized():
		e.add(source.Address(), history.EffectTrustlineAuthorized, details)
		// Forward compatibility
		setFlags := xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag)
		e.addTrustLineFlagsEffect(source, &op.Trustor, asset, &setFlags, nil)
	case xdr.TrustLineFlags(op.Authorize).IsAuthorizedToMaintainLiabilitiesFlag():
		e.add(
			source.Address(),
			history.EffectTrustlineAuthorizedToMaintainLiabilities,
			details,
		)
		// Forward compatibility
		setFlags := xdr.Uint32(xdr.TrustLineFlagsAuthorizedToMaintainLiabilitiesFlag)
		e.addTrustLineFlagsEffect(source, &op.Trustor, asset, &setFlags, nil)
	default:
		e.add(source.Address(), history.EffectTrustlineDeauthorized, details)
		// Forward compatibility, show both as cleared
		clearFlags := xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag | xdr.TrustLineFlagsAuthorizedToMaintainLiabilitiesFlag)
		e.addTrustLineFlagsEffect(source, &op.Trustor, asset, nil, &clearFlags)
	}
}

func (e *effectsWrapper) addAccountMergeEffects() {
	source := e.operation.SourceAccount()

	dest := e.operation.operation.Body.MustDestination()
	result := e.operation.OperationResult().MustAccountMergeResult()
	details := map[string]interface{}{
		"amount":     amount.String(result.MustSourceAccountBalance()),
		"asset_type": "native",
	}

	e.add(source.Address(), history.EffectAccountDebited, details)
	aid := dest.ToAccountId()
	e.add(aid.Address(), history.EffectAccountCredited, details)
	e.add(source.Address(), history.EffectAccountRemoved, map[string]interface{}{})
}

func (e *effectsWrapper) addInflationEffects() {
	payouts := e.operation.OperationResult().MustInflationResult().MustPayouts()
	for _, payout := range payouts {
		e.add(payout.Destination.Address(), history.EffectAccountCredited,
			map[string]interface{}{
				"amount":     amount.String(payout.Amount),
				"asset_type": "native",
			},
		)
	}
}

func (e *effectsWrapper) addManageDataEffects() error {
	source := e.operation.SourceAccount()
	op := e.operation.operation.Body.MustManageDataOp()
	details := map[string]interface{}{"name": op.DataName}
	effect := history.EffectType(0)
	changes, err := e.operation.transaction.GetOperationChanges(e.operation.index)
	if err != nil {
		return err
	}

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

	e.add(source.Address(), effect, details)
	return nil
}

func (e *effectsWrapper) addBumpSequenceEffects() error {
	source := e.operation.SourceAccount()
	changes, err := e.operation.transaction.GetOperationChanges(e.operation.index)
	if err != nil {
		return err
	}

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
			e.add(source.Address(), history.EffectSequenceBumped, details)
		}
		break
	}

	return nil
}

func setClaimableBalanceFlagDetails(details map[string]interface{}, flags xdr.ClaimableBalanceFlags) {
	if flags.IsClawbackEnabled() {
		details["claimable_balance_clawback_enabled_flag"] = true
		return
	}
}

func (e *effectsWrapper) addCreateClaimableBalanceEffects(changes []ingest.Change) error {
	op := e.operation.operation.Body.MustCreateClaimableBalanceOp()

	result := e.operation.OperationResult().MustCreateClaimableBalanceResult()
	balanceID, err := xdr.MarshalHex(result.BalanceId)
	if err != nil {
		return errors.Wrapf(err, "Invalid balanceId in op: %d", e.operation.index)
	}

	details := map[string]interface{}{
		"balance_id": balanceID,
		"amount":     amount.String(op.Amount),
		"asset":      op.Asset.StringCanonical(),
	}
	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeClaimableBalance || change.Post == nil {
			continue
		}
		setClaimableBalanceFlagDetails(details, change.Post.Data.ClaimableBalance.Flags())
		break
	}
	e.add(
		e.operation.SourceAccount().Address(),
		history.EffectClaimableBalanceCreated,
		details,
	)

	for _, c := range op.Claimants {
		cv0 := c.MustV0()
		e.add(
			cv0.Destination.Address(),
			history.EffectClaimableBalanceClaimantCreated,
			map[string]interface{}{
				"balance_id": balanceID,
				"amount":     amount.String(op.Amount),
				"predicate":  cv0.Predicate,
				"asset":      op.Asset.StringCanonical(),
			},
		)
	}

	details = map[string]interface{}{
		"amount": amount.String(op.Amount),
	}
	addAssetDetails(details, op.Asset, "")
	e.add(
		e.operation.SourceAccount().Address(),
		history.EffectAccountDebited,
		details,
	)

	return nil
}

func (e *effectsWrapper) addClaimClaimableBalanceEffects(changes []ingest.Change) error {
	op := e.operation.operation.Body.MustClaimClaimableBalanceOp()

	balanceID, err := xdr.MarshalHex(op.BalanceId)
	if err != nil {
		return fmt.Errorf("Invalid balanceId in op: %d", e.operation.index)
	}

	var cBalance xdr.ClaimableBalanceEntry
	found := false
	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeClaimableBalance {
			continue
		}

		if change.Pre != nil && change.Post == nil {
			cBalance = change.Pre.Data.MustClaimableBalance()
			preBalanceID, err := xdr.MarshalHex(cBalance.BalanceId)
			if err != nil {
				return fmt.Errorf("Invalid balanceId in meta changes for op: %d", e.operation.index)
			}

			if preBalanceID == balanceID {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("Change not found for balanceId : %s", balanceID)
	}

	details := map[string]interface{}{
		"amount":     amount.String(cBalance.Amount),
		"balance_id": balanceID,
		"asset":      cBalance.Asset.StringCanonical(),
	}
	setClaimableBalanceFlagDetails(details, cBalance.Flags())
	e.add(
		e.operation.SourceAccount().Address(),
		history.EffectClaimableBalanceClaimed,
		details,
	)

	details = map[string]interface{}{
		"amount": amount.String(cBalance.Amount),
	}
	addAssetDetails(details, cBalance.Asset, "")
	e.add(
		e.operation.SourceAccount().Address(),
		history.EffectAccountCredited,
		details,
	)

	return nil
}

func (e *effectsWrapper) addIngestTradeEffects(buyer xdr.AccountId, claims []xdr.ClaimOfferAtom) {
	for _, claim := range claims {
		if claim.AmountSold == 0 && claim.AmountBought == 0 {
			continue
		}

		seller := claim.SellerId
		bd, sd := tradeDetails(buyer, seller, claim)

		e.add(
			buyer.Address(),
			history.EffectTrade,
			bd,
		)

		e.add(
			seller.Address(),
			history.EffectTrade,
			sd,
		)
	}
}

func (e *effectsWrapper) addClawbackEffects() error {
	op := e.operation.operation.Body.MustClawbackOp()
	details := map[string]interface{}{
		"amount": amount.String(op.Amount),
	}
	source := e.operation.SourceAccount()
	addAssetDetails(details, op.Asset, "")

	// The funds will be burned, but even with that, we generated an account credited effect
	e.add(
		source.Address(),
		history.EffectAccountCredited,
		details,
	)

	from := op.From.ToAccountId()
	e.add(
		from.Address(),
		history.EffectAccountDebited,
		details,
	)

	return nil
}

func (e *effectsWrapper) addClawbackClaimableBalanceEffects(changes []ingest.Change) error {
	op := e.operation.operation.Body.MustClawbackClaimableBalanceOp()
	balanceId, err := xdr.MarshalHex(op.BalanceId)
	if err != nil {
		return errors.Wrapf(err, "Invalid balanceId in op %d", e.operation.index)
	}
	details := map[string]interface{}{
		"balance_id": balanceId,
	}
	e.add(
		e.operation.SourceAccount().Address(),
		history.EffectClaimableBalanceClawedBack,
		details,
	)

	// Generate the account credited effect (although the funds will be burned) for the asset issuer
	for _, c := range changes {
		if c.Type == xdr.LedgerEntryTypeClaimableBalance && c.Post == nil && c.Pre != nil {
			cb := c.Pre.Data.ClaimableBalance
			details = map[string]interface{}{"amount": amount.String(cb.Amount)}
			addAssetDetails(details, cb.Asset, "")
			e.add(
				e.operation.SourceAccount().Address(),
				history.EffectAccountCredited,
				details,
			)
			break
		}
	}

	return nil
}

func (e *effectsWrapper) addSetTrustLineFlagsEffects() error {
	source := e.operation.SourceAccount()
	op := e.operation.operation.Body.MustSetTrustLineFlagsOp()
	e.addTrustLineFlagsEffect(source, &op.Trustor, op.Asset, &op.SetFlags, &op.ClearFlags)
	return nil
}

func (e *effectsWrapper) addTrustLineFlagsEffect(
	account *xdr.AccountId,
	trustor *xdr.AccountId,
	asset xdr.Asset,
	setFlags *xdr.Uint32,
	clearFlags *xdr.Uint32) {
	details := map[string]interface{}{
		"trustor": trustor.Address(),
	}
	addAssetDetails(details, asset, "")

	var flagDetailsAdded bool
	if setFlags != nil {
		setTrustLineFlagDetails(details, xdr.TrustLineFlags(*setFlags), true)
		flagDetailsAdded = true
	}
	if clearFlags != nil {
		setTrustLineFlagDetails(details, xdr.TrustLineFlags(*clearFlags), false)
		flagDetailsAdded = true
	}

	if flagDetailsAdded {
		e.add(account.Address(), history.EffectTrustlineFlagsUpdated, details)
	}
}

func setTrustLineFlagDetails(flagDetails map[string]interface{}, flags xdr.TrustLineFlags, setValue bool) {
	if flags.IsAuthorized() {
		flagDetails["authorized_flag"] = setValue
	}
	if flags.IsAuthorizedToMaintainLiabilitiesFlag() {
		flagDetails["authorized_to_maintain_liabilites"] = setValue
	}
	if flags.IsClawbackEnabledFlag() {
		flagDetails["clawback_enabled_flag"] = setValue
	}
}

func setAuthFlagDetails(flagDetails map[string]interface{}, flags xdr.AccountFlags, setValue bool) {
	if flags.IsAuthRequired() {
		flagDetails["auth_required_flag"] = setValue
	}
	if flags.IsAuthRevocable() {
		flagDetails["auth_revocable_flag"] = setValue
	}
	if flags.IsAuthImmutable() {
		flagDetails["auth_immutable_flag"] = setValue
	}
	if flags.IsAuthClawbackEnabled() {
		flagDetails["auth_clawback_enabled_flag"] = setValue
	}
}

func tradeDetails(buyer, seller xdr.AccountId, claim xdr.ClaimOfferAtom) (bd map[string]interface{}, sd map[string]interface{}) {
	bd = map[string]interface{}{
		"offer_id":      claim.OfferId,
		"seller":        seller.Address(),
		"bought_amount": amount.String(claim.AmountSold),
		"sold_amount":   amount.String(claim.AmountBought),
	}
	addAssetDetails(bd, claim.AssetSold, "bought_")
	addAssetDetails(bd, claim.AssetBought, "sold_")

	sd = map[string]interface{}{
		"offer_id":      claim.OfferId,
		"seller":        buyer.Address(),
		"bought_amount": amount.String(claim.AmountBought),
		"sold_amount":   amount.String(claim.AmountSold),
	}
	addAssetDetails(sd, claim.AssetBought, "bought_")
	addAssetDetails(sd, claim.AssetSold, "sold_")

	return
}
