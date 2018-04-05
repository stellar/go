package bridge

import (
	"fmt"
	"net/http"
	"net/url"

	b "github.com/stellar/go/build"
	"github.com/stellar/go/protocols"
)

// PathPaymentOperationBody represents path_payment operation
type PathPaymentOperationBody struct {
	Source *string

	SendMax   string          `json:"send_max"`
	SendAsset protocols.Asset `json:"send_asset"`

	Destination       string
	DestinationAmount string          `json:"destination_amount"`
	DestinationAsset  protocols.Asset `json:"destination_asset"`

	Path []protocols.Asset
}

// ToValuesSpecial converts special values from http.Request to struct
func (op *PathPaymentOperationBody) FromRequestSpecial(r *http.Request, destination interface{}) error {
	var path []protocols.Asset

	for i := 0; i < 5; i++ {
		codeFieldName := fmt.Sprintf(pathCodeField, i)
		issuerFieldName := fmt.Sprintf(pathIssuerField, i)

		// If the element does not exist in PostForm break the loop
		if _, exists := r.PostForm[codeFieldName]; !exists {
			break
		}

		code := r.PostFormValue(codeFieldName)
		issuer := r.PostFormValue(issuerFieldName)

		if code == "" && issuer == "" {
			path = append(path, protocols.Asset{Type: "native"})
		} else {
			path = append(path, protocols.Asset{"credit", code, issuer})
		}
	}

	op.Path = path
	return nil
}

// ToValuesSpecial adds special values (not easily convertable) to given url.Values
func (op PathPaymentOperationBody) ToValuesSpecial(values url.Values) {
	for i, asset := range op.Path {
		values.Set(fmt.Sprintf(pathCodeField, i), asset.Code)
		values.Set(fmt.Sprintf(pathIssuerField, i), asset.Issuer)
	}
}

// ToTransactionMutator returns go-stellar-base TransactionMutator
func (op PathPaymentOperationBody) ToTransactionMutator() b.TransactionMutator {
	var path []b.Asset
	for _, pathAsset := range op.Path {
		path = append(path, pathAsset.ToBaseAsset())
	}

	mutators := []interface{}{
		b.Destination{op.Destination},
		b.PayWithPath{
			Asset:     op.SendAsset.ToBaseAsset(),
			MaxAmount: op.SendMax,
			Path:      path,
		},
	}

	if op.DestinationAsset.Code != "" && op.DestinationAsset.Issuer != "" {
		mutators = append(
			mutators,
			b.CreditAmount{
				op.DestinationAsset.Code,
				op.DestinationAsset.Issuer,
				op.DestinationAmount,
			},
		)
	} else {
		mutators = append(
			mutators,
			b.NativeAmount{op.DestinationAmount},
		)
	}

	if op.Source != nil {
		mutators = append(mutators, b.SourceAccount{*op.Source})
	}

	return b.Payment(mutators...)
}

// Validate validates if operation body is valid.
func (op PathPaymentOperationBody) Validate() error {
	panic("TODO")
	// if !protocols.IsValidAccountID(op.Destination) {
	// 	return protocols.NewInvalidParameterError("destination", op.Destination, "Destination must be a public key (starting with `G`).")
	// }

	// if !protocols.IsValidAmount(op.SendMax) {
	// 	return protocols.NewInvalidParameterError("send_max", op.SendMax, "Not a valid amount.")
	// }

	// if !protocols.IsValidAmount(op.DestinationAmount) {
	// 	return protocols.NewInvalidParameterError("destination_amount", op.DestinationAmount, "Not a valid amount.")
	// }

	// if !op.SendAsset.Validate() {
	// 	return protocols.NewInvalidParameterError("send_asset", op.SendAsset.String(), "Invalid asset.")
	// }

	// if !op.DestinationAsset.Validate() {
	// 	return protocols.NewInvalidParameterError("destination_asset", op.DestinationAsset.String(), "Invalid asset.")
	// }

	// if op.Source != nil && !protocols.IsValidAccountID(*op.Source) {
	// 	return protocols.NewInvalidParameterError("source", *op.Source, "Source must be a public key (starting with `G`).")
	// }

	// for i, asset := range op.Path {
	// 	if !asset.Validate() {
	// 		return protocols.NewInvalidParameterError("path["+strconv.Itoa(i)+"]", asset.String(), "Invalid asset.")
	// 	}
	// }

	// return nil
}
