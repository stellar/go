package ingest

import (
	"fmt"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/guregu/null"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type LedgerOperation struct {
	OperationIndex  int32
	Operation       xdr.Operation
	Transaction     LedgerTransaction
	LedgerCloseMeta xdr.LedgerCloseMeta
}

func (o LedgerOperation) sourceAccountXDR() xdr.MuxedAccount {
	sourceAccount := o.Operation.SourceAccount
	if sourceAccount != nil {
		return *sourceAccount
	}

	return o.Transaction.Envelope.SourceAccount()
}

func (o LedgerOperation) SourceAccount() string {
	muxedAccount := o.sourceAccountXDR()

	providedID := muxedAccount.ToAccountId()
	pointerToID := &providedID
	return pointerToID.Address()
}

func (o LedgerOperation) Type() int32 {
	return int32(o.Operation.Body.Type)
}

func (o LedgerOperation) TypeString() string {
	return xdr.OperationTypeToStringMap[o.Type()]
}

func (o LedgerOperation) ID() int64 {
	//operationIndex needs +1 increment to stay in sync with ingest package
	return toid.New(int32(o.LedgerCloseMeta.LedgerSequence()), int32(o.Transaction.Index), o.OperationIndex+1).ToInt64()
}

func (o LedgerOperation) SourceAccountMuxed() null.String {
	var address null.String
	muxedAccount := o.sourceAccountXDR()

	if muxedAccount.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return null.StringFrom(muxedAccount.Address())
	}

	return address
}

func (o LedgerOperation) TransactionID() int64 {
	return o.Transaction.TransactionID()
}

func (o LedgerOperation) LedgerSequence() uint32 {
	return o.LedgerCloseMeta.LedgerSequence()
}

func (o LedgerOperation) LedgerClosedAt() time.Time {
	return o.LedgerCloseMeta.LedgerClosedAt()
}

func (o LedgerOperation) OperationResultCode() string {
	var operationResultCode string
	operationResults, ok := o.Transaction.Result.Result.OperationResults()
	if ok {
		operationResultCode = operationResults[o.OperationIndex].Code.String()
	}

	return operationResultCode
}

func (o LedgerOperation) OperationTraceCode() string {
	var operationTraceCode string

	operationResults, ok := o.Transaction.Result.Result.OperationResults()
	if ok {
		operationResultTr, ok := operationResults[o.OperationIndex].GetTr()
		if ok {
			operationTraceCode, err := operationResultTr.MapOperationResultTr()
			if err != nil {
				panic(err)
			}
			return operationTraceCode
		}
	}

	return operationTraceCode
}

func (o LedgerOperation) OperationDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}

	switch o.Operation.Body.Type {
	case xdr.OperationTypeCreateAccount:
		details, err := o.CreateAccountDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypePayment:
		details, err := o.PaymentDetails()
		if err != nil {
			return details, err
		}
	case xdr.OperationTypePathPaymentStrictReceive:
		details, err := o.PathPaymentStrictReceiveDetails()
		if err != nil {
			return details, err
		}
	// same for all other operations
	default:
		return details, fmt.Errorf("unknown operation type: %s", o.Operation.Body.Type.String())
	}

	return details, nil
}

func (o LedgerOperation) CreateAccountDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetCreateAccountOp()
	if !ok {
		return details, fmt.Errorf("could not access CreateAccount info for this operation (index %d)", o.OperationIndex)
	}

	if err := addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "funder"); err != nil {
		return details, err
	}
	details["account"] = op.Destination.Address()
	details["starting_balance"] = xdr.ConvertStroopValueToReal(op.StartingBalance)

	return details, nil
}

func addAccountAndMuxedAccountDetails(result map[string]interface{}, a xdr.MuxedAccount, prefix string) error {
	account_id := a.ToAccountId()
	result[prefix] = account_id.Address()
	prefix = formatPrefix(prefix)
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

func formatPrefix(p string) string {
	if p != "" {
		p += "_"
	}
	return p
}

func (o LedgerOperation) PaymentDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetPaymentOp()
	if !ok {
		return details, fmt.Errorf("could not access Payment info for this operation (index %d)", o.OperationIndex)
	}

	if err := addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "from"); err != nil {
		return details, err
	}
	if err := addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
		return details, err
	}
	details["amount"] = xdr.ConvertStroopValueToReal(op.Amount)
	if err := addAssetDetailsToOperationDetails(details, op.Asset, ""); err != nil {
		return details, err
	}

	return details, nil
}

func addAssetDetailsToOperationDetails(result map[string]interface{}, asset xdr.Asset, prefix string) error {
	var assetType, code, issuer string
	err := asset.Extract(&assetType, &code, &issuer)
	if err != nil {
		return err
	}

	prefix = formatPrefix(prefix)
	result[prefix+"asset_type"] = assetType

	if asset.Type == xdr.AssetTypeAssetTypeNative {
		result[prefix+"asset_id"] = int64(-5706705804583548011)
		return nil
	}

	result[prefix+"asset_code"] = code
	result[prefix+"asset_issuer"] = issuer
	result[prefix+"asset_id"] = farmHashAsset(code, issuer, assetType)

	return nil
}

func farmHashAsset(assetCode, assetIssuer, assetType string) int64 {
	asset := fmt.Sprintf("%s%s%s", assetCode, assetIssuer, assetType)
	hash := farm.Fingerprint64([]byte(asset))

	return int64(hash)
}

func (o LedgerOperation) PathPaymentStrictReceiveDetails() (map[string]interface{}, error) {
	details := map[string]interface{}{}
	op, ok := o.Operation.Body.GetPathPaymentStrictReceiveOp()
	if !ok {
		return details, fmt.Errorf("could not access PathPaymentStrictReceive info for this operation (index %d)", o.OperationIndex)
	}

	if err := addAccountAndMuxedAccountDetails(details, o.sourceAccountXDR(), "from"); err != nil {
		return details, err
	}
	if err := addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
		return details, err
	}
	details["amount"] = xdr.ConvertStroopValueToReal(op.DestAmount)
	details["source_amount"] = amount.String(0)
	details["source_max"] = xdr.ConvertStroopValueToReal(op.SendMax)
	if err := addAssetDetailsToOperationDetails(details, op.DestAsset, ""); err != nil {
		return details, err
	}
	if err := addAssetDetailsToOperationDetails(details, op.SendAsset, "source"); err != nil {
		return details, err
	}

	if o.Transaction.Result.Successful() {
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
		details["source_amount"] = xdr.ConvertStroopValueToReal(result.SendAmount())
	}

	details["path"] = transformPath(op.Path)
	return details, nil
}

// Path is a representation of an asset without an ID that forms part of a path in a path payment
type Path struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
}

func transformPath(initialPath []xdr.Asset) []Path {
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
