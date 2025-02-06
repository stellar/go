package ingest

import (
	"encoding/base64"
	"fmt"
	"math/big"

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
	return muxedAccount.ToAccountId().Address()
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
	var err error

	operationResults, ok = o.Transaction.Result.Result.OperationResults()
	if ok {
		var operationResultTr xdr.OperationResultTr
		operationResultTr, ok = operationResults[o.OperationIndex].GetTr()
		if ok {
			operationTraceCode, err = operationResultTr.MapOperationResultTr()
			if err != nil {
				return "", err
			}
			return operationTraceCode, nil
		}
	}

	return operationTraceCode, nil
}

func (o *LedgerOperation) OperationDetails() (interface{}, error) {
	var err error
	var details interface{}

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
		details, err = o.SetTrustlineFlagsDetails()
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

func getMuxedAccountDetails(a xdr.MuxedAccount) (string, uint64, error) {
	var err error
	var muxedAccountAddress string
	var muxedAccountID uint64

	if a.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		muxedAccountAddress, err = a.GetAddress()
		if err != nil {
			return "", 0, err
		}
		muxedAccountID, err = a.GetId()
		if err != nil {
			return "", 0, err
		}
	}
	return muxedAccountAddress, muxedAccountID, nil
}

type LedgerKeyDetail struct {
	AccountID                string `json:"account_id"`
	ClaimableBalanceID       string `json:"claimable_balance_id"`
	DataAccountID            string `json:"data_account_id"`
	DataName                 string `json:"data_name"`
	OfferID                  int64  `json:"offer_id,string"`
	TrustlineAccountID       string `json:"trustline_account_id"`
	TrustlineLiquidityPoolID string `json:"trustline_liquidity_pool_id"`
	TrustlineAssetCode       string `json:"trustline_asset_code"`
	TrustlineAssetIssuer     string `json:"trustline_asset_issuer"`
	TrustlineAssetType       string `json:"trustline_asset_type"`
	LiquidityPoolID          string `json:"liquidity_pool_id"`
}

func addLedgerKeyToDetails(ledgerKey xdr.LedgerKey) (LedgerKeyDetail, error) {
	var err error
	var ledgerKeyDetail LedgerKeyDetail

	switch ledgerKey.Type {
	case xdr.LedgerEntryTypeAccount:
		ledgerKeyDetail.AccountID = ledgerKey.Account.AccountId.Address()
	case xdr.LedgerEntryTypeClaimableBalance:
		var marshalHex string
		marshalHex, err = xdr.MarshalHex(ledgerKey.ClaimableBalance.BalanceId)
		if err != nil {
			return LedgerKeyDetail{}, fmt.Errorf("in claimable balance: %w", err)
		}
		ledgerKeyDetail.ClaimableBalanceID = marshalHex
	case xdr.LedgerEntryTypeData:
		ledgerKeyDetail.DataAccountID = ledgerKey.Data.AccountId.Address()
		ledgerKeyDetail.DataName = string(ledgerKey.Data.DataName)
	case xdr.LedgerEntryTypeOffer:
		ledgerKeyDetail.OfferID = int64(ledgerKey.Offer.OfferId)
	case xdr.LedgerEntryTypeTrustline:
		ledgerKeyDetail.TrustlineAccountID = ledgerKey.TrustLine.AccountId.Address()
		if ledgerKey.TrustLine.Asset.Type == xdr.AssetTypeAssetTypePoolShare {
			ledgerKeyDetail.TrustlineLiquidityPoolID, err = PoolIDToString(*ledgerKey.TrustLine.Asset.LiquidityPoolId)
			if err != nil {
				return LedgerKeyDetail{}, err
			}
		} else {
			var assetCode, assetIssuer, assetType string
			err = ledgerKey.TrustLine.Asset.ToAsset().Extract(&assetType, &assetCode, &assetIssuer)
			if err != nil {
				return LedgerKeyDetail{}, err
			}

			ledgerKeyDetail.TrustlineAssetCode = assetCode
			ledgerKeyDetail.TrustlineAssetIssuer = assetIssuer
			ledgerKeyDetail.TrustlineAssetType = assetType
		}
	case xdr.LedgerEntryTypeLiquidityPool:
		ledgerKeyDetail.LiquidityPoolID, err = PoolIDToString(ledgerKey.LiquidityPool.LiquidityPoolId)
		if err != nil {
			return LedgerKeyDetail{}, err
		}
	}

	return ledgerKeyDetail, nil
}

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

func (o *LedgerOperation) parseAssetBalanceChangesFromContractEvents() ([]BalanceChangeDetail, error) {
	balanceChanges := []BalanceChangeDetail{}

	diagnosticEvents, err := o.Transaction.GetDiagnosticEvents()
	if err != nil {
		// this operation in this context must be an InvokeHostFunctionOp, therefore V3Meta should be present
		// as it's in same soroban model, so if any err, it's real,
		return nil, err
	}

	for _, contractEvent := range o.filterEvents(diagnosticEvents) {
		// Parse the xdr contract event to contractevents.StellarAssetContractEvent model

		var err error
		var balanceChangeDetail BalanceChangeDetail
		var sacEvent contractevents.StellarAssetContractEvent
		// has some convenience like to/from attributes are expressed in strkey format for accounts(G...) and contracts(C...)
		if sacEvent, err = contractevents.NewStellarAssetContractEvent(&contractEvent, o.NetworkPassphrase); err == nil {
			switch sacEvent.GetType() {
			case contractevents.EventTypeTransfer:
				transferEvt := sacEvent.(*contractevents.TransferEvent)
				balanceChangeDetail, err = createSACBalanceChangeEntry(transferEvt.From, transferEvt.To, transferEvt.Amount, transferEvt.Asset, "transfer")
				if err != nil {
					return []BalanceChangeDetail{}, err
				}

				balanceChanges = append(balanceChanges, balanceChangeDetail)
			case contractevents.EventTypeMint:
				mintEvt := sacEvent.(*contractevents.MintEvent)
				balanceChangeDetail, err = createSACBalanceChangeEntry("", mintEvt.To, mintEvt.Amount, mintEvt.Asset, "mint")
				if err != nil {
					return []BalanceChangeDetail{}, err
				}

				balanceChanges = append(balanceChanges, balanceChangeDetail)
			case contractevents.EventTypeClawback:
				clawbackEvt := sacEvent.(*contractevents.ClawbackEvent)
				balanceChangeDetail, err = createSACBalanceChangeEntry(clawbackEvt.From, "", clawbackEvt.Amount, clawbackEvt.Asset, "clawback")
				if err != nil {
					return []BalanceChangeDetail{}, err
				}

				balanceChanges = append(balanceChanges, balanceChangeDetail)
			case contractevents.EventTypeBurn:
				burnEvt := sacEvent.(*contractevents.BurnEvent)
				balanceChangeDetail, err = createSACBalanceChangeEntry(burnEvt.From, "", burnEvt.Amount, burnEvt.Asset, "burn")
				if err != nil {
					return []BalanceChangeDetail{}, err
				}

				balanceChanges = append(balanceChanges, balanceChangeDetail)
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

type BalanceChangeDetail struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type"`
	Amount      string `json:"amount"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
}

// fromAccount   - strkey format of contract or address
// toAccount     - strkey format of contract or address, or nillable
// amountChanged - absolute value that asset balance changed
// asset         - the fully qualified issuer:code for asset that had balance change
// changeType    - the type of source sac event that triggered this change
//
// return        - a balance changed record expressed as map of key/value's
func createSACBalanceChangeEntry(fromAccount string, toAccount string, amountChanged xdr.Int128Parts, asset xdr.Asset, changeType string) (BalanceChangeDetail, error) {
	balanceChangeDetail := BalanceChangeDetail{
		Type:   changeType,
		Amount: amount.String128(amountChanged),
	}

	if fromAccount != "" {
		balanceChangeDetail.From = fromAccount
	}
	if toAccount != "" {
		balanceChangeDetail.To = toAccount
	}

	var assetCode, assetIssuer, assetType string
	err := asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return BalanceChangeDetail{}, err
	}

	return balanceChangeDetail, nil
}

type PreImageDetails struct {
	From        string `json:"from"`
	Address     string `json:"address"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
}

func switchContractIdPreimageType(contractIdPreimage xdr.ContractIdPreimage) (PreImageDetails, error) {
	switch contractIdPreimage.Type {
	case xdr.ContractIdPreimageTypeContractIdPreimageFromAddress:
		fromAddress := contractIdPreimage.MustFromAddress()
		address, err := fromAddress.Address.String()
		if err != nil {
			return PreImageDetails{}, err
		}
		return PreImageDetails{
			From:    "address",
			Address: address,
		}, nil
	case xdr.ContractIdPreimageTypeContractIdPreimageFromAsset:
		var assetCode, assetIssuer, assetType string
		contractIdPreimage.MustFromAsset().Extract(&assetType, &assetCode, &assetIssuer)

		return PreImageDetails{
			From:        "asset",
			AssetCode:   assetCode,
			AssetIssuer: assetIssuer,
			AssetType:   assetType,
		}, nil

	default:
		return PreImageDetails{}, fmt.Errorf("unknown contract id type: %s", contractIdPreimage.Type)
	}
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

func PoolIDToString(id xdr.PoolId) (string, error) {
	return xdr.MarshalBase64(id)
}

type Claimant struct {
	Destination string             `json:"destination"`
	Predicate   xdr.ClaimPredicate `json:"predicate"`
}

func transformClaimants(claimants []xdr.Claimant) []Claimant {
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
