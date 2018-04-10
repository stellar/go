package bridge

import (
	"net/http"

	"github.com/stellar/go/services/internal/bridge-compliance-shared/http/helpers"
)

var (
	// AllowTrustMalformed is an error response
	AllowTrustMalformed = &helpers.ErrorResponse{Code: "allow_trust_malformed", Message: "Asset name is malformed.", Status: http.StatusBadRequest}
	// AllowTrustNoTrustline is an error response
	AllowTrustNoTrustline = &helpers.ErrorResponse{Code: "allow_trust_no_trustline", Message: "Trustor does not have a trustline yet.", Status: http.StatusBadRequest}
	// AllowTrustTrustNotRequired is an error response
	AllowTrustTrustNotRequired = &helpers.ErrorResponse{Code: "allow_trust_trust_not_required", Message: "Authorizing account does not require allowing trust. Set AUTH_REQUIRED_FLAG on your account to use this feature.", Status: http.StatusBadRequest}
	// AllowTrustCantRevoke is an error response
	AllowTrustCantRevoke = &helpers.ErrorResponse{Code: "allow_trust_cant_revoke", Message: "Authorizing account has AUTH_REVOCABLE_FLAG set. Can't revoke the trustline.", Status: http.StatusBadRequest}
)

// AuthorizeRequest represents request made to /authorize endpoint of bridge server
type AuthorizeRequest struct {
	AccountID string `name:"account_id" required:""`
	AssetCode string `name:"asset_code" required:""`
}

// Validate validates if request fields are valid. Useful when checking if a request is correct.
func (request *AuthorizeRequest) Validate( /*allowedAssets []config.Asset, */ issuingAccountID string) error {
	panic("TODO")
	// err := request.FormRequest.CheckRequired(request)
	// if err != nil {
	// 	return err
	// }

	// if !helpers.IsValidAccountID(request.AccountID) {
	// 	return helpers.NewInvalidParameterError("account_id", request.AccountID, "Account ID must start with `G`.")
	// }

	// // Is asset allowed?
	// allowed := false
	// for _, asset := range allowedAssets {
	// 	if asset.Code == request.AssetCode && asset.Issuer == issuingAccountID {
	// 		allowed = true
	// 		break
	// 	}
	// }

	// if !allowed {
	// 	return helpers.NewInvalidParameterError("asset_code", request.AssetCode, "Asset code not allowed.")
	// }

	// return nil
}
