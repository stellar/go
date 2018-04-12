package ingest

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"time"

	"github.com/stellar/go/clients/stellarcore"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/meta"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/participants"
	"github.com/stellar/go/support/errors"
	sTime "github.com/stellar/go/support/time"
	"github.com/stellar/go/xdr"
)

// Run starts an attempt to ingest the range of ledgers specified in this
// session.
func (is *Session) Run() {
	if is.Cursor == nil {
		is.Err = errors.New("no cursor set on session")
		return
	}

	is.Err = is.Ingestion.Start()
	if is.Err != nil {
		return
	}

	defer is.Ingestion.Rollback()

	for is.Cursor.NextLedger() {
		is.validateLedger()
		is.clearLedger()
		is.ingestLedger()
		is.flush()

		if is.Err != nil {
			break
		}
	}
	is.Cursor.AssetsModified.UpdateAssetStats(is)

	if is.Err != nil {
		is.Ingestion.Rollback()
		return
	}

	if is.Cursor.Err != nil {
		is.Err = is.Cursor.Err
		return
	}

	is.Err = is.Ingestion.Close()
	if is.Err != nil {
		return
	}

	is.Err = is.reportCursorState()
}

func (is *Session) clearLedger() {
	if is.Err != nil {
		return
	}

	if !is.ClearExisting {
		return
	}
	start := time.Now()
	is.Err = is.Ingestion.Clear(is.Cursor.LedgerRange())
	if is.Metrics != nil {
		is.Metrics.ClearLedgerTimer.Update(time.Since(start))
	}
}

func (is *Session) effectFlagDetails(flagDetails map[string]bool, flagPtr *xdr.Uint32, setValue bool) {
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

func (is *Session) flush() {
	if is.Err != nil {
		return
	}
	is.Err = is.Ingestion.Flush()
}

func (is *Session) ingestEffects() {
	if is.Err != nil {
		return
	}

	effects := &EffectIngestion{
		Dest:        is.Ingestion,
		OperationID: is.Cursor.OperationID(),
		parent:      is.Ingestion,
	}
	source := is.Cursor.OperationSourceAccount()
	opbody := is.Cursor.Operation().Body

	switch is.Cursor.OperationType() {
	case xdr.OperationTypeCreateAccount:
		op := opbody.MustCreateAccountOp()

		effects.Add(op.Destination, history.EffectAccountCreated,
			map[string]interface{}{
				"starting_balance": amount.String(op.StartingBalance),
			},
		)

		effects.Add(source, history.EffectAccountDebited,
			map[string]interface{}{
				"asset_type": "native",
				"amount":     amount.String(op.StartingBalance),
			},
		)

		effects.Add(op.Destination, history.EffectSignerCreated,
			map[string]interface{}{
				"public_key": op.Destination.Address(),
				"weight":     keypair.DefaultSignerWeight,
			},
		)

	case xdr.OperationTypePayment:
		op := opbody.MustPaymentOp()
		dets := map[string]interface{}{"amount": amount.String(op.Amount)}
		is.assetDetails(dets, op.Asset, "")
		effects.Add(op.Destination, history.EffectAccountCredited, dets)
		effects.Add(source, history.EffectAccountDebited, dets)
	case xdr.OperationTypePathPayment:
		result := is.Cursor.OperationResult().MustPathPaymentResult().MustSuccess()
		is.ingestTradeEffects(effects, source, result.Offers)
	case xdr.OperationTypeManageOffer:
		result := is.Cursor.OperationResult().MustManageOfferResult().MustSuccess()
		is.ingestTradeEffects(effects, source, result.OffersClaimed)
	case xdr.OperationTypeCreatePassiveOffer:
		claims := []xdr.ClaimOfferAtom{}
		result := is.Cursor.OperationResult()

		// KNOWN ISSUE:  stellar-core creates results for CreatePassiveOffer operations
		// with the wrong result arm set.
		if result.Type == xdr.OperationTypeManageOffer {
			claims = result.MustManageOfferResult().MustSuccess().OffersClaimed
		} else {
			claims = result.MustCreatePassiveOfferResult().MustSuccess().OffersClaimed
		}

		is.ingestTradeEffects(effects, source, claims)
	case xdr.OperationTypeSetOptions:
		op := opbody.MustSetOptionsOp()

		if op.HomeDomain != nil {
			effects.Add(source, history.EffectAccountHomeDomainUpdated,
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
			effects.Add(source, history.EffectAccountThresholdsUpdated, thresholdDetails)
		}

		flagDetails := map[string]bool{}
		is.effectFlagDetails(flagDetails, op.SetFlags, true)
		is.effectFlagDetails(flagDetails, op.ClearFlags, false)

		if len(flagDetails) > 0 {
			effects.Add(source, history.EffectAccountFlagsUpdated, flagDetails)
		}

		if op.InflationDest != nil {
			effects.Add(source, history.EffectAccountInflationDestinationUpdated,
				map[string]interface{}{
					"inflation_destination": op.InflationDest.Address(),
				},
			)
		}

		is.ingestSignerEffects(effects, op)

	case xdr.OperationTypeChangeTrust:
		op := opbody.MustChangeTrustOp()
		dets := map[string]interface{}{"limit": amount.String(op.Limit)}
		key := xdr.LedgerKey{}
		effect := history.EffectType(0)

		is.assetDetails(dets, op.Line, "")

		key.SetTrustline(source, op.Line)

		before, after, err := is.Cursor.BeforeAndAfter(key)

		// NOTE:  when an account trusts itself, the transaction is successful but
		// no ledger entries are actually modified, leading to an "empty meta"
		// situation.  We simply continue on to the next operation in that scenario.
		if err == meta.ErrMetaNotFound {
			return
		}

		if err != nil {
			is.Err = err
			return
		}

		switch {
		case before == nil && after != nil:
			effect = history.EffectTrustlineCreated
		case before != nil && after == nil:
			effect = history.EffectTrustlineRemoved
		case before != nil && after != nil:
			effect = history.EffectTrustlineUpdated
		default:
			panic("Invalid before-and-after state")
		}

		effects.Add(source, effect, dets)
	case xdr.OperationTypeAllowTrust:
		op := opbody.MustAllowTrustOp()
		asset := op.Asset.ToAsset(source)
		dets := map[string]interface{}{
			"trustor": op.Trustor.Address(),
		}
		is.assetDetails(dets, asset, "")

		if op.Authorize {
			effects.Add(source, history.EffectTrustlineAuthorized, dets)
		} else {
			effects.Add(source, history.EffectTrustlineDeauthorized, dets)
		}

	case xdr.OperationTypeAccountMerge:
		dest := opbody.MustDestination()
		result := is.Cursor.OperationResult().MustAccountMergeResult()
		dets := map[string]interface{}{
			"amount":     amount.String(result.MustSourceAccountBalance()),
			"asset_type": "native",
		}
		effects.Add(source, history.EffectAccountDebited, dets)
		effects.Add(dest, history.EffectAccountCredited, dets)
		effects.Add(source, history.EffectAccountRemoved, map[string]interface{}{})
	case xdr.OperationTypeInflation:
		payouts := is.Cursor.OperationResult().MustInflationResult().MustPayouts()
		for _, payout := range payouts {
			effects.Add(payout.Destination, history.EffectAccountCredited,
				map[string]interface{}{
					"amount":     amount.String(payout.Amount),
					"asset_type": "native",
				},
			)
		}
	case xdr.OperationTypeManageData:
		op := opbody.MustManageDataOp()
		dets := map[string]interface{}{"name": op.DataName}
		key := xdr.LedgerKey{}
		effect := history.EffectType(0)

		key.SetData(source, string(op.DataName))

		before, after, err := is.Cursor.BeforeAndAfter(key)
		if err != nil {
			is.Err = err
			return
		}

		if after != nil {
			raw := after.Data.MustData().DataValue
			dets["value"] = base64.StdEncoding.EncodeToString(raw)
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

		effects.Add(source, effect, dets)

	default:
		is.Err = fmt.Errorf("Unknown operation type: %s", is.Cursor.OperationType())
		return
	}

	is.Err = effects.Finish()
}

// ingestLedger ingests the current ledger
func (is *Session) ingestLedger() {
	if is.Err != nil {
		return
	}

	start := time.Now()
	is.Ingestion.Ledger(
		is.Cursor.LedgerID(),
		is.Cursor.Ledger(),
		is.Cursor.SuccessfulTransactionCount(),
		is.Cursor.SuccessfulLedgerOperationCount(),
	)

	for is.Cursor.NextTx() {
		is.ingestTransaction()
	}

	is.Ingested++
	if is.Metrics != nil {
		is.Metrics.IngestLedgerTimer.Update(time.Since(start))
	}

	return
}

func (is *Session) ingestOperation() {
	if is.Err != nil {
		return
	}

	is.Err = is.Ingestion.Operation(
		is.Cursor.OperationID(),
		is.Cursor.TransactionID(),
		is.Cursor.OperationOrder(),
		is.Cursor.OperationSourceAccount(),
		is.Cursor.OperationType(),
		is.operationDetails(),
	)
	if is.Err != nil {
		return
	}

	is.ingestOperationParticipants()
	is.ingestEffects()
	is.ingestTrades()
	is.Err = is.Cursor.AssetsModified.IngestOperation(
		is.Err,
		is.Cursor.Operation(),
		&is.Cursor.Transaction().Envelope.Tx.SourceAccount,
		&core.Q{Session: is.Ingestion.DB},
	)
}

func (is *Session) ingestOperationParticipants() {
	if is.Err != nil {
		return
	}

	// Find the participants
	var p []xdr.AccountId
	p, is.Err = participants.ForOperation(
		&is.Cursor.Transaction().Envelope.Tx,
		is.Cursor.Operation(),
	)
	if is.Err != nil {
		return
	}

	is.Ingestion.OperationParticipants(is.Cursor.OperationID(), p)
}

func (is *Session) ingestSignerEffects(effects *EffectIngestion, op xdr.SetOptionsOp) {
	source := is.Cursor.OperationSourceAccount()

	be, ae, err := is.Cursor.BeforeAndAfter(source.LedgerKey())
	if err != nil {
		is.Err = err
		return
	}

	// HACK (scott) 2017-11-27:  Prevent crashes when BeforeAndAfter fails to
	// correctly work.
	if be == nil || ae == nil {
		// TODO (scott) 2018-03-02: log some info to help us track down the crash, doofus
		return
	}

	beforeAccount := be.Data.MustAccount()
	afterAccount := ae.Data.MustAccount()

	before := beforeAccount.SignerSummary()
	after := afterAccount.SignerSummary()

	// if before and after are the same, the signers have not changed
	if reflect.DeepEqual(before, after) {
		return
	}

	for addy := range before {
		weight, ok := after[addy]
		if !ok {
			effects.Add(source, history.EffectSignerRemoved, map[string]interface{}{
				"public_key": addy,
			})
			continue
		}
		effects.Add(source, history.EffectSignerUpdated, map[string]interface{}{
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

		effects.Add(source, history.EffectSignerCreated, map[string]interface{}{
			"public_key": addy,
			"weight":     weight,
		})
	}

}

func (is *Session) ingestTrades() {
	if is.Err != nil {
		return
	}

	buyer := is.Cursor.OperationSourceAccount()
	trades := []xdr.ClaimOfferAtom{}

	switch is.Cursor.OperationType() {
	case xdr.OperationTypePathPayment:
		trades = is.Cursor.OperationResult().
			MustPathPaymentResult().
			MustSuccess().
			Offers

	case xdr.OperationTypeManageOffer:
		trades = is.Cursor.OperationResult().MustManageOfferResult().MustSuccess().OffersClaimed
	case xdr.OperationTypeCreatePassiveOffer:
		result := is.Cursor.OperationResult()

		// KNOWN ISSUE:  stellar-core creates results for CreatePassiveOffer operations
		// with the wrong result arm set.
		if result.Type == xdr.OperationTypeManageOffer {
			trades = result.MustManageOfferResult().MustSuccess().OffersClaimed
		} else {
			trades = result.MustCreatePassiveOfferResult().MustSuccess().OffersClaimed
		}
	}

	q := history.Q{Session: is.Ingestion.DB}
	for i, trade := range trades {
		// stellar-core will opportunisticly garbage collect invalid offers (in the
		// event that a trader spends down their balance).  These garbage collected
		// offers get emitted in the result with the amount values set to zero.
		//
		// These zeroed ClaimOfferAtom values do not represent trades, and so we
		// skip them.
		if trade.AmountBought == 0 && trade.AmountSold == 0 {
			continue
		}

		//extract original offer price
		key := xdr.LedgerKey{}
		key.SetOffer(trade.SellerId, uint64(trade.OfferId))
		before, _, err := is.Cursor.BeforeAndAfter(key)
		if err != nil {
			is.Err = err
			return
		}
		offerPrice := before.Data.Offer.Price

		is.Err = q.InsertTrade(
			is.Cursor.OperationID(),
			int32(i),
			buyer,
			trade,
			offerPrice,
			sTime.MillisFromSeconds(is.Cursor.Ledger().CloseTime),
		)
		if is.Err != nil {
			return
		}
	}
}

func (is *Session) ingestTradeEffects(effects *EffectIngestion, buyer xdr.AccountId, claims []xdr.ClaimOfferAtom) {
	if is.Err != nil {
		return
	}

	for _, claim := range claims {
		seller := claim.SellerId
		bd, sd := is.tradeDetails(buyer, seller, claim)
		effects.Add(buyer, history.EffectTrade, bd)
		effects.Add(seller, history.EffectTrade, sd)
	}
}

func (is *Session) tradeDetails(buyer, seller xdr.AccountId, claim xdr.ClaimOfferAtom) (bd map[string]interface{}, sd map[string]interface{}) {
	bd = map[string]interface{}{
		"offer_id":      claim.OfferId,
		"seller":        seller.Address(),
		"bought_amount": amount.String(claim.AmountSold),
		"sold_amount":   amount.String(claim.AmountBought),
	}
	is.assetDetails(bd, claim.AssetSold, "bought_")
	is.assetDetails(bd, claim.AssetBought, "sold_")

	sd = map[string]interface{}{
		"offer_id":      claim.OfferId,
		"seller":        buyer.Address(),
		"bought_amount": amount.String(claim.AmountBought),
		"sold_amount":   amount.String(claim.AmountSold),
	}
	is.assetDetails(sd, claim.AssetBought, "bought_")
	is.assetDetails(sd, claim.AssetSold, "sold_")

	return
}

func (is *Session) ingestTransaction() {
	if is.Err != nil {
		return
	}

	// skip ingesting failed transactions
	if !is.Cursor.Transaction().IsSuccessful() {
		return
	}
	is.Ingestion.Transaction(
		is.Cursor.TransactionID(),
		is.Cursor.Transaction(),
		is.Cursor.TransactionFee(),
	)

	for is.Cursor.NextOp() {
		is.ingestOperation()
	}

	is.ingestTransactionParticipants()
}

func (is *Session) ingestTransactionParticipants() {
	if is.Err != nil {
		return
	}

	// Find the participants
	var p []xdr.AccountId
	p, is.Err = participants.ForTransaction(
		&is.Cursor.Transaction().Envelope.Tx,
		&is.Cursor.Transaction().ResultMeta,
		&is.Cursor.TransactionFee().Changes,
	)
	if is.Err != nil {
		return
	}

	is.Ingestion.TransactionParticipants(is.Cursor.TransactionID(), p)
}

// assetDetails sets the details for `a` on `result` using keys with `prefix`
func (is *Session) assetDetails(result map[string]interface{}, a xdr.Asset, prefix string) error {
	var (
		t    string
		code string
		i    string
	)
	err := a.Extract(&t, &code, &i)
	if err != nil {
		return err
	}
	result[prefix+"asset_type"] = t

	if a.Type == xdr.AssetTypeAssetTypeNative {
		return nil
	}

	result[prefix+"asset_code"] = code
	result[prefix+"asset_issuer"] = i
	return nil
}

// operationDetails returns the details regarding the current operation, suitable
// for ingestion into a history_operation row
func (is *Session) operationDetails() map[string]interface{} {
	details := map[string]interface{}{}
	c := is.Cursor
	source := c.OperationSourceAccount()

	switch c.OperationType() {
	case xdr.OperationTypeCreateAccount:
		op := c.Operation().Body.MustCreateAccountOp()
		details["funder"] = source.Address()
		details["account"] = op.Destination.Address()
		details["starting_balance"] = amount.String(op.StartingBalance)
	case xdr.OperationTypePayment:
		op := c.Operation().Body.MustPaymentOp()
		details["from"] = source.Address()
		details["to"] = op.Destination.Address()
		details["amount"] = amount.String(op.Amount)
		is.assetDetails(details, op.Asset, "")
	case xdr.OperationTypePathPayment:
		op := c.Operation().Body.MustPathPaymentOp()
		details["from"] = source.Address()
		details["to"] = op.Destination.Address()

		result := c.OperationResult().MustPathPaymentResult()

		details["amount"] = amount.String(op.DestAmount)
		details["source_amount"] = amount.String(result.SendAmount())
		details["source_max"] = amount.String(op.SendMax)
		is.assetDetails(details, op.DestAsset, "")
		is.assetDetails(details, op.SendAsset, "source_")

		var path = make([]map[string]interface{}, len(op.Path))
		for i := range op.Path {
			path[i] = make(map[string]interface{})
			is.assetDetails(path[i], op.Path[i], "")
		}
		details["path"] = path
	case xdr.OperationTypeManageOffer:
		op := c.Operation().Body.MustManageOfferOp()
		details["offer_id"] = op.OfferId
		details["amount"] = amount.String(op.Amount)
		details["price"] = op.Price.String()
		details["price_r"] = map[string]interface{}{
			"n": op.Price.N,
			"d": op.Price.D,
		}
		is.assetDetails(details, op.Buying, "buying_")
		is.assetDetails(details, op.Selling, "selling_")

	case xdr.OperationTypeCreatePassiveOffer:
		op := c.Operation().Body.MustCreatePassiveOfferOp()
		details["amount"] = amount.String(op.Amount)
		details["price"] = op.Price.String()
		details["price_r"] = map[string]interface{}{
			"n": op.Price.N,
			"d": op.Price.D,
		}
		is.assetDetails(details, op.Buying, "buying_")
		is.assetDetails(details, op.Selling, "selling_")
	case xdr.OperationTypeSetOptions:
		op := c.Operation().Body.MustSetOptionsOp()

		if op.InflationDest != nil {
			details["inflation_dest"] = op.InflationDest.Address()
		}

		if op.SetFlags != nil && *op.SetFlags > 0 {
			is.operationFlagDetails(details, int32(*op.SetFlags), "set")
		}

		if op.ClearFlags != nil && *op.ClearFlags > 0 {
			is.operationFlagDetails(details, int32(*op.ClearFlags), "clear")
		}

		if op.MasterWeight != nil {
			details["master_key_weight"] = *op.MasterWeight
		}

		if op.LowThreshold != nil {
			details["low_threshold"] = *op.LowThreshold
		}

		if op.MedThreshold != nil {
			details["med_threshold"] = *op.MedThreshold
		}

		if op.HighThreshold != nil {
			details["high_threshold"] = *op.HighThreshold
		}

		if op.HomeDomain != nil {
			details["home_domain"] = *op.HomeDomain
		}

		if op.Signer != nil {
			details["signer_key"] = op.Signer.Key.Address()
			details["signer_weight"] = op.Signer.Weight
		}
	case xdr.OperationTypeChangeTrust:
		op := c.Operation().Body.MustChangeTrustOp()
		is.assetDetails(details, op.Line, "")
		details["trustor"] = source.Address()
		details["trustee"] = details["asset_issuer"]
		details["limit"] = amount.String(op.Limit)
	case xdr.OperationTypeAllowTrust:
		op := c.Operation().Body.MustAllowTrustOp()
		is.assetDetails(details, op.Asset.ToAsset(source), "")
		details["trustee"] = source.Address()
		details["trustor"] = op.Trustor.Address()
		details["authorize"] = op.Authorize
	case xdr.OperationTypeAccountMerge:
		aid := c.Operation().Body.MustDestination()
		details["account"] = source.Address()
		details["into"] = aid.Address()
	case xdr.OperationTypeInflation:
		// no inflation details, presently
	case xdr.OperationTypeManageData:
		op := c.Operation().Body.MustManageDataOp()
		details["name"] = string(op.DataName)
		if op.DataValue != nil {
			details["value"] = base64.StdEncoding.EncodeToString(*op.DataValue)
		} else {
			details["value"] = nil
		}
	default:
		panic(fmt.Errorf("Unknown operation type: %s", c.OperationType()))
	}

	return details
}

// operationFlagDetails sets the account flag details for `f` on `result`.
func (is *Session) operationFlagDetails(result map[string]interface{}, f int32, prefix string) {
	var (
		n []int32
		s []string
	)

	if (f & int32(xdr.AccountFlagsAuthRequiredFlag)) > 0 {
		n = append(n, int32(xdr.AccountFlagsAuthRequiredFlag))
		s = append(s, "auth_required")
	}

	if (f & int32(xdr.AccountFlagsAuthRevocableFlag)) > 0 {
		n = append(n, int32(xdr.AccountFlagsAuthRevocableFlag))
		s = append(s, "auth_revocable")
	}

	if (f & int32(xdr.AccountFlagsAuthImmutableFlag)) > 0 {
		n = append(n, int32(xdr.AccountFlagsAuthImmutableFlag))
		s = append(s, "auth_immutable")
	}

	result[prefix+"_flags"] = n
	result[prefix+"_flags_s"] = s
}

// reportCursorState makes an http request to the configured stellar-core server
// to report that it has finished processing the data being ingested.  This
// allows stellar-core to free that storage when next it runs its own
// maintenance.
func (is *Session) reportCursorState() error {
	if is.StellarCoreURL == "" {
		return nil
	}

	if is.SkipCursorUpdate {
		return nil
	}

	core := &stellarcore.Client{URL: is.StellarCoreURL}

	err := core.SetCursor(context.Background(), "HORIZON", is.Cursor.LastLedger)

	if err != nil {
		return errors.Wrap(err, "SetCursor failed")
	}

	return nil
}

// validate ledger
func (is *Session) validateLedger() {
	if is.Err != nil {
		return
	}

	// TODO: if the cursor is running forward, load the previous legder and
	// validate.  if reverse, load the next ledger and validate.

	// if we can find no ledger where one should be, emit a warning because we
	// cannot validate.  The normal scenario for this to occur is an empty history
	// databse.

	// if hashes mistmatch, return an error

}
