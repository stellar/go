package compliance

import (
	"encoding/json"

	"github.com/stellar/go/services/internal/bridge-compliance-shared/http/helpers"
)

// ReceiveRequest represents request sent to /receive endpoint of compliance server
type ReceiveRequest struct {
	Memo string `name:"memo" required:""`
}

// Validate validates if request fields are valid. Useful when checking if a request is correct.
func (request *ReceiveRequest) Validate() error {
	panic("TODO")
	// err := request.formRequest.CheckRequired(request)
	// if err != nil {
	// 	return err
	// }
	// return nil
}

// ReceiveResponse represents response returned by /receive endpoint
type ReceiveResponse struct {
	helpers.SuccessResponse
	// The AuthData hash of this memo.
	Data string `json:"data"`
}

// Marshal marshals ReceiveResponse
func (response *ReceiveResponse) Marshal() ([]byte, error) {
	return json.MarshalIndent(response, "", "  ")
}
