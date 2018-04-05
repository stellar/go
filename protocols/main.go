package protocols

import (
	"net/http"
	"net/url"

	"github.com/stellar/go/support/errors"
)

var (
	// InternalServerError is an error response
	InternalServerError = &ErrorResponse{Code: "internal_server_error", Message: "Internal Server Error, please try again.", Status: http.StatusInternalServerError}
	// InvalidParameterError is an error response
	InvalidParameterError = &ErrorResponse{Code: "invalid_parameter", Message: "Invalid parameter.", Status: http.StatusBadRequest}

	// missingParameterError is an error response
	missingParameterError = &ErrorResponse{Code: "missing_parameter", Message: "Required parameter is missing.", Status: http.StatusBadRequest}
)

// SpecialValuesConvertable allows converting special values (not easily convertable):
// * from struct to url.Values
// * from http.Request to struct
type SpecialValuesConvertable interface {
	FromRequestSpecial(r *http.Request, destination interface{}) error
	ToValuesSpecial(values url.Values)
}

// ForwardDestination contains fields required to create forward federation request
type ForwardDestination struct {
	Domain string     `name:"domain"`
	Fields url.Values `name:"fields"`
}

// Response represents response that can be returned by a server
type Response interface {
	HTTPStatus() int
	Marshal() ([]byte, error)
}

// SuccessResponse can be embedded in success responses
type SuccessResponse struct{}

func (r *SuccessResponse) HTTPStatus() int {
	return http.StatusOK
}

// Write writes a response to the given http.ResponseWriter
func Write(w http.ResponseWriter, response Response) error {
	w.WriteHeader(response.HTTPStatus())
	body, err := response.Marshal()
	if err != nil {
		return errors.Wrap(err, "Error marshaling response")
	}
	w.Write(body)
	return nil
}

// ErrorResponse represents error response and implements server.Response and error interfaces
type ErrorResponse struct {
	// HTTP status code
	Status int `json:"-"`
	// Error status code
	Code string `json:"code"`
	// Error message that will be returned to API consumer
	Message string `json:"message"`
	// Additional information returned to API consumer
	MoreInfo string `json:"more_info,omitempty"`
	// Error data that will be returned to API consumer
	Data map[string]interface{} `json:"data,omitempty"`
}

// Asset represents asset
type Asset struct {
	Type   string `name:"asset_type" json:"type"`
	Code   string `name:"asset_code" json:"code"`
	Issuer string `name:"asset_issuer" json:"issuer"`
}
