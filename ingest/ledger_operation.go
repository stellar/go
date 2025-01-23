package ingest

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"

	"github.com/dgryski/go-farm"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/contractevents"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type LedgerOperation struct {
	OperationIndex    int32
	Operation         xdr.Operation
	Transaction       *LedgerTransaction
	NetworkPassphrase string
}

func (o *LedgerOperation) sourceAccountXDR() xdr.MuxedAccount {
	sourceAccount := o.Operation.SourceAccount
	if sourceAccount != nil {
		return *sourceAccount
	}

	return o.Transaction.Envelope.SourceAccount()
}

func (o *LedgerOperation) SourceAccount() string {
	muxedAccount := o.sourceAccountXDR()
	providedID := muxedAccount.ToAccountId()
	pointerToID := &providedID

	return pointerToID.Address()
}

func (o *LedgerOperation) Type() int32 {
	return int32(o.Operation.Body.Type)
}

func (o *LedgerOperation) TypeString() string {
	return xdr.OperationTypeToStringMap[o.Type()]
}

func (o *LedgerOperation) ID() int64 {
	//operationIndex needs +1 increment to stay in sync with ingest package
	return toid.New(int32(o.Transaction.Ledger.LedgerSequence()), int32(o.Transaction.Index), o.OperationIndex+1).ToInt64()
}

func (o *LedgerOperation) SourceAccountMuxed() (string, bool) {
	muxedAccount := o.sourceAccountXDR()
	if muxedAccount.Type != xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return "", false
	}

	return muxedAccount.Address(), true
}

func (o *LedgerOperation) OperationResultCode() string {
	var operationResultCode string
	operationResults, ok := o.Transaction.Result.Result.OperationResults()
	if ok {
		operationResultCode = operationResults[o.OperationIndex].Code.String()
	}

	return operationResultCode
}

func (o *LedgerOperation) OperationTraceCode() (string, error) {
	var operationTraceCode string
	var operationResults []xdr.OperationResult
	var ok bool

	operationResults, ok = o.Transaction.Result.Result.OperationResults()
	if ok {
		var operationResultTr xdr.OperationResultTr
		operationResultTr, ok = operationResults[o.OperationIndex].GetTr()
		if ok {
			operationTraceCode, err := operationResultTr.MapOperationResultTr()
			if err != nil {
				return "", err
			}
			return operationTraceCode, nil
		}
	}

	return operationTraceCode, nil
}

func (o *LedgerOperation) OperationDetails() (map[string]interface{}, error) {
	var err error
	details := map[string]interface{}{}

	switch o.Operation.Body.Type {
	case xdr.OperationTypeCreateAccount:
		details, err = o.CreateAccountDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypePayment:
		details, err = o.PaymentDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypePathPaymentStrictReceive:
		details, err = o.PathPaymentStrictReceiveDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypePathPaymentStrictSend:
		details, err = o.PathPaymentStrictSendDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeManageBuyOffer:
		details, err = o.ManageBuyOfferDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeManageSellOffer:
		details, err = o.ManageSellOfferDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeCreatePassiveSellOffer:
		details, err = o.CreatePassiveSellOfferDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeSetOptions:
		details, err = o.SetOptionsDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeChangeTrust:
		details, err = o.ChangeTrustDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeAllowTrust:
		details, err = o.AllowTrustDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeAccountMerge:
		details, err = o.AccountMergeDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeInflation:
		details, err = o.InflationDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeManageData:
		details, err = o.ManageDataDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeBumpSequence:
		details, err = o.BumpSequenceDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeCreateClaimableBalance:
		details, err = o.CreateClaimableBalanceDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeClaimClaimableBalance:
		details, err = o.ClaimClaimableBalanceDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeBeginSponsoringFutureReserves:
		details, err = o.BeginSponsoringFutureReservesDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeEndSponsoringFutureReserves:
		details, err = o.EndSponsoringFutureReserveDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeRevokeSponsorship:
		details, err = o.RevokeSponsorshipDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeClawback:
		details, err = o.ClawbackDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeClawbackClaimableBalance:
		details, err = o.ClawbackClaimableBalanceDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeSetTrustLineFlags:
		details, err = o.SetTrustLineFlagsDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeLiquidityPoolDeposit:
		details, err = o.LiquidityPoolDepositDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeLiquidityPoolWithdraw:
		details, err = o.LiquidityPoolWithdrawDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeInvokeHostFunction:
		details, err = o.InvokeHostFunctionDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeExtendFootprintTtl:
		details, err = o.ExtendFootprintTtlDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypeRestoreFootprint:
		details, err = o.RestoreFootprintDetails()
		if err != nil {
			return details, err
		}
	default:
		return details, fmt.Errorf("unknown operation type: %s", o.Operation.Body.Type.String())
	}

	return details, nil
}

func (o *LedgerOperation) CreateAccountDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetCreateAccountOp()
	if !ok {
		return details, fmt.Errorf("could not access CreateAccount info for this operation (index %d)", o.OperationIndex)
	}

	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "funder"); err != nil {
		return details, err
	}
	details["account"] = op.Destination.Address()
	details["starting_balance"] = int64(op.StartingBalance)

	return details, nil
}

func (o *LedgerOperation) addAccountAndMuxedAccountDetails(result map[string]interface{}, a xdr.MuxedAccount, prefix string) error {
	account_id := a.ToAccountId()
	result[prefix] = account_id.Address()
	prefix = o.FormatPrefix(prefix)
	if a.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		muxedAccountAddress, err := a.GetAddress()
		if err != nil {
			return err
		}
		result[prefix+"muxed"] = muxedAccountAddress
		muxedAccountId, err := a.GetId()
		if err != nil {
			return err
		}
		result[prefix+"muxed_id"] = muxedAccountId
	}
	return nil
}

func (o *LedgerOperation) PaymentDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetPaymentOp()
	if !ok {
		return details, fmt.Errorf("could not access Payment info for this operation (index %d)", o.OperationIndex)
	}

	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "from"); err != nil {
		return details, err
	}
	if err := o.addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
		return details, err
	}
	details["amount"] = int64(op.Amount)
	if err := o.addAssetDetails(details, op.Asset, ""); err != nil {
		return details, err
	}

	return details, nil
}

func (o *LedgerOperation) PathPaymentStrictReceiveDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetPathPaymentStrictReceiveOp()
	if !ok {
		return details, fmt.Errorf("could not access PathPaymentStrictReceive info for this operation (index %d)", o.OperationIndex)
	}

	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "from"); err != nil {
		return details, err
	}
	if err := o.addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
		return details, err
	}
	details["amount"] = int64(op.DestAmount)
	details["source_amount"] = amount.String(0)
	details["source_max"] = int64(op.SendMax)
	if err := o.addAssetDetails(details, op.DestAsset, ""); err != nil {
		return details, err
	}
	if err := o.addAssetDetails(details, op.SendAsset, "source"); err != nil {
		return details, err
	}

	if o.Transaction.Successful() {
		allOperationResults, ok := o.Transaction.Result.OperationResults()
		if !ok {
			return details, fmt.Errorf("could not access any results for this transaction")
		}
		currentOperationResult := allOperationResults[o.OperationIndex]
		resultBody, ok := currentOperationResult.GetTr()
		if !ok {
			return details, fmt.Errorf("could not access result body for this operation (index %d)", o.OperationIndex)
		}
		result, ok := resultBody.GetPathPaymentStrictReceiveResult()
		if !ok {
			return details, fmt.Errorf("could not access PathPaymentStrictReceive result info for this operation (index %d)", o.OperationIndex)
		}
		details["source_amount"] = int64(result.SendAmount())
	}

	details["path"] = o.TransformPath(op.Path)
	return details, nil
}

func (o *LedgerOperation) PathPaymentStrictSendDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetPathPaymentStrictSendOp()
	if !ok {
		return details, fmt.Errorf("could not access PathPaymentStrictSend info for this operation (index %d)", o.OperationIndex)
	}

	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "from"); err != nil {
		return details, err
	}
	if err := o.addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
		return details, err
	}
	details["amount"] = float64(0)
	details["source_amount"] = int64(op.SendAmount)
	details["destination_min"] = amount.String(op.DestMin)
	if err := o.addAssetDetails(details, op.DestAsset, ""); err != nil {
		return details, err
	}
	if err := o.addAssetDetails(details, op.SendAsset, "source"); err != nil {
		return details, err
	}

	if o.Transaction.Successful() {
		allOperationResults, ok := o.Transaction.Result.OperationResults()
		if !ok {
			return details, fmt.Errorf("could not access any results for this transaction")
		}
		currentOperationResult := allOperationResults[o.OperationIndex]
		resultBody, ok := currentOperationResult.GetTr()
		if !ok {
			return details, fmt.Errorf("could not access result body for this operation (index %d)", o.OperationIndex)
		}
		result, ok := resultBody.GetPathPaymentStrictSendResult()
		if !ok {
			return details, fmt.Errorf("could not access GetPathPaymentStrictSendResult result info for this operation (index %d)", o.OperationIndex)
		}
		details["amount"] = int64(result.DestAmount())
	}

	details["path"] = o.TransformPath(op.Path)

	return details, nil
}
func (o *LedgerOperation) ManageBuyOfferDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetManageBuyOfferOp()
	if !ok {
		return details, fmt.Errorf("could not access ManageBuyOffer info for this operation (index %d)", o.OperationIndex)
	}

	details["offer_id"] = int64(op.OfferId)
	details["amount"] = int64(op.BuyAmount)
	if err := o.addPriceDetails(details, op.Price, ""); err != nil {
		return details, err
	}

	if err := o.addAssetDetails(details, op.Buying, "buying"); err != nil {
		return details, err
	}
	if err := o.addAssetDetails(details, op.Selling, "selling"); err != nil {
		return details, err
	}

	return details, nil
}

func (o *LedgerOperation) addPriceDetails(result map[string]interface{}, price xdr.Price, prefix string) error {
	prefix = o.FormatPrefix(prefix)
	parsedPrice, err := strconv.ParseFloat(price.String(), 64)
	if err != nil {
		return err
	}
	result[prefix+"price"] = parsedPrice
	result[prefix+"price_r"] = Price{
		Numerator:   int32(price.N),
		Denominator: int32(price.D),
	}
	return nil
}

func (o *LedgerOperation) ManageSellOfferDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetManageSellOfferOp()
	if !ok {
		return details, fmt.Errorf("could not access ManageSellOffer info for this operation (index %d)", o.OperationIndex)
	}

	details["offer_id"] = int64(op.OfferId)
	details["amount"] = int64(op.Amount)
	if err := o.addPriceDetails(details, op.Price, ""); err != nil {
		return details, err
	}

	if err := o.addAssetDetails(details, op.Buying, "buying"); err != nil {
		return details, err
	}
	if err := o.addAssetDetails(details, op.Selling, "selling"); err != nil {
		return details, err
	}

	return details, nil
}
func (o *LedgerOperation) CreatePassiveSellOfferDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetCreatePassiveSellOfferOp()
	if !ok {
		return details, fmt.Errorf("could not access CreatePassiveSellOffer info for this operation (index %d)", o.OperationIndex)
	}

	details["amount"] = int64(op.Amount)
	if err := o.addPriceDetails(details, op.Price, ""); err != nil {
		return details, err
	}

	if err := o.addAssetDetails(details, op.Buying, "buying"); err != nil {
		return details, err
	}
	if err := o.addAssetDetails(details, op.Selling, "selling"); err != nil {
		return details, err
	}

	return details, nil
}
func (o *LedgerOperation) SetOptionsDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetSetOptionsOp()
	if !ok {
		return details, fmt.Errorf("could not access GetSetOptions info for this operation (index %d)", o.OperationIndex)
	}

	if op.InflationDest != nil {
		details["inflation_dest"] = op.InflationDest.Address()
	}

	if op.SetFlags != nil && *op.SetFlags > 0 {
		o.addOperationFlagToOperationDetails(details, uint32(*op.SetFlags), "set")
	}

	if op.ClearFlags != nil && *op.ClearFlags > 0 {
		o.addOperationFlagToOperationDetails(details, uint32(*op.ClearFlags), "clear")
	}

	if op.MasterWeight != nil {
		details["master_key_weight"] = uint32(*op.MasterWeight)
	}

	if op.LowThreshold != nil {
		details["low_threshold"] = uint32(*op.LowThreshold)
	}

	if op.MedThreshold != nil {
		details["med_threshold"] = uint32(*op.MedThreshold)
	}

	if op.HighThreshold != nil {
		details["high_threshold"] = uint32(*op.HighThreshold)
	}

	if op.HomeDomain != nil {
		details["home_domain"] = string(*op.HomeDomain)
	}

	if op.Signer != nil {
		details["signer_key"] = op.Signer.Key.Address()
		details["signer_weight"] = uint32(op.Signer.Weight)
	}

	return details, nil
}

func (o *LedgerOperation) addOperationFlagToOperationDetails(result map[string]interface{}, flag uint32, prefix string) {
	intFlags := make([]int32, 0)
	stringFlags := make([]string, 0)

	if (int64(flag) & int64(xdr.AccountFlagsAuthRequiredFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthRequiredFlag))
		stringFlags = append(stringFlags, "auth_required")
	}

	if (int64(flag) & int64(xdr.AccountFlagsAuthRevocableFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthRevocableFlag))
		stringFlags = append(stringFlags, "auth_revocable")
	}

	if (int64(flag) & int64(xdr.AccountFlagsAuthImmutableFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthImmutableFlag))
		stringFlags = append(stringFlags, "auth_immutable")
	}

	if (int64(flag) & int64(xdr.AccountFlagsAuthClawbackEnabledFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthClawbackEnabledFlag))
		stringFlags = append(stringFlags, "auth_clawback_enabled")
	}

	prefix = o.FormatPrefix(prefix)
	result[prefix+"flags"] = intFlags
	result[prefix+"flags_s"] = stringFlags
}

func (o *LedgerOperation) ChangeTrustDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetChangeTrustOp()
	if !ok {
		return details, fmt.Errorf("could not access GetChangeTrust info for this operation (index %d)", o.OperationIndex)
	}

	if op.Line.Type == xdr.AssetTypeAssetTypePoolShare {
		if err := o.addLiquidityPoolAssetDetails(details, *op.Line.LiquidityPool); err != nil {
			return details, err
		}
	} else {
		if err := o.addAssetDetails(details, op.Line.ToAsset(), ""); err != nil {
			return details, err
		}
		details["trustee"] = details["asset_issuer"]
	}

	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "trustor"); err != nil {
		return details, err
	}

	details["limit"] = int64(op.Limit)

	return details, nil
}

func (o *LedgerOperation) addLiquidityPoolAssetDetails(result map[string]interface{}, lpp xdr.LiquidityPoolParameters) error {
	result["asset_type"] = "liquidity_pool_shares"
	if lpp.Type != xdr.LiquidityPoolTypeLiquidityPoolConstantProduct {
		return fmt.Errorf("unknown liquidity pool type %d", lpp.Type)
	}
	cp := lpp.ConstantProduct
	poolID, err := xdr.NewPoolId(cp.AssetA, cp.AssetB, cp.Fee)
	if err != nil {
		return err
	}

	result["liquidity_pool_id"] = o.PoolIDToString(poolID)

	return nil
}

func (o *LedgerOperation) AllowTrustDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetAllowTrustOp()
	if !ok {
		return details, fmt.Errorf("could not access AllowTrust info for this operation (index %d)", o.OperationIndex)
	}

	if err := o.addAssetDetails(details, op.Asset.ToAsset(o.sourceAccountXDR().ToAccountId()), ""); err != nil {
		return details, err
	}
	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "trustee"); err != nil {
		return details, err
	}
	details["trustor"] = op.Trustor.Address()
	shouldAuth := xdr.TrustLineFlags(op.Authorize).IsAuthorized()
	details["authorize"] = shouldAuth
	shouldAuthLiabilities := xdr.TrustLineFlags(op.Authorize).IsAuthorizedToMaintainLiabilitiesFlag()
	if shouldAuthLiabilities {
		details["authorize_to_maintain_liabilities"] = shouldAuthLiabilities
	}
	shouldClawbackEnabled := xdr.TrustLineFlags(op.Authorize).IsClawbackEnabledFlag()
	if shouldClawbackEnabled {
		details["clawback_enabled"] = shouldClawbackEnabled
	}

	return details, nil
}

func (o *LedgerOperation) AccountMergeDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	destinationAccount, ok := o.Operation.Body.GetDestination()
	if !ok {
		return details, fmt.Errorf("could not access Destination info for this operation (index %d)", o.OperationIndex)
	}

	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "account"); err != nil {
		return details, err
	}
	if err := o.addAccountAndMuxedAccountDetails(details, destinationAccount, "into"); err != nil {
		return details, err
	}

	return details, nil
}

// Inflation operations don't have information that affects the details struct
func (o *LedgerOperation) InflationDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	return details, nil
}

func (o *LedgerOperation) ManageDataDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetManageDataOp()
	if !ok {
		return details, fmt.Errorf("could not access GetManageData info for this operation (index %d)", o.OperationIndex)
	}

	details["name"] = string(op.DataName)
	if op.DataValue != nil {
		details["value"] = base64.StdEncoding.EncodeToString(*op.DataValue)
	} else {
		details["value"] = nil
	}

	return details, nil
}

func (o *LedgerOperation) BumpSequenceDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetBumpSequenceOp()
	if !ok {
		return details, fmt.Errorf("could not access BumpSequence info for this operation (index %d)", o.OperationIndex)
	}

	details["bump_to"] = fmt.Sprintf("%d", op.BumpTo)

	return details, nil
}
func (o *LedgerOperation) CreateClaimableBalanceDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetCreateClaimableBalanceOp()
	if !ok {
		return details, fmt.Errorf("could not access CreateClaimableBalance info for this operation (index %d)", o.OperationIndex)
	}

	details["asset"] = op.Asset.StringCanonical()
	err := o.addAssetDetails(details, op.Asset, "")
	if err != nil {
		return details, err
	}

	details["amount"] = int64(op.Amount)
	details["claimants"] = o.TransformClaimants(op.Claimants)

	return details, nil
}

func (o *LedgerOperation) ClaimClaimableBalanceDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetClaimClaimableBalanceOp()
	if !ok {
		return details, fmt.Errorf("could not access ClaimClaimableBalance info for this operation (index %d)", o.OperationIndex)
	}

	balanceID, err := xdr.MarshalHex(op.BalanceId)
	if err != nil {
		return details, fmt.Errorf("invalid balanceId in op: %d", o.OperationIndex)
	}
	details["balance_id"] = balanceID
	if err := o.addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "claimant"); err != nil {
		return details, err
	}

	return details, nil
}

func (o *LedgerOperation) BeginSponsoringFutureReservesDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetBeginSponsoringFutureReservesOp()
	if !ok {
		return details, fmt.Errorf("could not access BeginSponsoringFutureReserves info for this operation (index %d)", o.OperationIndex)
	}

	details["sponsored_id"] = op.SponsoredId.Address()

	return details, nil
}

func (o *LedgerOperation) EndSponsoringFutureReserveDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	beginSponsorOp := o.findInitatingBeginSponsoringOp()
	if beginSponsorOp != nil {
		beginSponsorshipSource := o.sourceAccountXDR()
		if err := o.addAccountAndMuxedAccountDetails(details, beginSponsorshipSource, "begin_sponsor"); err != nil {
			return details, err
		}
	}

	return details, nil
}

func (o *LedgerOperation) findInitatingBeginSponsoringOp() *SponsorshipOutput {
	if !o.Transaction.Successful() {
		// Failed transactions may not have a compliant sandwich structure
		// we can rely on (e.g. invalid nesting or a being operation with the wrong sponsoree ID)
		// and thus we bail out since we could return incorrect information.
		return nil
	}
	sponsoree := o.sourceAccountXDR().ToAccountId()
	operations := o.Transaction.Envelope.Operations()
	for i := int(o.OperationIndex) - 1; i >= 0; i-- {
		if beginOp, ok := operations[i].Body.GetBeginSponsoringFutureReservesOp(); ok &&
			beginOp.SponsoredId.Address() == sponsoree.Address() {
			result := SponsorshipOutput{
				Operation:      operations[i],
				OperationIndex: uint32(i),
			}
			return &result
		}
	}
	return nil
}

func (o *LedgerOperation) RevokeSponsorshipDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetRevokeSponsorshipOp()
	if !ok {
		return details, fmt.Errorf("could not access RevokeSponsorship info for this operation (index %d)", o.OperationIndex)
	}

	switch op.Type {
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry:
		if err := o.addLedgerKeyToDetails(details, *op.LedgerKey); err != nil {
			return details, err
		}
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner:
		details["signer_account_id"] = op.Signer.AccountId.Address()
		details["signer_key"] = op.Signer.SignerKey.Address()
	}

	return details, nil
}

func (o *LedgerOperation) addLedgerKeyToDetails(result map[string]interface{}, ledgerKey xdr.LedgerKey) error {
	switch ledgerKey.Type {
	case xdr.LedgerEntryTypeAccount:
		result["account_id"] = ledgerKey.Account.AccountId.Address()
	case xdr.LedgerEntryTypeClaimableBalance:
		marshalHex, err := xdr.MarshalHex(ledgerKey.ClaimableBalance.BalanceId)
		if err != nil {
			return fmt.Errorf("in claimable balance: %w", err)
		}
		result["claimable_balance_id"] = marshalHex
	case xdr.LedgerEntryTypeData:
		result["data_account_id"] = ledgerKey.Data.AccountId.Address()
		result["data_name"] = string(ledgerKey.Data.DataName)
	case xdr.LedgerEntryTypeOffer:
		result["offer_id"] = int64(ledgerKey.Offer.OfferId)
	case xdr.LedgerEntryTypeTrustline:
		result["trustline_account_id"] = ledgerKey.TrustLine.AccountId.Address()
		if ledgerKey.TrustLine.Asset.Type == xdr.AssetTypeAssetTypePoolShare {
			result["trustline_liquidity_pool_id"] = o.PoolIDToString(*ledgerKey.TrustLine.Asset.LiquidityPoolId)
		} else {
			result["trustline_asset"] = ledgerKey.TrustLine.Asset.ToAsset().StringCanonical()
		}
	case xdr.LedgerEntryTypeLiquidityPool:
		result["liquidity_pool_id"] = o.PoolIDToString(ledgerKey.LiquidityPool.LiquidityPoolId)
	}
	return nil
}

func (o *LedgerOperation) ClawbackDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetClawbackOp()
	if !ok {
		return details, fmt.Errorf("could not access Clawback info for this operation (index %d)", o.OperationIndex)
	}

	if err := o.addAssetDetails(details, op.Asset, ""); err != nil {
		return details, err
	}
	if err := o.addAccountAndMuxedAccountDetails(details, op.From, "from"); err != nil {
		return details, err
	}
	details["amount"] = int64(op.Amount)

	return details, nil
}
func (o *LedgerOperation) ClawbackClaimableBalanceDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetClawbackClaimableBalanceOp()
	if !ok {
		return details, fmt.Errorf("could not access ClawbackClaimableBalance info for this operation (index %d)", o.OperationIndex)
	}

	balanceID, err := xdr.MarshalHex(op.BalanceId)
	if err != nil {
		return details, fmt.Errorf("invalid balanceId in op: %d", o.OperationIndex)
	}
	details["balance_id"] = balanceID

	return details, nil
}
func (o *LedgerOperation) SetTrustLineFlagsDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetSetTrustLineFlagsOp()
	if !ok {
		return details, fmt.Errorf("could not access SetTrustLineFlags info for this operation (index %d)", o.OperationIndex)
	}

	details["trustor"] = op.Trustor.Address()
	if err := o.addAssetDetails(details, op.Asset, ""); err != nil {
		return details, err
	}
	if op.SetFlags > 0 {
		o.addTrustLineFlagToDetails(details, xdr.TrustLineFlags(op.SetFlags), "set")

	}
	if op.ClearFlags > 0 {
		o.addTrustLineFlagToDetails(details, xdr.TrustLineFlags(op.ClearFlags), "clear")
	}

	return details, nil
}

func (o *LedgerOperation) addTrustLineFlagToDetails(result map[string]interface{}, f xdr.TrustLineFlags, prefix string) {
	var (
		n []int32
		s []string
	)

	if f.IsAuthorized() {
		n = append(n, int32(xdr.TrustLineFlagsAuthorizedFlag))
		s = append(s, "authorized")
	}

	if f.IsAuthorizedToMaintainLiabilitiesFlag() {
		n = append(n, int32(xdr.TrustLineFlagsAuthorizedToMaintainLiabilitiesFlag))
		s = append(s, "authorized_to_maintain_liabilities")
	}

	if f.IsClawbackEnabledFlag() {
		n = append(n, int32(xdr.TrustLineFlagsTrustlineClawbackEnabledFlag))
		s = append(s, "clawback_enabled")
	}

	prefix = o.FormatPrefix(prefix)
	result[prefix+"flags"] = n
	result[prefix+"flags_s"] = s
}

func (o *LedgerOperation) LiquidityPoolDepositDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetLiquidityPoolDepositOp()
	if !ok {
		return details, fmt.Errorf("could not access LiquidityPoolDeposit info for this operation (index %d)", o.OperationIndex)
	}

	details["liquidity_pool_id"] = o.PoolIDToString(op.LiquidityPoolId)
	var (
		assetA, assetB         xdr.Asset
		depositedA, depositedB xdr.Int64
		sharesReceived         xdr.Int64
	)
	if o.Transaction.Successful() {
		// we will use the defaults (omitted asset and 0 amounts) if the transaction failed
		lp, delta, err := o.getLiquidityPoolAndProductDelta(&op.LiquidityPoolId)
		if err != nil {
			return nil, err
		}
		params := lp.Body.ConstantProduct.Params
		assetA, assetB = params.AssetA, params.AssetB
		depositedA, depositedB = delta.ReserveA, delta.ReserveB
		sharesReceived = delta.TotalPoolShares
	}

	// Process ReserveA Details
	if err := o.addAssetDetails(details, assetA, "reserve_a"); err != nil {
		return details, err
	}
	details["reserve_a_max_amount"] = int64(op.MaxAmountA)
	depositA, err := strconv.ParseFloat(amount.String(depositedA), 64)
	if err != nil {
		return details, err
	}
	details["reserve_a_deposit_amount"] = depositA

	//Process ReserveB Details
	if err := o.addAssetDetails(details, assetB, "reserve_b"); err != nil {
		return details, err
	}
	details["reserve_b_max_amount"] = int64(op.MaxAmountB)
	depositB, err := strconv.ParseFloat(amount.String(depositedB), 64)
	if err != nil {
		return details, err
	}
	details["reserve_b_deposit_amount"] = depositB

	if err := o.addPriceDetails(details, op.MinPrice, "min"); err != nil {
		return details, err
	}
	if err := o.addPriceDetails(details, op.MaxPrice, "max"); err != nil {
		return details, err
	}

	sharesToFloat, err := strconv.ParseFloat(amount.String(sharesReceived), 64)
	if err != nil {
		return details, err
	}
	details["shares_received"] = sharesToFloat

	return details, nil
}

// operation xdr.Operation, operationIndex int32, transaction LedgerTransaction, ledgerSeq int32
func (o *LedgerOperation) getLiquidityPoolAndProductDelta(lpID *xdr.PoolId) (*xdr.LiquidityPoolEntry, *LiquidityPoolDelta, error) {
	changes, err := o.Transaction.GetOperationChanges(uint32(o.OperationIndex))
	if err != nil {
		return nil, nil, err
	}

	for _, c := range changes {
		if c.Type != xdr.LedgerEntryTypeLiquidityPool {
			continue
		}
		// The delta can be caused by a full removal or full creation of the liquidity pool
		var lp *xdr.LiquidityPoolEntry
		var preA, preB, preShares xdr.Int64
		if c.Pre != nil {
			if lpID != nil && c.Pre.Data.LiquidityPool.LiquidityPoolId != *lpID {
				// if we were looking for specific pool id, then check on it
				continue
			}
			lp = c.Pre.Data.LiquidityPool
			if c.Pre.Data.LiquidityPool.Body.Type != xdr.LiquidityPoolTypeLiquidityPoolConstantProduct {
				return nil, nil, fmt.Errorf("unexpected liquity pool body type %d", c.Pre.Data.LiquidityPool.Body.Type)
			}
			cpPre := c.Pre.Data.LiquidityPool.Body.ConstantProduct
			preA, preB, preShares = cpPre.ReserveA, cpPre.ReserveB, cpPre.TotalPoolShares
		}
		var postA, postB, postShares xdr.Int64
		if c.Post != nil {
			if lpID != nil && c.Post.Data.LiquidityPool.LiquidityPoolId != *lpID {
				// if we were looking for specific pool id, then check on it
				continue
			}
			lp = c.Post.Data.LiquidityPool
			if c.Post.Data.LiquidityPool.Body.Type != xdr.LiquidityPoolTypeLiquidityPoolConstantProduct {
				return nil, nil, fmt.Errorf("unexpected liquity pool body type %d", c.Post.Data.LiquidityPool.Body.Type)
			}
			cpPost := c.Post.Data.LiquidityPool.Body.ConstantProduct
			postA, postB, postShares = cpPost.ReserveA, cpPost.ReserveB, cpPost.TotalPoolShares
		}
		delta := &LiquidityPoolDelta{
			ReserveA:        postA - preA,
			ReserveB:        postB - preB,
			TotalPoolShares: postShares - preShares,
		}
		return lp, delta, nil
	}

	return nil, nil, fmt.Errorf("liquidity pool change not found")
}

func (o *LedgerOperation) LiquidityPoolWithdrawDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetLiquidityPoolWithdrawOp()
	if !ok {
		return details, fmt.Errorf("could not access LiquidityPoolWithdraw info for this operation (index %d)", o.OperationIndex)
	}

	details["liquidity_pool_id"] = o.PoolIDToString(op.LiquidityPoolId)
	var (
		assetA, assetB       xdr.Asset
		receivedA, receivedB xdr.Int64
	)
	if o.Transaction.Successful() {
		// we will use the defaults (omitted asset and 0 amounts) if the transaction failed
		lp, delta, err := o.getLiquidityPoolAndProductDelta(&op.LiquidityPoolId)
		if err != nil {
			return nil, err
		}
		params := lp.Body.ConstantProduct.Params
		assetA, assetB = params.AssetA, params.AssetB
		receivedA, receivedB = -delta.ReserveA, -delta.ReserveB
	}
	// Process AssetA Details
	if err := o.addAssetDetails(details, assetA, "reserve_a"); err != nil {
		return details, err
	}
	details["reserve_a_min_amount"] = int64(op.MinAmountA)
	details["reserve_a_withdraw_amount"] = int64(receivedA)

	// Process AssetB Details
	if err := o.addAssetDetails(details, assetB, "reserve_b"); err != nil {
		return details, err
	}
	details["reserve_b_min_amount"] = int64(op.MinAmountB)
	details["reserve_b_withdraw_amount"] = int64(receivedB)

	details["shares"] = int64(op.Amount)

	return details, nil
}

func (o *LedgerOperation) InvokeHostFunctionDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetInvokeHostFunctionOp()
	if !ok {
		return details, fmt.Errorf("could not access InvokeHostFunction info for this operation (index %d)", o.OperationIndex)
	}

	details["function"] = op.HostFunction.Type.String()

	switch op.HostFunction.Type {
	case xdr.HostFunctionTypeHostFunctionTypeInvokeContract:
		invokeArgs := op.HostFunction.MustInvokeContract()
		args := make([]xdr.ScVal, 0, len(invokeArgs.Args)+2)
		args = append(args, xdr.ScVal{Type: xdr.ScValTypeScvAddress, Address: &invokeArgs.ContractAddress})
		args = append(args, xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &invokeArgs.FunctionName})
		args = append(args, invokeArgs.Args...)

		details["type"] = "invoke_contract"

		contractId, err := invokeArgs.ContractAddress.String()
		if err != nil {
			return nil, err
		}

		details["ledger_key_hash"] = o.Transaction.LedgerKeyHashFromTxEnvelope()
		details["contract_id"] = contractId
		var contractCodeHash string
		contractCodeHash, ok = o.Transaction.ContractCodeHashFromTxEnvelope()
		if ok {
			details["contract_code_hash"] = contractCodeHash
		}

		details["parameters"], details["parameters_decoded"] = o.serializeParameters(args)

		balanceChanges, err := o.parseAssetBalanceChangesFromContractEvents()
		if err != nil {
			return nil, err
		}

		details["asset_balance_changes"] = balanceChanges

	case xdr.HostFunctionTypeHostFunctionTypeCreateContract:
		args := op.HostFunction.MustCreateContract()
		details["type"] = "create_contract"

		details["ledger_key_hash"] = o.Transaction.LedgerKeyHashFromTxEnvelope()

		var contractID string
		contractID, ok = o.Transaction.contractIdFromTxEnvelope()
		if ok {
			details["contract_id"] = contractID
		}

		var contractCodeHash string
		contractCodeHash, ok = o.Transaction.ContractCodeHashFromTxEnvelope()
		if ok {
			details["contract_code_hash"] = contractCodeHash
		}

		preimageTypeMap, err := o.switchContractIdPreimageType(args.ContractIdPreimage)
		if err != nil {
			return details, nil
		}

		for key, val := range preimageTypeMap {
			if _, ok := preimageTypeMap[key]; ok {
				details[key] = val
			}
		}
	case xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm:
		details["type"] = "upload_wasm"
		details["ledger_key_hash"] = o.Transaction.LedgerKeyHashFromTxEnvelope()

		var contractCodeHash string
		contractCodeHash, ok = o.Transaction.ContractCodeHashFromTxEnvelope()
		if ok {
			details["contract_code_hash"] = contractCodeHash
		}
	case xdr.HostFunctionTypeHostFunctionTypeCreateContractV2:
		args := op.HostFunction.MustCreateContractV2()
		details["type"] = "create_contract_v2"

		details["ledger_key_hash"] = o.Transaction.LedgerKeyHashFromTxEnvelope()

		var contractID string
		contractID, ok = o.Transaction.contractIdFromTxEnvelope()
		if ok {
			details["contract_id"] = contractID
		}

		var contractCodeHash string
		contractCodeHash, ok = o.Transaction.ContractCodeHashFromTxEnvelope()
		if ok {
			details["contract_code_hash"] = contractCodeHash
		}

		// ConstructorArgs is a list of ScVals
		// This will initially be handled the same as InvokeContractParams until a different
		// model is found necessary.
		constructorArgs := args.ConstructorArgs
		details["parameters"], details["parameters_decoded"] = o.serializeParameters(constructorArgs)

		preimageTypeMap, err := o.switchContractIdPreimageType(args.ContractIdPreimage)
		if err != nil {
			return details, nil
		}

		for key, val := range preimageTypeMap {
			if _, ok := preimageTypeMap[key]; ok {
				details[key] = val
			}
		}
	default:
		panic(fmt.Errorf("unknown host function type: %s", op.HostFunction.Type))
	}

	return details, nil
}

func (o *LedgerOperation) ExtendFootprintTtlDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetExtendFootprintTtlOp()
	if !ok {
		return details, fmt.Errorf("could not access ExtendFootprintTtl info for this operation (index %d)", o.OperationIndex)
	}

	details["type"] = "extend_footprint_ttl"
	details["extend_to"] = int32(op.ExtendTo)

	details["ledger_key_hash"] = o.Transaction.LedgerKeyHashFromTxEnvelope()

	var contractID string
	contractID, ok = o.Transaction.contractIdFromTxEnvelope()
	if ok {
		details["contract_id"] = contractID
	}

	var contractCodeHash string
	contractCodeHash, ok = o.Transaction.ContractCodeHashFromTxEnvelope()
	if ok {
		details["contract_code_hash"] = contractCodeHash
	}

	return details, nil
}
func (o *LedgerOperation) RestoreFootprintDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	_, ok := o.Operation.Body.GetRestoreFootprintOp()
	if !ok {
		return details, fmt.Errorf("could not access RestoreFootprint info for this operation (index %d)", o.OperationIndex)
	}

	details["type"] = "restore_footprint"

	details["ledger_key_hash"] = o.Transaction.LedgerKeyHashFromTxEnvelope()

	var contractID string
	contractID, ok = o.Transaction.contractIdFromTxEnvelope()
	if ok {
		details["contract_id"] = contractID
	}

	var contractCodeHash string
	contractCodeHash, ok = o.Transaction.ContractCodeHashFromTxEnvelope()
	if ok {
		details["contract_code_hash"] = contractCodeHash
	}

	return details, nil
}

func (o *LedgerOperation) serializeParameters(args []xdr.ScVal) ([]map[string]string, []map[string]string) {
	params := make([]map[string]string, 0, len(args))
	paramsDecoded := make([]map[string]string, 0, len(args))

	for _, param := range args {
		serializedParam := map[string]string{}
		serializedParam["value"] = "n/a"
		serializedParam["type"] = "n/a"

		serializedParamDecoded := map[string]string{}
		serializedParamDecoded["value"] = "n/a"
		serializedParamDecoded["type"] = "n/a"

		if scValTypeName, ok := param.ArmForSwitch(int32(param.Type)); ok {
			serializedParam["type"] = scValTypeName
			serializedParamDecoded["type"] = scValTypeName
			if raw, err := param.MarshalBinary(); err == nil {
				serializedParam["value"] = base64.StdEncoding.EncodeToString(raw)
				serializedParamDecoded["value"] = param.String()
			}
		}
		params = append(params, serializedParam)
		paramsDecoded = append(paramsDecoded, serializedParamDecoded)
	}

	return params, paramsDecoded
}

func (o *LedgerOperation) parseAssetBalanceChangesFromContractEvents() ([]map[string]interface{}, error) {
	balanceChanges := []map[string]interface{}{}

	diagnosticEvents, err := o.Transaction.GetDiagnosticEvents()
	if err != nil {
		// this operation in this context must be an InvokeHostFunctionOp, therefore V3Meta should be present
		// as it's in same soroban model, so if any err, it's real,
		return nil, err
	}

	for _, contractEvent := range o.filterEvents(diagnosticEvents) {
		// Parse the xdr contract event to contractevents.StellarAssetContractEvent model

		// has some convenience like to/from attributes are expressed in strkey format for accounts(G...) and contracts(C...)
		if sacEvent, err := contractevents.NewStellarAssetContractEvent(&contractEvent, o.NetworkPassphrase); err == nil {
			switch sacEvent.GetType() {
			case contractevents.EventTypeTransfer:
				transferEvt := sacEvent.(*contractevents.TransferEvent)
				balanceChanges = append(balanceChanges, o.createSACBalanceChangeEntry(transferEvt.From, transferEvt.To, transferEvt.Amount, transferEvt.Asset, "transfer"))
			case contractevents.EventTypeMint:
				mintEvt := sacEvent.(*contractevents.MintEvent)
				balanceChanges = append(balanceChanges, o.createSACBalanceChangeEntry("", mintEvt.To, mintEvt.Amount, mintEvt.Asset, "mint"))
			case contractevents.EventTypeClawback:
				clawbackEvt := sacEvent.(*contractevents.ClawbackEvent)
				balanceChanges = append(balanceChanges, o.createSACBalanceChangeEntry(clawbackEvt.From, "", clawbackEvt.Amount, clawbackEvt.Asset, "clawback"))
			case contractevents.EventTypeBurn:
				burnEvt := sacEvent.(*contractevents.BurnEvent)
				balanceChanges = append(balanceChanges, o.createSACBalanceChangeEntry(burnEvt.From, "", burnEvt.Amount, burnEvt.Asset, "burn"))
			}
		}
	}

	return balanceChanges, nil
}

func (o *LedgerOperation) filterEvents(diagnosticEvents []xdr.DiagnosticEvent) []xdr.ContractEvent {
	var filtered []xdr.ContractEvent
	for _, diagnosticEvent := range diagnosticEvents {
		if !diagnosticEvent.InSuccessfulContractCall || diagnosticEvent.Event.Type != xdr.ContractEventTypeContract {
			continue
		}
		filtered = append(filtered, diagnosticEvent.Event)
	}
	return filtered
}

// fromAccount   - strkey format of contract or address
// toAccount     - strkey format of contract or address, or nillable
// amountChanged - absolute value that asset balance changed
// asset         - the fully qualified issuer:code for asset that had balance change
// changeType    - the type of source sac event that triggered this change
//
// return        - a balance changed record expressed as map of key/value's
func (o *LedgerOperation) createSACBalanceChangeEntry(fromAccount string, toAccount string, amountChanged xdr.Int128Parts, asset xdr.Asset, changeType string) map[string]interface{} {
	balanceChange := map[string]interface{}{}

	if fromAccount != "" {
		balanceChange["from"] = fromAccount
	}
	if toAccount != "" {
		balanceChange["to"] = toAccount
	}

	balanceChange["type"] = changeType
	balanceChange["amount"] = amount.String128(amountChanged)
	o.addAssetDetails(balanceChange, asset, "")
	return balanceChange
}

// addAssetDetails sets the details for `a` on `result` using keys with `prefix`
func (o *LedgerOperation) addAssetDetails(result map[string]interface{}, a xdr.Asset, prefix string) error {
	var (
		assetType string
		code      string
		issuer    string
	)
	err := a.Extract(&assetType, &code, &issuer)
	if err != nil {
		err = fmt.Errorf("xdr.Asset.Extract error: %w", err)
		return err
	}

	prefix = o.FormatPrefix(prefix)
	result[prefix+"asset_type"] = assetType

	if a.Type == xdr.AssetTypeAssetTypeNative {
		result[prefix+"asset_id"] = int64(-5706705804583548011)
		return nil
	}

	result[prefix+"asset_code"] = code
	result[prefix+"asset_issuer"] = issuer
	result[prefix+"asset_id"] = o.FarmHashAsset(code, issuer, assetType)

	return nil
}

func (o *LedgerOperation) switchContractIdPreimageType(contractIdPreimage xdr.ContractIdPreimage) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	switch contractIdPreimage.Type {
	case xdr.ContractIdPreimageTypeContractIdPreimageFromAddress:
		fromAddress := contractIdPreimage.MustFromAddress()
		address, err := fromAddress.Address.String()
		if err != nil {
			return details, err
		}
		details["from"] = "address"
		details["address"] = address
	case xdr.ContractIdPreimageTypeContractIdPreimageFromAsset:
		details["from"] = "asset"
		details["asset"] = contractIdPreimage.MustFromAsset().StringCanonical()
		err := o.addAssetDetails(details, contractIdPreimage.MustFromAsset(), "")
		if err != nil {
			return details, err
		}
	default:
		panic(fmt.Errorf("unknown contract id type: %s", contractIdPreimage.Type))
	}

	return details, nil
}

func (o *LedgerOperation) ConvertStroopValueToReal(input int64) float64 {
	output, _ := big.NewRat(int64(input), int64(10000000)).Float64()
	return output
}

func (o *LedgerOperation) FormatPrefix(p string) string {
	if p != "" {
		p += "_"
	}
	return p
}

func (o *LedgerOperation) FarmHashAsset(assetCode, assetIssuer, assetType string) int64 {
	asset := fmt.Sprintf("%s%s%s", assetCode, assetIssuer, assetType)
	hash := farm.Fingerprint64([]byte(asset))

	return int64(hash)
}

// Path is a representation of an asset without an ID that forms part of a path in a path payment
type Path struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
}

func (o *LedgerOperation) TransformPath(initialPath []xdr.Asset) []Path {
	if len(initialPath) == 0 {
		return nil
	}
	var path = make([]Path, 0)
	for _, pathAsset := range initialPath {
		var assetType, code, issuer string
		err := pathAsset.Extract(&assetType, &code, &issuer)
		if err != nil {
			return nil
		}

		path = append(path, Path{
			AssetType:   assetType,
			AssetIssuer: issuer,
			AssetCode:   code,
		})
	}
	return path
}

type Price struct {
	Numerator   int32 `json:"n"`
	Denominator int32 `json:"d"`
}

func (o *LedgerOperation) PoolIDToString(id xdr.PoolId) string {
	return xdr.Hash(id).HexString()
}

type Claimant struct {
	Destination string             `json:"destination"`
	Predicate   xdr.ClaimPredicate `json:"predicate"`
}

func (o *LedgerOperation) TransformClaimants(claimants []xdr.Claimant) []Claimant {
	var transformed []Claimant
	for _, c := range claimants {
		switch c.Type {
		case 0:
			transformed = append(transformed, Claimant{
				Destination: c.V0.Destination.Address(),
				Predicate:   c.V0.Predicate,
			})
		}
	}
	return transformed
}

type SponsorshipOutput struct {
	Operation      xdr.Operation
	OperationIndex uint32
}

type LiquidityPoolDelta struct {
	ReserveA        xdr.Int64
	ReserveB        xdr.Int64
	TotalPoolShares xdr.Int64
}
