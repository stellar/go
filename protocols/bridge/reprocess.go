package bridge

import (
	"encoding/json"
	"net/http"
)

// ReprocessRequest represents request made to /reprocess endpoint of bridge server
type ReprocessRequest struct {
	OperationID string `name:"operation_id" required:""`
	// Force is required for reprocessing successful payments. Please use with caution!
	Force bool `name:"force"`
}

// Validate validates if request fields are valid. Useful when checking if a request is correct.
func (request *ReprocessRequest) Validate() error {
	panic("TODO")
	// err := request.FormRequest.CheckRequired(request)
	// if err != nil {
	// 	return err
	// }

	// return nil
}

// ReprocessResponse represents a response returned by /reprocess endpoint
type ReprocessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (r ReprocessResponse) HTTPStatus() int {
	if r.Status == "ok" {
		return http.StatusOK
	} else {
		return http.StatusBadRequest
	}
}

func (r ReprocessResponse) Marshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
