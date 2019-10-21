package actions

import (
	"github.com/asaskevich/govalidator"

	"github.com/stellar/go/services/horizon/internal/assets"
	"github.com/stellar/go/xdr"
)

// Validateable allow structs to define their own custom validations.
type Validateable interface {
	Validate() error
}

func init() {
	govalidator.TagMap["accountID"] = govalidator.Validator(isAccountID)
	govalidator.TagMap["assetType"] = govalidator.Validator(isAssetType)
}

var customTagsErrorMessages = map[string]string{
	"accountID": "Account ID must start with `G` and contain 56 alphanum characters",
	"assetType": "Asset type must be native, credit_alphanum4 or credit_alphanum12",
}

func getErrorFieldMessage(err error) (string, string) {
	var field, message string

	switch err := err.(type) {
	case govalidator.Error:
		field = err.Name
		validator := err.Validator
		m, ok := customTagsErrorMessages[validator]
		// Give priority to inline custom error messages.
		// CustomErrorMessageExists when the validator is defined like:
		// `validatorName~custom message`
		if !ok || err.CustomErrorMessageExists {
			m = err.Err.Error()
		}
		message = m
	case govalidator.Errors:
		for _, item := range err.Errors() {
			field, message = getErrorFieldMessage(item)
			break
		}
	}

	return field, message
}

func isAssetType(str string) bool {
	if _, err := assets.Parse(str); err != nil {
		return false
	}

	return true
}

func isAccountID(str string) bool {
	if _, err := xdr.AddressToAccountId(str); err != nil {
		return false
	}

	return true
}
