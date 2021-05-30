package horizonclient

import (
	"net/http"

	"github.com/stellar/go/support/errors"
)

// IsNotFoundError returns true if the error is a horizonclient.Error with
// a not_found problem indicating that the resource is not found on
// Horizon.
func IsNotFoundError(err error) bool {
	hErr := GetError(err)
	if hErr == nil {
		return false
	}

	return hErr.Problem.Type == "https://stellar.org/horizon-errors/not_found"
}

// IsHorizonAPITimeoutError returns true if the error is a horizonclient.Error with
// a timeout problem indicating that Horizon timed out.
func IsHorizonAPITimeoutError(err error) bool {
	hErr := GetError(err)
	if hErr == nil {
		return false
	}

	return hErr.Problem.Status == http.StatusGatewayTimeout
}

// GetError returns an error that can be interpreted as a horizon-specific
// error. If err cannot be interpreted as a horizon-specific error, a nil error
// is returned. The caller should still check whether err is nil.
func GetError(err error) *Error {
	var hErr *Error

	err = errors.Cause(err)
	switch e := err.(type) {
	case *Error:
		hErr = e
	case Error:
		hErr = &e
	}

	return hErr
}
