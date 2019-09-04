package txnbuild

import (
	"fmt"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
)

// validateStellarPublicKey returns true if a public key is valid. Otherwise, it returns false.
// It is a wrapper around the IsValidEd25519PublicKey method of the strkey package.
func validateStellarPublicKey(publicKey string) bool {
	return strkey.IsValidEd25519PublicKey(publicKey)
}

// validateStellarAsset checks if the asset supplied is a valid stellar Asset. It returns an error if the asset is
// nil, has an invalid asset code or issuer.
func validateStellarAsset(asset Asset) error {
	if asset == nil {
		return errors.New("asset is required")
	}

	if asset.IsNative() {
		return nil
	}

	_, err := asset.GetType()
	if err != nil {
		return errors.New("asset code length must be between 1 and 12 characters")
	}

	if !validateStellarPublicKey(asset.GetIssuer()) {
		return errors.New("asset issuer is not a valid stellar public key")
	}

	return nil
}

// validateNumber checks if the provided value is a valid stellar amount, it returns an error if not.
// This is used to validate price and amount fields in structs.
func validateNumber(n interface{}) error {
	var stellarAmount int64
	// type switch can be extended to handle other types. Currently, the types for number values in the txnbuild
	// package are string or int64.
	switch value := n.(type) {
	case int64:
		stellarAmount = value
	case string:
		v, err := amount.ParseInt64(value)
		if err != nil {
			return err
		}
		stellarAmount = v
	default:
		return errors.New("validation failed, unknown type")
	}

	if stellarAmount < 0 {
		return errors.New("value should be positve or zero")
	}
	return nil
}

// validateAllowTrustAsset checks if the provided asset is valid for use in AllowTrust operation.
// It returns an error if the asset is invalid.
// The asset must be non native (XLM) with a valid asset code.
func validateAllowTrustAsset(asset Asset) error {
	if asset == nil {
		return errors.New("asset is required")
	}

	if asset.IsNative() {
		return errors.New("native (XLM) asset type is not allowed")
	}

	_, err := asset.GetType()
	if err != nil {
		return errors.Errorf("asset code length must be between 1 and 12 characters")
	}
	return nil
}

// validateChangeTrustAsset checks if the provided asset is valid for use in ChangeTrust operation.
// It returns an error if the asset is invalid.
// The asset must be non native (XLM) with a valid asset code and issuer.
func validateChangeTrustAsset(asset Asset) error {
	err := validateAllowTrustAsset(asset)
	if err != nil {
		return err
	}

	if !validateStellarPublicKey(asset.GetIssuer()) {
		return errors.New("asset issuer is not a valid stellar public key")
	}

	return nil
}

// validatePassiveOffer checks if the fields of a CreatePassiveOffer struct are valid.
// It checks that the buying and selling assets are valid stellar assets, and that amount and price are valid.
// It returns an error if any field is invalid.
func validatePassiveOffer(buying, selling Asset, offerAmount, price string) error {
	err := validateStellarAsset(buying)
	if err != nil {
		return NewValidationError("Buying", err.Error())
	}

	err = validateStellarAsset(selling)
	if err != nil {
		return NewValidationError("Selling", err.Error())
	}

	err = validateNumber(offerAmount)
	if err != nil {
		return NewValidationError("Amount", err.Error())
	}

	err = validateNumber(price)
	if err != nil {
		return NewValidationError("Price", err.Error())
	}

	return nil
}

// validateOffer checks if the fields of ManageBuyOffer or ManageSellOffer struct are valid.
// It checks that the buying and selling assets are valid stellar assets, and that amount, price and offerID
// are valid. It returns an error if any field is invalid.
func validateOffer(buying, selling Asset, offerAmount, price string, offerID int64) error {
	err := validatePassiveOffer(buying, selling, offerAmount, price)
	if err != nil {
		return err
	}

	err = validateNumber(offerID)
	if err != nil {
		return NewValidationError("OfferID", err.Error())
	}
	return nil
}

// ValidationError is a custom error struct that holds validation errors of txnbuild's operation structs.
type ValidationError struct {
	Field   string // Field is the struct field on which the validation error occured.
	Message string // Message is the validation error message.
}

// Error for ValidationError struct implements the error interface.
func (opError *ValidationError) Error() string {
	return fmt.Sprintf("Field: %s, Error: %s", opError.Field, opError.Message)
}

// NewValidationError creates a ValidationError struct with the provided field and message values.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
