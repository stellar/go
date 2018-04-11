package bridge

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
